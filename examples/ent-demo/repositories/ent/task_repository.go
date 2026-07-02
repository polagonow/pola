package ent

import (
	"context"
	"fmt"

	"ent-demo/db/client/ent"
	entTask "ent-demo/db/client/ent/task"
	"ent-demo/repositories"
)

type taskRepository struct {
	client *ent.Client
}

// NewTaskRepository creates a new Ent-backed TaskRepository.
func NewTaskRepository(client *ent.Client) repositories.TaskRepository {
	return &taskRepository{client: client}
}

func (r *taskRepository) Create(ctx context.Context, entity *repositories.Task) error {
	_, err := r.client.Task.Create().
		SetTitle(entity.Title).
		SetDone(entity.Done).
		Save(ctx)
	return err
}

func (r *taskRepository) Get(ctx context.Context, id uint) (*repositories.Task, error) {
	e, err := r.client.Task.Get(ctx, int(id))
	if err != nil {
		return nil, fmt.Errorf("get task by id: %w", err)
	}
	return &repositories.Task{
		ID:    uint(e.ID),
		Title: e.Title,
		Done:  e.Done,
	}, nil
}

func (r *taskRepository) List(ctx context.Context, params repositories.ListParams) (*repositories.ListResult[*repositories.Task], error) {
	params = params.Normalize()

	total, err := r.client.Task.Query().Count(ctx)
	if err != nil {
		return nil, fmt.Errorf("count tasks: %w", err)
	}

	entries, err := r.client.Task.Query().
		Offset(params.Offset()).
		Limit(params.PerPage).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("list tasks: %w", err)
	}

	items := make([]*repositories.Task, len(entries))
	for i, e := range entries {
		items[i] = &repositories.Task{
			ID:    uint(e.ID),
			Title: e.Title,
			Done:  e.Done,
		}
	}

	totalPages := total / params.PerPage
	if total%params.PerPage != 0 {
		totalPages++
	}

	return &repositories.ListResult[*repositories.Task]{
		Items:      items,
		Total:      total,
		Page:       params.Page,
		PerPage:    params.PerPage,
		TotalPages: totalPages,
	}, nil
}

func (r *taskRepository) Update(ctx context.Context, entity *repositories.Task) error {
	_, err := r.client.Task.UpdateOneID(int(entity.ID)).
		SetTitle(entity.Title).
		SetDone(entity.Done).
		Save(ctx)
	return err
}

func (r *taskRepository) Delete(ctx context.Context, id uint) error {
	return r.client.Task.DeleteOneID(int(id)).Exec(ctx)
}

// Ensure task import is used.
var _ = entTask.Table
