package handler

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"task-management/backend/internal/model"
	"task-management/backend/internal/repository"
	"task-management/backend/internal/service"
)

type handlerRepository struct{ updateErr error }

func (r *handlerRepository) List(context.Context, model.TaskFilter) (model.TaskList, error) {
	return model.TaskList{}, nil
}
func (r *handlerRepository) Create(context.Context, model.CreateTaskInput) (model.Task, error) {
	return model.Task{}, nil
}
func (r *handlerRepository) Update(context.Context, int64, model.UpdateTaskInput) (model.Task, error) {
	return model.Task{}, r.updateErr
}
func (r *handlerRepository) Delete(context.Context, int64) error { return nil }

func TestUpdateDuplicateTitleReturnsConflict(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	handler := NewTaskHandler(service.NewTaskService(&handlerRepository{updateErr: repository.ErrDuplicateTitle}, nil))
	handler.Register(router.Group("/api"))

	body := bytes.NewBufferString(`{"title":"Existing","description":"","status":"todo","assignee":""}`)
	request := httptest.NewRequest(http.MethodPut, "/api/tasks/1", body)
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusConflict {
		t.Fatalf("status = %d, want %d; body=%s", response.Code, http.StatusConflict, response.Body.String())
	}
	if !bytes.Contains(response.Body.Bytes(), []byte(`"code":"DUPLICATE_TASK_TITLE"`)) {
		t.Fatalf("unexpected response body: %s", response.Body.String())
	}
}
