package services

import (
	"context"
	"testing"

	"github.com/axfinn/todoIngPlus/backend-go/internal/models"
	"github.com/axfinn/todoIngPlus/backend-go/internal/repository/mocks"
)

func TestTaskServiceCreate(t *testing.T) {
	mockRepo := &mocks.TaskRepositoryMock{InsertFn: func(ctx context.Context, t *models.Task) error { t.ID = "gen123"; return nil }}
	svc := &TaskService{repo: mockRepo}
	res, err := svc.Create(context.Background(), "u1", models.Task{Title: "A"})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if res.ID != "gen123" {
		t.Fatalf("expected id gen123 got %s", res.ID)
	}
}
