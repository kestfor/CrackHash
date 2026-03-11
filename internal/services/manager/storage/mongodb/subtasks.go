package mongodb

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/kestfor/CrackHash/internal/services/manager"
	"github.com/kestfor/CrackHash/internal/services/worker"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type subTaskStorage struct {
	col *mongo.Collection
}

func NewSubTaskStorage(db *mongo.Database) (*subTaskStorage, error) {
	col := db.Collection("subtasks")

	index := mongo.IndexModel{
		Keys: bson.D{
			{Key: "task_id", Value: 1},
			{Key: "start_index", Value: 1},
			{Key: "end_index", Value: 1},
		},
		Options: options.Index().SetUnique(true),
	}

	if _, err := col.Indexes().CreateOne(context.Background(), index); err != nil {
		return nil, fmt.Errorf("create subtasks index: %w", err)
	}

	return &subTaskStorage{col: col}, nil
}

func (s *subTaskStorage) CreateBatch(ctx context.Context, tasks []worker.Task) error {
	docs := make([]interface{}, len(tasks))
	for i, t := range tasks {
		docs[i] = manager.SubTask{
			Task:   t,
			Status: manager.SubTaskStatusPending,
		}
	}

	_, err := s.col.InsertMany(ctx, docs)
	if err != nil {
		return fmt.Errorf("insert subtasks: %w", err)
	}

	return nil
}

func (s *subTaskStorage) FindPending(ctx context.Context) ([]manager.SubTask, error) {
	cursor, err := s.col.Find(ctx, bson.M{"status": manager.SubTaskStatusPending})
	if err != nil {
		return nil, fmt.Errorf("find pending subtasks: %w", err)
	}
	defer cursor.Close(ctx)

	var result []manager.SubTask
	if err := cursor.All(ctx, &result); err != nil {
		return nil, fmt.Errorf("decode pending subtasks: %w", err)
	}

	return result, nil
}

func (s *subTaskStorage) MarkSent(ctx context.Context, taskID uuid.UUID, startIndex, endIndex uint64) error {
	filter := bson.M{
		"task_id":     taskID,
		"start_index": startIndex,
		"end_index":   endIndex,
	}

	update := bson.M{
		"$set": bson.M{"status": manager.SubTaskStatusSent},
	}

	_, err := s.col.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("mark subtask sent: %w", err)
	}

	return nil
}

func (s *subTaskStorage) Has(ctx context.Context, taskID uuid.UUID) (bool, error) {
	filter := bson.M{
		"task_id": taskID,
	}
	count, err := s.col.CountDocuments(ctx, filter)
	if err != nil {
		return false, fmt.Errorf("check subtasks: %w", err)
	}
	return count > 0, nil
}
