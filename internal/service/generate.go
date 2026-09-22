package service

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/natdanai/quizme/internal/ai"
	"github.com/natdanai/quizme/internal/model"
	"github.com/natdanai/quizme/internal/repository"
)

// AIGenerator is the minimal seam the generation flow needs from the AI client.
type AIGenerator interface {
	Generate(ctx context.Context, n int, topics []string) ([]ai.GeneratedQuestion, int, error)
}

// GenerateResult is the payload passed to the notifier on terminal states.
type GenerateResult struct {
	BatchDate     time.Time
	Status        string
	QuestionCount int
	Topics        []string
	Error         error
}

// Notifier receives a terminal-state payload. Implementations must not block
// the caller; the service calls them in a goroutine with panic recovery.
type Notifier interface {
	Notify(ctx context.Context, payload GenerateResult) error
}

// Repository is the minimal surface the service needs from the data layer.
// It is satisfied by *repository.Repo and by test doubles.
type Repository interface {
	GetOrCreateBatch(ctx context.Context, batchDate time.Time) (model.QuizBatch, bool, error)
	GetBatchByDate(ctx context.Context, batchDate time.Time) (model.QuizBatch, error)
	LatestSuccessfulBatch(ctx context.Context) (model.QuizBatch, error)
	GetQuestion(ctx context.Context, id int) (model.Question, error)
	GetQuestionsByBatch(ctx context.Context, batchID int) ([]model.Question, error)
	UpdateBatchStatus(ctx context.Context, id int, status string) error
	ListTopics(ctx context.Context) ([]model.Topic, error)
	CreateTopic(ctx context.Context, name string, weight int) (model.Topic, error)
	GetStats(ctx context.Context) ([]model.UserTopicStats, error)
	SubmitAnswer(ctx context.Context, questionID int, selectedOption string, isCorrect bool) error
	InsertQuestions(ctx context.Context, batchID int, questions []model.Question) error
	InsertGenerationLog(ctx context.Context, log model.GenerationLog) (model.GenerationLog, error)
}

// Service implements the generation and quiz business logic.
type Service struct {
	Repo     Repository
	AI       AIGenerator
	Notifier Notifier
	Now      func() time.Time
	Rand     *rand.Rand
}

// New creates a service with real time/random sources.
func New(repo *repository.Repo, ai AIGenerator, notifier Notifier) *Service {
	return &Service{
		Repo:     repo,
		AI:       ai,
		Notifier: notifier,
		Now:      time.Now,
		Rand:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// Generate creates today's batch, selects topics adaptively, calls the AI with
// typed retry budgets, inserts validated questions, and logs the result.
//
// It is idempotent: rerunning for a date that already succeeded is a no-op; a
// failed date is reset to pending and retried. A pending batch that already has
// questions is finalized as success to avoid duplicate generation.
func (s *Service) Generate(ctx context.Context) error {
	today := s.Now().UTC().Truncate(24 * time.Hour)

	batch, created, err := s.Repo.GetOrCreateBatch(ctx, today)
	if err != nil {
		return err
	}

	if !created {
		switch batch.Status {
		case "success":
			return nil
		case "pending":
			existing, err := s.Repo.GetQuestionsByBatch(ctx, batch.ID)
			if err != nil {
				return err
			}
			if len(existing) > 0 {
				// Previous run wrote questions but crashed before marking success.
				_ = s.Repo.UpdateBatchStatus(ctx, batch.ID, "success")
				return nil
			}
		case "failed":
			if err := s.Repo.UpdateBatchStatus(ctx, batch.ID, "pending"); err != nil {
				return err
			}
			batch.Status = "pending"
		}
	}

	topics, err := s.Repo.ListTopics(ctx)
	if err != nil {
		return err
	}
	if len(topics) == 0 {
		return s.failBatch(ctx, batch, 0, errors.New("no topics configured"))
	}

	stats, err := s.Repo.GetStats(ctx)
	if err != nil {
		return err
	}

	selected := SelectTopics(5, topics, stats, s.Rand)
	if len(selected) != 5 {
		return s.failBatch(ctx, batch, 0, fmt.Errorf("selected %d topics, want 5", len(selected)))
	}

	qs, tokens, err := s.generateWithRetry(ctx, 5, selected)
	if err != nil {
		return s.failBatch(ctx, batch, tokens, err)
	}

	questions, err := toQuestions(qs, topics)
	if err != nil {
		return s.failBatch(ctx, batch, tokens, err)
	}

	return s.succeedBatch(ctx, batch, questions, tokens, selected)
}

func (s *Service) generateWithRetry(ctx context.Context, n int, topics []string) ([]ai.GeneratedQuestion, int, error) {
	const maxAttempts = 3
	var (
		lastErr error
		tokens  int
	)
	for attempt := 0; attempt < maxAttempts; attempt++ {
		qs, t, err := s.AI.Generate(ctx, n, topics)
		if err == nil {
			return qs, t, nil
		}
		lastErr = err
		tokens = t
		if ai.IsValidationError(err) {
			if attempt == 0 {
				continue
			}
			break
		}
		if ai.IsAIError(err) {
			if attempt < maxAttempts-1 {
				continue
			}
			break
		}
		break
	}
	return nil, tokens, lastErr
}

func (s *Service) succeedBatch(ctx context.Context, batch model.QuizBatch, questions []model.Question, tokens int, selected []string) error {
	if err := s.Repo.InsertQuestions(ctx, batch.ID, questions); err != nil {
		return s.failBatch(ctx, batch, tokens, err)
	}
	if err := s.Repo.UpdateBatchStatus(ctx, batch.ID, "success"); err != nil {
		return s.failBatch(ctx, batch, tokens, err)
	}
	s.log(ctx, batch.ID, "success", nil, tokens)
	s.notify(ctx, GenerateResult{
		BatchDate:     batch.BatchDate,
		Status:        "success",
		QuestionCount: len(questions),
		Topics:        selected,
	})
	return nil
}

func (s *Service) failBatch(ctx context.Context, batch model.QuizBatch, tokens int, err error) error {
	statusErr := s.Repo.UpdateBatchStatus(ctx, batch.ID, "failed")
	msg := err.Error()
	s.log(ctx, batch.ID, "failed", &msg, tokens)
	s.notify(ctx, GenerateResult{
		BatchDate: batch.BatchDate,
		Status:    "failed",
		Error:     err,
	})
	if statusErr != nil {
		return errors.Join(err, statusErr)
	}
	return err
}

func (s *Service) log(ctx context.Context, batchID int, status string, msg *string, tokens int) {
	tok := &tokens
	if tokens == 0 {
		tok = nil
	}
	_, _ = s.Repo.InsertGenerationLog(ctx, model.GenerationLog{
		BatchID:      &batchID,
		Status:       status,
		ErrorMessage: msg,
		TokensUsed:   tok,
	})
}

func normalizeTopic(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}

func (s *Service) notify(ctx context.Context, r GenerateResult) {
	if s.Notifier == nil {
		return
	}
	go func() {
		defer func() {
			if rec := recover(); rec != nil {
				// Fire-and-forget: never propagate notifier panics.
			}
		}()
		_ = s.Notifier.Notify(ctx, r)
	}()
}

func toQuestions(generated []ai.GeneratedQuestion, topics []model.Topic) ([]model.Question, error) {
	byName := make(map[string]int, len(topics))
	for _, t := range topics {
		byName[normalizeTopic(t.Name)] = t.ID
	}

	questions := make([]model.Question, len(generated))
	for i, g := range generated {
		topicID, ok := byName[normalizeTopic(g.Topic)]
		if !ok {
			return nil, fmt.Errorf("unknown topic %q", g.Topic)
		}
		exp := g.Explanation
		questions[i] = model.Question{
			TopicID:       topicID,
			Prompt:        g.Prompt,
			Options:       g.Options,
			CorrectOption: g.CorrectOption,
			Explanation:   &exp,
			Source:        "ai_generated",
		}
	}
	return questions, nil
}
