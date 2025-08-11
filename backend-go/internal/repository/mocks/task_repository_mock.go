package mocks

import (
	"context"

	"github.com/axfinn/todoIngPlus/backend-go/internal/models"
	"github.com/axfinn/todoIngPlus/backend-go/internal/repository"
	"go.mongodb.org/mongo-driver/bson"
)

type TaskRepositoryMock struct {
	InsertFn       func(ctx context.Context, t *models.Task) error
	ListFn         func(ctx context.Context, userID string, status string, page, limit int64) ([]models.Task, int64, error)
	FindByIDFn     func(ctx context.Context, userID, id string) (*models.Task, error)
	UpdatePartialFn func(ctx context.Context, userID, id string, set bson.M) (*models.Task, error)
	DeleteFn       func(ctx context.Context, userID, id string) error
}

var _ repository.TaskRepository = (*TaskRepositoryMock)(nil)

func (m *TaskRepositoryMock) Insert(ctx context.Context, t *models.Task) error { return m.callInsert(ctx, t) }
func (m *TaskRepositoryMock) List(ctx context.Context, userID string, status string, page, limit int64) ([]models.Task, int64, error) { return m.callList(ctx, userID, status, page, limit) }
func (m *TaskRepositoryMock) FindByID(ctx context.Context, userID, id string) (*models.Task, error) { return m.callFindByID(ctx, userID, id) }
func (m *TaskRepositoryMock) UpdatePartial(ctx context.Context, userID, id string, set bson.M) (*models.Task, error) { return m.callUpdatePartial(ctx, userID, id, set) }
func (m *TaskRepositoryMock) Delete(ctx context.Context, userID, id string) error { return m.callDelete(ctx, userID, id) }

// internal wrappers with nil checks
func (m *TaskRepositoryMock) callInsert(ctx context.Context, t *models.Task) error { if m.InsertFn!=nil { return m.InsertFn(ctx,t) }; return nil }
func (m *TaskRepositoryMock) callList(ctx context.Context, userID, status string, page, limit int64) ([]models.Task,int64,error) { if m.ListFn!=nil { return m.ListFn(ctx,userID,status,page,limit) }; return nil,0,nil }
func (m *TaskRepositoryMock) callFindByID(ctx context.Context, userID,id string) (*models.Task,error) { if m.FindByIDFn!=nil { return m.FindByIDFn(ctx,userID,id) }; return nil,nil }
func (m *TaskRepositoryMock) callUpdatePartial(ctx context.Context, userID,id string, set bson.M) (*models.Task,error) { if m.UpdatePartialFn!=nil { return m.UpdatePartialFn(ctx,userID,id,set) }; return nil,nil }
func (m *TaskRepositoryMock) callDelete(ctx context.Context, userID,id string) error { if m.DeleteFn!=nil { return m.DeleteFn(ctx,userID,id) }; return nil }
