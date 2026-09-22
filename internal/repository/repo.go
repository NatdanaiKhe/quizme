package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/natdanai/quizme/internal/model"
)

// Repo provides Postgres access for every §6 table.
type Repo struct {
	pool *pgxpool.Pool
}

// New opens a connection pool with sane defaults for a single-user MVP.
func New(ctx context.Context, databaseURL string) (*Repo, error) {
	cfg, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse database url: %w", err)
	}
	cfg.MaxConns = 5

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("open pool: %w", err)
	}

	return &Repo{pool: pool}, nil
}

// Close releases the pool.
func (r *Repo) Close() {
	r.pool.Close()
}

// Pool exposes the underlying connection pool for test helpers.
func (r *Repo) Pool() *pgxpool.Pool {
	return r.pool
}

// ListTopics returns every topic.
func (r *Repo) ListTopics(ctx context.Context) ([]model.Topic, error) {
	rows, err := r.pool.Query(ctx, `SELECT id, name, weight, created_at FROM topics ORDER BY id`)
	if err != nil {
		return nil, fmt.Errorf("list topics: %w", err)
	}
	defer rows.Close()

	var topics []model.Topic
	for rows.Next() {
		var t model.Topic
		if err := rows.Scan(&t.ID, &t.Name, &t.Weight, &t.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan topic: %w", err)
		}
		topics = append(topics, t)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list topics rows: %w", err)
	}
	return topics, nil
}

// CreateTopic inserts a new topic.
func (r *Repo) CreateTopic(ctx context.Context, name string, weight int) (model.Topic, error) {
	var t model.Topic
	if err := r.pool.QueryRow(ctx,
		`INSERT INTO topics (name, weight) VALUES ($1, $2) RETURNING id, name, weight, created_at`,
		name, weight,
	).Scan(&t.ID, &t.Name, &t.Weight, &t.CreatedAt); err != nil {
		return model.Topic{}, fmt.Errorf("create topic %s: %w", name, err)
	}
	return t, nil
}

// GetOrCreateBatch returns the batch for date, creating it when absent.
// The unique constraint on batch_date makes the insert-on-conflict safe.
func (r *Repo) GetOrCreateBatch(ctx context.Context, batchDate time.Time) (model.QuizBatch, bool, error) {
	var b model.QuizBatch
	err := r.pool.QueryRow(ctx,
		`INSERT INTO quiz_batches (batch_date, status) VALUES ($1, 'pending')
		 ON CONFLICT (batch_date) DO NOTHING
		 RETURNING id, batch_date, status, created_at`,
		batchDate,
	).Scan(&b.ID, &b.BatchDate, &b.Status, &b.CreatedAt)
	if err == nil {
		return b, true, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return model.QuizBatch{}, false, fmt.Errorf("insert batch %s: %w", batchDate.Format(time.DateOnly), err)
	}

	if err := r.pool.QueryRow(ctx,
		`SELECT id, batch_date, status, created_at FROM quiz_batches WHERE batch_date = $1`,
		batchDate,
	).Scan(&b.ID, &b.BatchDate, &b.Status, &b.CreatedAt); err != nil {
		return model.QuizBatch{}, false, fmt.Errorf("select batch %s: %w", batchDate.Format(time.DateOnly), err)
	}
	return b, false, nil
}

// GetBatchByDate returns the batch for a date, or pgx.ErrNoRows if absent.
// Read-only: it never creates a row, so it is safe to call from GET paths.
func (r *Repo) GetBatchByDate(ctx context.Context, batchDate time.Time) (model.QuizBatch, error) {
	var b model.QuizBatch
	err := r.pool.QueryRow(ctx,
		`SELECT id, batch_date, status, created_at FROM quiz_batches WHERE batch_date = $1`,
		batchDate,
	).Scan(&b.ID, &b.BatchDate, &b.Status, &b.CreatedAt)
	if err != nil {
		return model.QuizBatch{}, fmt.Errorf("select batch %s: %w", batchDate.Format(time.DateOnly), err)
	}
	return b, nil
}

// UpdateBatchStatus sets a batch's status.
func (r *Repo) UpdateBatchStatus(ctx context.Context, id int, status string) error {
	if _, err := r.pool.Exec(ctx,
		`UPDATE quiz_batches SET status = $1 WHERE id = $2`,
		status, id,
	); err != nil {
		return fmt.Errorf("update batch %d status %s: %w", id, status, err)
	}
	return nil
}

// LatestSuccessfulBatch returns the most recent successful batch.
func (r *Repo) LatestSuccessfulBatch(ctx context.Context) (model.QuizBatch, error) {
	var b model.QuizBatch
	err := r.pool.QueryRow(ctx,
		`SELECT id, batch_date, status, created_at FROM quiz_batches
		 WHERE status = 'success' ORDER BY batch_date DESC LIMIT 1`,
	).Scan(&b.ID, &b.BatchDate, &b.Status, &b.CreatedAt)
	if err != nil {
		return model.QuizBatch{}, fmt.Errorf("latest successful batch: %w", err)
	}
	return b, nil
}

// InsertQuestions stores a batch's questions in a single multi-row INSERT.
func (r *Repo) InsertQuestions(ctx context.Context, batchID int, questions []model.Question) error {
	if len(questions) == 0 {
		return nil
	}

	sql := `INSERT INTO questions (batch_id, topic_id, prompt, options, correct_option, explanation, source) VALUES `
	args := make([]any, 0, len(questions)*7)
	placeholder := 1
	for i, q := range questions {
		if i > 0 {
			sql += ", "
		}
		sql += fmt.Sprintf("($%d, $%d, $%d, $%d, $%d, $%d, $%d)",
			placeholder, placeholder+1, placeholder+2, placeholder+3, placeholder+4, placeholder+5, placeholder+6)
		optionsJSON, err := json.Marshal(q.Options)
		if err != nil {
			return fmt.Errorf("marshal options for question %d: %w", i, err)
		}
		args = append(args, batchID, q.TopicID, q.Prompt, optionsJSON, q.CorrectOption, q.Explanation, q.Source)
		placeholder += 7
	}

	if _, err := r.pool.Exec(ctx, sql, args...); err != nil {
		return fmt.Errorf("insert questions for batch %d: %w", batchID, err)
	}
	return nil
}

// GetQuestion returns a single question by id, or pgx.ErrNoRows if absent.
func (r *Repo) GetQuestion(ctx context.Context, id int) (model.Question, error) {
	q, err := scanQuestion(r.pool.QueryRow(ctx,
		`SELECT id, batch_id, topic_id, prompt, options, correct_option, explanation, source, created_at
		 FROM questions WHERE id = $1`,
		id,
	))
	if err != nil {
		return model.Question{}, fmt.Errorf("select question %d: %w", id, err)
	}
	return q, nil
}

// GetQuestionsByBatch returns every question for a batch.
func (r *Repo) GetQuestionsByBatch(ctx context.Context, batchID int) ([]model.Question, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, batch_id, topic_id, prompt, options, correct_option, explanation, source, created_at
		 FROM questions WHERE batch_id = $1 ORDER BY id`,
		batchID,
	)
	if err != nil {
		return nil, fmt.Errorf("select questions for batch %d: %w", batchID, err)
	}
	defer rows.Close()

	var questions []model.Question
	for rows.Next() {
		q, err := scanQuestion(rows)
		if err != nil {
			return nil, fmt.Errorf("scan question for batch %d: %w", batchID, err)
		}
		questions = append(questions, q)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("select questions rows for batch %d: %w", batchID, err)
	}
	return questions, nil
}

// SubmitAnswer records an answer and updates per-topic stats atomically.
func (r *Repo) SubmitAnswer(ctx context.Context, questionID int, selectedOption string, isCorrect bool) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin answer tx: %w", err)
	}
	defer tx.Rollback(ctx)

	var topicID int
	if err := tx.QueryRow(ctx,
		`SELECT topic_id FROM questions WHERE id = $1`,
		questionID,
	).Scan(&topicID); err != nil {
		return fmt.Errorf("lookup question %d: %w", questionID, err)
	}

	if _, err := tx.Exec(ctx,
		`INSERT INTO user_answers (question_id, selected_option, is_correct) VALUES ($1, $2, $3)`,
		questionID, selectedOption, isCorrect,
	); err != nil {
		return fmt.Errorf("insert answer for question %d: %w", questionID, err)
	}

	const upsertStats = `
		INSERT INTO user_topic_stats (topic_id, correct_count, wrong_count, last_practiced_at, accuracy_rate)
		VALUES (
			$1,
			CASE WHEN $2 THEN 1 ELSE 0 END,
			CASE WHEN $2 THEN 0 ELSE 1 END,
			now(),
			CASE WHEN $2 THEN 100.00 ELSE 0.00 END
		)
		ON CONFLICT (topic_id) DO UPDATE SET
			correct_count = user_topic_stats.correct_count + EXCLUDED.correct_count,
			wrong_count = user_topic_stats.wrong_count + EXCLUDED.wrong_count,
			last_practiced_at = EXCLUDED.last_practiced_at,
			accuracy_rate = round(
				((user_topic_stats.correct_count + EXCLUDED.correct_count)::numeric /
				 NULLIF(user_topic_stats.correct_count + user_topic_stats.wrong_count + EXCLUDED.correct_count + EXCLUDED.wrong_count, 0)) * 100,
			2)
		RETURNING topic_id, correct_count, wrong_count, last_practiced_at, accuracy_rate::float8`

	var stats model.UserTopicStats
	if err := tx.QueryRow(ctx, upsertStats, topicID, isCorrect).Scan(
		&stats.TopicID, &stats.CorrectCount, &stats.WrongCount, &stats.LastPracticedAt, &stats.AccuracyRate,
	); err != nil {
		return fmt.Errorf("upsert stats for topic %d: %w", topicID, err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit answer tx: %w", err)
	}
	return nil
}

// GetStats returns every per-topic stats row.
func (r *Repo) GetStats(ctx context.Context) ([]model.UserTopicStats, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT topic_id, correct_count, wrong_count, last_practiced_at, accuracy_rate::float8
		 FROM user_topic_stats ORDER BY topic_id`,
	)
	if err != nil {
		return nil, fmt.Errorf("select stats: %w", err)
	}
	defer rows.Close()

	var stats []model.UserTopicStats
	for rows.Next() {
		var s model.UserTopicStats
		if err := rows.Scan(&s.TopicID, &s.CorrectCount, &s.WrongCount, &s.LastPracticedAt, &s.AccuracyRate); err != nil {
			return nil, fmt.Errorf("scan stats: %w", err)
		}
		stats = append(stats, s)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("select stats rows: %w", err)
	}
	return stats, nil
}

// InsertGenerationLog records a generation run.
func (r *Repo) InsertGenerationLog(ctx context.Context, log model.GenerationLog) (model.GenerationLog, error) {
	var inserted model.GenerationLog
	if err := r.pool.QueryRow(ctx,
		`INSERT INTO generation_logs (batch_id, status, error_message, tokens_used)
		 VALUES ($1, $2, $3, $4) RETURNING id, batch_id, status, error_message, tokens_used, created_at`,
		log.BatchID, log.Status, log.ErrorMessage, log.TokensUsed,
	).Scan(&inserted.ID, &inserted.BatchID, &inserted.Status, &inserted.ErrorMessage, &inserted.TokensUsed, &inserted.CreatedAt); err != nil {
		return model.GenerationLog{}, fmt.Errorf("insert generation log: %w", err)
	}
	return inserted, nil
}

// RecentGenerationLogs returns the most recent n log entries.
func (r *Repo) RecentGenerationLogs(ctx context.Context, n int) ([]model.GenerationLog, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, batch_id, status, error_message, tokens_used, created_at
		 FROM generation_logs ORDER BY created_at DESC LIMIT $1`,
		n,
	)
	if err != nil {
		return nil, fmt.Errorf("select recent generation logs: %w", err)
	}
	defer rows.Close()

	var logs []model.GenerationLog
	for rows.Next() {
		var l model.GenerationLog
		if err := rows.Scan(&l.ID, &l.BatchID, &l.Status, &l.ErrorMessage, &l.TokensUsed, &l.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan generation log: %w", err)
		}
		logs = append(logs, l)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("select generation logs rows: %w", err)
	}
	return logs, nil
}

func scanQuestion(row pgx.Row) (model.Question, error) {
	var q model.Question
	var optionsJSON []byte
	if err := row.Scan(
		&q.ID, &q.BatchID, &q.TopicID, &q.Prompt, &optionsJSON, &q.CorrectOption,
		&q.Explanation, &q.Source, &q.CreatedAt,
	); err != nil {
		return model.Question{}, err
	}
	if err := json.Unmarshal(optionsJSON, &q.Options); err != nil {
		return model.Question{}, fmt.Errorf("unmarshal options: %w", err)
	}
	return q, nil
}
