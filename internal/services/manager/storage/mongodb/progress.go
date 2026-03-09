package mongodb

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/kestfor/CrackHash/internal/services/worker"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type taskProgressStorage struct {
	col *mongo.Collection
}

func NewTaskProgressStorage(db *mongo.Database) (*taskProgressStorage, error) {
	col := db.Collection("task_progress")

	index := mongo.IndexModel{
		Keys: bson.D{
			{Key: "task_id", Value: 1},
			{Key: "worker_id", Value: 1},
		},
		Options: options.Index().SetUnique(true),
	}

	_, err := col.Indexes().CreateOne(context.Background(), index)
	if err != nil {
		return nil, fmt.Errorf("create task progress index: %w", err)
	}

	return &taskProgressStorage{
		col: col,
	}, nil
}

func (r *taskProgressStorage) Upsert(ctx context.Context, p worker.TaskProgress) error {
	filter := bson.M{
		"task_id":   p.TaskID,
		"worker_id": p.WorkerID,
		"$or": []bson.M{
			{"iterations_done": bson.M{"$lte": p.IterationsDone}},
			{"iterations_done": bson.M{"$exists": false}},
		},
	}

	update := bson.M{
		"$set": bson.M{
			"status":           p.Status,
			"iterations_done":  p.IterationsDone,
			"total_iterations": p.TotalIterations,
			"result":           p.Result,
			"updated_at":       time.Now(),
		},
	}

	opts := options.Update().SetUpsert(true)

	_, err := r.col.UpdateOne(ctx, filter, update, opts)
	return err
}

func (r *taskProgressStorage) Collect(
	ctx context.Context,
	taskID uuid.UUID,
) ([]worker.TaskProgress, error) {

	filter := bson.M{
		"task_id": taskID,
	}

	cursor, err := r.col.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var result []worker.TaskProgress

	for cursor.Next(ctx) {
		p := worker.TaskProgress{}
		if err := cursor.Decode(&p); err != nil {
			return nil, err
		}

		result = append(result, p)
	}

	return result, cursor.Err()
}
