package service

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/natdanai/quizme/internal/model"
)

type answerRepo struct {
	Repository
	q   model.Question
	err error
}

func (a answerRepo) GetQuestion(ctx context.Context, id int) (model.Question, error) {
	if a.err != nil {
		return model.Question{}, a.err
	}
	return a.q, nil
}

func (a answerRepo) SubmitAnswer(ctx context.Context, questionID int, selectedOption string, isCorrect bool) error {
	return nil
}

func TestAnswerCorrect(t *testing.T) {
	exp := "because"
	svc := &Service{
		Repo: answerRepo{q: model.Question{ID: 1, CorrectOption: "b", Explanation: &exp}},
	}

	correct, explanation, err := svc.Answer(context.Background(), 1, "b")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !correct {
		t.Fatal("want correct=true")
	}
	if explanation == nil || *explanation != exp {
		t.Fatalf("want explanation %q, got %v", exp, explanation)
	}
}

func TestAnswerWrong(t *testing.T) {
	exp := "nope"
	svc := &Service{
		Repo: answerRepo{q: model.Question{ID: 2, CorrectOption: "c", Explanation: &exp}},
	}

	correct, explanation, err := svc.Answer(context.Background(), 2, "a")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if correct {
		t.Fatal("want correct=false")
	}
	if explanation == nil || *explanation != exp {
		t.Fatalf("want explanation %q, got %v", exp, explanation)
	}
}

func TestAnswerNotFound(t *testing.T) {
	svc := &Service{
		Repo: answerRepo{err: pgx.ErrNoRows},
	}

	_, _, err := svc.Answer(context.Background(), 999, "a")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("want ErrNotFound, got %v", err)
	}
}
