package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/go-sql-driver/mysql"
	"task-management/backend/internal/model"
)

var (
	ErrNotFound       = errors.New("task not found")
	ErrDuplicateTitle = errors.New("duplicate task title")
)

var allowedSorts = map[string]string{
	"created_at_asc":  "created_at ASC",
	"created_at_desc": "created_at DESC",
	"title_asc":       "title ASC",
	"title_desc":      "title DESC",
}

type MySQLTaskRepository struct{ db *sql.DB }

func NewMySQLTaskRepository(db *sql.DB) *MySQLTaskRepository { return &MySQLTaskRepository{db: db} }

func buildListQuery(filter model.TaskFilter) (string, string, []any) {
	where := []string{"deleted_at IS NULL"}
	args := make([]any, 0, 3)
	if filter.Status != "" {
		where = append(where, "status = ?")
		args = append(args, filter.Status)
	}
	if filter.Keyword != "" {
		where = append(where, "(title LIKE ? OR description LIKE ?)")
		term := "%" + filter.Keyword + "%"
		args = append(args, term, term)
	}
	if filter.Assignee != "" {
		where = append(where, "assignee = ?")
		args = append(args, filter.Assignee)
	}
	clause := strings.Join(where, " AND ")
	order := allowedSorts[filter.Sort]
	if order == "" {
		order = allowedSorts[model.DefaultSort]
	}
	selectQuery := "SELECT id, title, description, status, assignee, created_at, updated_at, deleted_at FROM tasks WHERE " + clause + " ORDER BY " + order + " LIMIT ? OFFSET ?"
	countQuery := "SELECT COUNT(*) FROM tasks WHERE " + clause
	return selectQuery, countQuery, args
}

func (r *MySQLTaskRepository) List(ctx context.Context, filter model.TaskFilter) (model.TaskList, error) {
	query, countQuery, args := buildListQuery(filter)
	var total int
	if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return model.TaskList{}, fmt.Errorf("count tasks: %w", err)
	}
	listArgs := append(append([]any{}, args...), filter.Limit, (filter.Page-1)*filter.Limit)
	rows, err := r.db.QueryContext(ctx, query, listArgs...)
	if err != nil {
		return model.TaskList{}, fmt.Errorf("list tasks: %w", err)
	}
	defer rows.Close()
	tasks := make([]model.Task, 0)
	for rows.Next() {
		var task model.Task
		if err := rows.Scan(&task.ID, &task.Title, &task.Description, &task.Status, &task.Assignee, &task.CreatedAt, &task.UpdatedAt, &task.DeletedAt); err != nil {
			return model.TaskList{}, fmt.Errorf("scan task: %w", err)
		}
		tasks = append(tasks, task)
	}
	if err := rows.Err(); err != nil {
		return model.TaskList{}, fmt.Errorf("iterate tasks: %w", err)
	}
	totalPages := 0
	if total > 0 {
		totalPages = (total + filter.Limit - 1) / filter.Limit
	}
	return model.TaskList{Data: tasks, Meta: model.PageMeta{Page: filter.Page, Limit: filter.Limit, Total: total, TotalPages: totalPages}}, nil
}

func (r *MySQLTaskRepository) Create(ctx context.Context, input model.CreateTaskInput) (model.Task, error) {
	result, err := r.db.ExecContext(ctx, "INSERT INTO tasks (title, description, status, assignee) VALUES (?, ?, ?, ?)", input.Title, input.Description, input.Status, input.Assignee)
	if err != nil {
		return model.Task{}, mapMySQLError(err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return model.Task{}, fmt.Errorf("read task id: %w", err)
	}
	return r.findActive(ctx, id)
}

func (r *MySQLTaskRepository) Update(ctx context.Context, id int64, input model.UpdateTaskInput) (model.Task, error) {
	result, err := r.db.ExecContext(ctx, "UPDATE tasks SET title = ?, description = ?, status = ?, assignee = ?, updated_at = NOW() WHERE id = ? AND deleted_at IS NULL", input.Title, input.Description, input.Status, input.Assignee, id)
	if err != nil {
		return model.Task{}, mapMySQLError(err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return model.Task{}, fmt.Errorf("read affected rows: %w", err)
	}
	if affected == 0 {
		return model.Task{}, ErrNotFound
	}
	return r.findActive(ctx, id)
}

func (r *MySQLTaskRepository) Delete(ctx context.Context, id int64) error {
	result, err := r.db.ExecContext(ctx, "UPDATE tasks SET deleted_at = NOW(), updated_at = NOW() WHERE id = ? AND deleted_at IS NULL", id)
	if err != nil {
		return fmt.Errorf("delete task: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("read affected rows: %w", err)
	}
	if affected == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *MySQLTaskRepository) findActive(ctx context.Context, id int64) (model.Task, error) {
	var task model.Task
	err := r.db.QueryRowContext(ctx, "SELECT id, title, description, status, assignee, created_at, updated_at, deleted_at FROM tasks WHERE id = ? AND deleted_at IS NULL", id).Scan(&task.ID, &task.Title, &task.Description, &task.Status, &task.Assignee, &task.CreatedAt, &task.UpdatedAt, &task.DeletedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return model.Task{}, ErrNotFound
	}
	if err != nil {
		return model.Task{}, fmt.Errorf("find task: %w", err)
	}
	return task, nil
}

func mapMySQLError(err error) error {
	var mysqlErr *mysql.MySQLError
	if errors.As(err, &mysqlErr) && mysqlErr.Number == 1062 {
		return ErrDuplicateTitle
	}
	return fmt.Errorf("database operation: %w", err)
}
