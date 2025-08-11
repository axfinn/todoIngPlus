package repository

import (
	"context"
	"errors"
	"time"

	"github.com/axfinn/todoIngPlus/backend-go/internal/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type ReportRepository interface {
	Insert(ctx context.Context, r *models.Report) error
	FindByID(ctx context.Context, userID, id string) (*models.Report, error)
	ListByUser(ctx context.Context, userID string) ([]models.Report, error)
	Delete(ctx context.Context, userID, id string) (bool, error)
}

type mongoReportRepo struct { db *mongo.Database }

func NewReportRepository(db *mongo.Database) ReportRepository { return &mongoReportRepo{db: db} }

func (m *mongoReportRepo) coll() *mongo.Collection { return m.db.Collection("reports") }

func (m *mongoReportRepo) Insert(ctx context.Context, r *models.Report) error {
	if r == nil { return errors.New("nil report") }
	if r.CreatedAt.IsZero() { r.CreatedAt = time.Now() }
	if r.UpdatedAt.IsZero() { r.UpdatedAt = r.CreatedAt }
	_, err := m.coll().InsertOne(ctx, r)
	return err
}

func (m *mongoReportRepo) FindByID(ctx context.Context, userID, id string) (*models.Report, error) {
	var out models.Report
	if err := m.coll().FindOne(ctx, bson.M{"_id": id, "userId": userID}).Decode(&out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (m *mongoReportRepo) ListByUser(ctx context.Context, userID string) ([]models.Report, error) {
	cur, err := m.coll().Find(ctx, bson.M{"userId": userID})
	if err != nil { return nil, err }
	defer cur.Close(ctx)
	var list []models.Report
	for cur.Next(ctx) { var r models.Report; if cur.Decode(&r)==nil { list = append(list, r) } }
	return list, cur.Err()
}

func (m *mongoReportRepo) Delete(ctx context.Context, userID, id string) (bool, error) {
	res, err := m.coll().DeleteOne(ctx, bson.M{"_id": id, "userId": userID})
	if err != nil { return false, err }
	return res.DeletedCount > 0, nil
}
