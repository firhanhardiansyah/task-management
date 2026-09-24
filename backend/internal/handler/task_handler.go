package handler

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"task-management/backend/internal/model"
	"task-management/backend/internal/service"
)

type TaskHandler struct{ service *service.TaskService }

func NewTaskHandler(service *service.TaskService) *TaskHandler { return &TaskHandler{service: service} }

func (h *TaskHandler) Register(group *gin.RouterGroup) {
	group.GET("/tasks", h.List)
	group.POST("/tasks", h.Create)
	group.PUT("/tasks/:id", h.Update)
	group.DELETE("/tasks/:id", h.Delete)
}

func (h *TaskHandler) List(c *gin.Context) {
	filter, err := parseFilter(c)
	if err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}
	result, err := h.service.List(c.Request.Context(), filter)
	if err != nil {
		writeInternalError(c, err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *TaskHandler) Create(c *gin.Context) {
	var input model.CreateTaskInput
	if err := c.ShouldBindJSON(&input); err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request body")
		return
	}
	if input.Status == "" {
		input.Status = model.StatusTodo
	}
	input = sanitizeCreate(input)
	if err := validateInput(input); err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}
	task, err := h.service.Create(c.Request.Context(), input)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	c.JSON(http.StatusCreated, task)
}

func (h *TaskHandler) Update(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var input model.UpdateTaskInput
	if err := c.ShouldBindJSON(&input); err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request body")
		return
	}
	input = sanitizeCreate(input)
	if err := validateInput(input); err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}
	task, err := h.service.Update(c.Request.Context(), id, input)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, task)
}

func (h *TaskHandler) Delete(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	if err := h.service.Delete(c.Request.Context(), id); err != nil {
		handleServiceError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func parseFilter(c *gin.Context) (model.TaskFilter, error) {
	page, err := parsePositiveInt(c.Query("page"), 1)
	if err != nil {
		return model.TaskFilter{}, errors.New("page must be a positive integer")
	}
	limit, err := parsePositiveInt(c.Query("limit"), 10)
	if err != nil || limit > 100 {
		return model.TaskFilter{}, errors.New("limit must be between 1 and 100")
	}
	status := strings.ToLower(strings.TrimSpace(c.Query("status")))
	if status != "" && !validStatus(status) {
		return model.TaskFilter{}, errors.New("status must be todo, in_progress, or done")
	}
	sort := strings.TrimSpace(c.Query("sort"))
	if sort == "" {
		sort = model.DefaultSort
	}
	validSort := sort == "created_at_asc" || sort == "created_at_desc" || sort == "title_asc" || sort == "title_desc"
	if !validSort {
		return model.TaskFilter{}, errors.New("invalid sort value")
	}
	return model.TaskFilter{Status: status, Keyword: c.Query("keyword"), Assignee: c.Query("assignee"), Page: page, Limit: limit, Sort: sort}, nil
}

func parsePositiveInt(raw string, fallback int) (int, error) {
	if raw == "" {
		return fallback, nil
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value < 1 {
		return 0, errors.New("invalid positive integer")
	}
	return value, nil
}

func parseID(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id < 1 {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Task ID must be a positive integer")
		return 0, false
	}
	return id, true
}

func sanitizeCreate(input model.CreateTaskInput) model.CreateTaskInput {
	input.Title = strings.TrimSpace(input.Title)
	input.Description = strings.TrimSpace(input.Description)
	input.Status = strings.ToLower(strings.TrimSpace(input.Status))
	input.Assignee = strings.TrimSpace(input.Assignee)
	return input
}

func validateInput(input model.CreateTaskInput) error {
	if input.Title == "" || len(input.Title) > 255 {
		return errors.New("title is required and must not exceed 255 characters")
	}
	if !validStatus(input.Status) {
		return errors.New("status must be todo, in_progress, or done")
	}
	return nil
}

func validStatus(status string) bool {
	return status == model.StatusTodo || status == model.StatusInProgress || status == model.StatusDone
}

func handleServiceError(c *gin.Context, err error) {
	switch {
	case service.IsNotFound(err):
		writeError(c, http.StatusNotFound, "TASK_NOT_FOUND", "Task not found")
	case service.IsDuplicateTitle(err):
		writeError(c, http.StatusConflict, "DUPLICATE_TASK_TITLE", "A task with this title already exists")
	default:
		writeInternalError(c, err)
	}
}

func writeInternalError(c *gin.Context, err error) {
	_ = c.Error(err)
	writeError(c, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "An unexpected error occurred")
}

func writeError(c *gin.Context, status int, code, message string) {
	c.JSON(status, gin.H{"error": gin.H{"code": code, "message": message}})
}
