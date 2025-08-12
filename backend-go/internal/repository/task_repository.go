package repository

import (
	"context"
	"errors"
	"time"

	"github.com/axfinn/todoIngPlus/backend-go/internal/models"
	"github.com/axfinn/todoIngPlus/backend-go/internal/repository/common"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type TaskRepository interface {
	Insert(ctx context.Context, t *models.Task) error
	List(ctx context.Context, userID string, status string, page, limit int64) ([]models.Task, int64, error)
	FindByID(ctx context.Context, userID, id string) (*models.Task, error)
	UpdatePartial(ctx context.Context, userID, id string, set bson.M) (*models.Task, error)
	Delete(ctx context.Context, userID, id string) error
}

type mongoTaskRepo struct{ db *mongo.Database }

func NewTaskRepository(db *mongo.Database) TaskRepository { return &mongoTaskRepo{db: db} }

func (r *mongoTaskRepo) coll() *mongo.Collection { return r.db.Collection("tasks") }

func (r *mongoTaskRepo) Insert(ctx context.Context, t *models.Task) error {
	if t == nil {
		return errors.New("nil task")
	}
	now := time.Now()
	t.CreatedAt = now
	t.UpdatedAt = now
	res, err := r.coll().InsertOne(ctx, t)
	if err != nil {
		return err
	}
	if oid, ok := res.InsertedID.(primitive.ObjectID); ok {
		t.ID = oid.Hex()
	}
	return nil
}

func (r *mongoTaskRepo) List(ctx context.Context, userID, status string, page, limit int64) ([]models.Task, int64, error) {
	filter := bson.M{"createdBy": userID}
	if status != "" {
		filter["status"] = status
	}
	page, limit = common.Normalize(page, limit, 200)
	opts := options.Find().SetLimit(limit).SetSkip((page - 1) * limit)
	cur, err := r.coll().Find(ctx, filter, opts)
	if err != nil {
		return nil, 0, err
	}
	defer cur.Close(ctx)
	var list []models.Task
	for cur.Next(ctx) {
		var m models.Task
		if cur.Decode(&m) == nil {
			list = append(list, m)
		}
	}
	return list, int64(len(list)), cur.Err()
}

func (r *mongoTaskRepo) FindByID(ctx context.Context, userID, id string) (*models.Task, error) {
	filter := bson.M{"createdBy": userID, "$or": []bson.M{{"_id": id}}}
	if oid, err := primitive.ObjectIDFromHex(id); err == nil {
		filter["$or"] = append(filter["$or"].([]bson.M), bson.M{"_id": oid})
	}
	var m models.Task
	if err := r.coll().FindOne(ctx, filter).Decode(&m); err != nil {
		return nil, err
	}
	return &m, nil
}

func (r *mongoTaskRepo) UpdatePartial(ctx context.Context, userID, id string, set bson.M) (*models.Task, error) {
	set["updatedAt"] = time.Now()
	filter := bson.M{"createdBy": userID, "$or": []bson.M{{"_id": id}}}
	if oid, err := primitive.ObjectIDFromHex(id); err == nil {
		filter["$or"] = append(filter["$or"].([]bson.M), bson.M{"_id": oid})
	}
	if _, err := r.coll().UpdateOne(ctx, filter, bson.M{"$set": set}); err != nil {
		return nil, err
	}
	return r.FindByID(ctx, userID, id)
}

func (r *mongoTaskRepo) Delete(ctx context.Context, userID, id string) error {
	filter := bson.M{"createdBy": userID, "$or": []bson.M{{"_id": id}}}
	if oid, err := primitive.ObjectIDFromHex(id); err == nil {
		filter["$or"] = append(filter["$or"].([]bson.M), bson.M{"_id": oid})
	}
	_, err := r.coll().DeleteOne(ctx, filter)
	return err
}
