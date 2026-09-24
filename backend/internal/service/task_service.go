package service

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/url"
	"strconv"
	"strings"
	"time"

	"task-management/backend/internal/model"
	"task-management/backend/internal/repository"
)

const taskListCacheTTL = 60 * time.Second

type TaskCache interface {
	Get(context.Context, string) ([]byte, error)
	Set(context.Context, string, []byte, time.Duration) error
	InvalidateTaskLists(context.Context) error
}

type TaskService struct {
	repository repository.TaskRepository
	cache      TaskCache
}

func NewTaskService(repository repository.TaskRepository, cache TaskCache) *TaskService {
	return &TaskService{repository: repository, cache: cache}
}

func NormalizeFilter(filter model.TaskFilter) model.TaskFilter {
	filter.Status = strings.ToLower(strings.TrimSpace(filter.Status))
	filter.Keyword = strings.TrimSpace(filter.Keyword)
	filter.Assignee = strings.TrimSpace(filter.Assignee)
	if filter.Page == 0 {
		filter.Page = 1
	}
	if filter.Limit == 0 {
		filter.Limit = 10
	}
	if filter.Sort == "" {
		filter.Sort = model.DefaultSort
	}
	return filter
}

func CacheKey(filter model.TaskFilter) string {
	filter = NormalizeFilter(filter)
	values := url.Values{}
	values.Set("assignee", filter.Assignee)
	values.Set("keyword", filter.Keyword)
	values.Set("limit", strconv.Itoa(filter.Limit))
	values.Set("page", strconv.Itoa(filter.Page))
	values.Set("sort", filter.Sort)
	values.Set("status", filter.Status)
	return "tasks:list:" + values.Encode()
}

func (s *TaskService) List(ctx context.Context, filter model.TaskFilter) (model.TaskList, error) {
	filter = NormalizeFilter(filter)
	key := CacheKey(filter)
	if s.cache != nil {
		payload, err := s.cache.Get(ctx, key)
		if err == nil && len(payload) > 0 {
			var result model.TaskList
			if json.Unmarshal(payload, &result) == nil {
				return result, nil
			}
		} else if err != nil {
			log.Printf("task list cache read failed: %v", err)
		}
	}
	result, err := s.repository.List(ctx, filter)
	if err != nil {
		return model.TaskList{}, err
	}
	if s.cache != nil {
		payload, marshalErr := json.Marshal(result)
		if marshalErr == nil {
			if err := s.cache.Set(ctx, key, payload, taskListCacheTTL); err != nil {
				log.Printf("task list cache write failed: %v", err)
			}
		}
	}
	return result, nil
}

func (s *TaskService) Create(ctx context.Context, input model.CreateTaskInput) (model.Task, error) {
	task, err := s.repository.Create(ctx, input)
	if err != nil {
		return model.Task{}, err
	}
	s.invalidate(ctx)
	return task, nil
}

func (s *TaskService) Update(ctx context.Context, id int64, input model.UpdateTaskInput) (model.Task, error) {
	task, err := s.repository.Update(ctx, id, input)
	if err != nil {
		return model.Task{}, err
	}
	s.invalidate(ctx)
	return task, nil
}

func (s *TaskService) Delete(ctx context.Context, id int64) error {
	if err := s.repository.Delete(ctx, id); err != nil {
		return err
	}
	s.invalidate(ctx)
	return nil
}

func (s *TaskService) invalidate(ctx context.Context) {
	if s.cache != nil {
		if err := s.cache.InvalidateTaskLists(ctx); err != nil {
			log.Printf("task list cache invalidation failed: %v", err)
		}
	}
}

func IsNotFound(err error) bool       { return errors.Is(err, repository.ErrNotFound) }
func IsDuplicateTitle(err error) bool { return errors.Is(err, repository.ErrDuplicateTitle) }
