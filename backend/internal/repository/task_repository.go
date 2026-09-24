package repository

import (
	"context"

	"task-management/backend/internal/model"
)

type TaskRepository interface {
	List(context.Context, model.TaskFilter) (model.TaskList, error)
	Create(context.Context, model.CreateTaskInput) (model.Task, error)
	Update(context.Context, int64, model.UpdateTaskInput) (model.Task, error)
	Delete(context.Context, int64) error
}
