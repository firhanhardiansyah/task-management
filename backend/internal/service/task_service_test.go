package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"task-management/backend/internal/model"
)

type fakeRepository struct {
	listFilter model.TaskFilter
	listResult model.TaskList
	listCalls  int
	createTask model.Task
	updateTask model.Task
	err        error
	deletedID  int64
}

func (f *fakeRepository) List(_ context.Context, filter model.TaskFilter) (model.TaskList, error) {
	f.listFilter = filter
	f.listCalls++
	return f.listResult, f.err
}
func (f *fakeRepository) Create(_ context.Context, input model.CreateTaskInput) (model.Task, error) {
	if f.createTask.Title == "" {
		f.createTask.Title = input.Title
	}
	return f.createTask, f.err
}
func (f *fakeRepository) Update(_ context.Context, id int64, input model.UpdateTaskInput) (model.Task, error) {
	if f.updateTask.ID == 0 {
		f.updateTask = model.Task{ID: id, Title: input.Title, Description: input.Description, Status: input.Status, Assignee: input.Assignee}
	}
	return f.updateTask, f.err
}
func (f *fakeRepository) Delete(_ context.Context, id int64) error { f.deletedID = id; return f.err }

type fakeCache struct {
	values        map[string][]byte
	setTTL        time.Duration
	invalidations int
	err           error
}

func (f *fakeCache) Get(_ context.Context, key string) ([]byte, error) { return f.values[key], f.err }
func (f *fakeCache) Set(_ context.Context, key string, value []byte, ttl time.Duration) error {
	if f.values == nil {
		f.values = map[string][]byte{}
	}
	f.values[key], f.setTTL = value, ttl
	return f.err
}
func (f *fakeCache) InvalidateTaskLists(context.Context) error { f.invalidations++; return f.err }

func TestUpdateSucceedsAndInvalidatesCache(t *testing.T) {
	repo := &fakeRepository{}
	cache := &fakeCache{}
	svc := NewTaskService(repo, cache)
	task, err := svc.Update(context.Background(), 7, model.UpdateTaskInput{Title: "Updated", Status: "done"})
	if err != nil {
		t.Fatalf("Update returned error: %v", err)
	}
	if task.ID != 7 || task.Title != "Updated" {
		t.Fatalf("unexpected task: %#v", task)
	}
	if cache.invalidations != 1 {
		t.Fatalf("invalidations = %d, want 1", cache.invalidations)
	}
}

func TestListForwardsNormalizedKeywordAndCachesForSixtySeconds(t *testing.T) {
	repo := &fakeRepository{listResult: model.TaskList{Data: []model.Task{{ID: 1, Title: "Login bug"}}}}
	cache := &fakeCache{}
	svc := NewTaskService(repo, cache)
	result, err := svc.List(context.Background(), model.TaskFilter{Keyword: " login "})
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}
	if repo.listFilter.Keyword != "login" || repo.listFilter.Page != 1 || repo.listFilter.Limit != 10 {
		t.Fatalf("filter was not normalized: %#v", repo.listFilter)
	}
	if len(result.Data) != 1 || result.Data[0].Title != "Login bug" {
		t.Fatalf("unexpected result: %#v", result)
	}
	if cache.setTTL != 60*time.Second {
		t.Fatalf("TTL = %s, want 60s", cache.setTTL)
	}
	if _, err := svc.List(context.Background(), model.TaskFilter{Keyword: "login", Page: 1, Limit: 10, Sort: model.DefaultSort}); err != nil {
		t.Fatalf("cached List returned error: %v", err)
	}
	if repo.listCalls != 1 {
		t.Fatalf("repository list calls = %d, want 1 after cache hit", repo.listCalls)
	}
}

func TestMutationInvalidation(t *testing.T) {
	tests := []struct {
		name string
		call func(*TaskService) error
	}{
		{"create", func(s *TaskService) error {
			_, err := s.Create(context.Background(), model.CreateTaskInput{Title: "A", Status: "todo"})
			return err
		}},
		{"update", func(s *TaskService) error {
			_, err := s.Update(context.Background(), 1, model.UpdateTaskInput{Title: "A", Status: "done"})
			return err
		}},
		{"delete", func(s *TaskService) error { return s.Delete(context.Background(), 1) }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cache := &fakeCache{}
			if err := test.call(NewTaskService(&fakeRepository{}, cache)); err != nil {
				t.Fatalf("mutation returned error: %v", err)
			}
			if cache.invalidations != 1 {
				t.Fatalf("invalidations = %d, want 1", cache.invalidations)
			}
		})
	}
}

func TestFailedMutationDoesNotInvalidate(t *testing.T) {
	cache := &fakeCache{}
	svc := NewTaskService(&fakeRepository{err: errors.New("database unavailable")}, cache)
	if _, err := svc.Update(context.Background(), 1, model.UpdateTaskInput{}); err == nil {
		t.Fatal("expected error")
	}
	if cache.invalidations != 0 {
		t.Fatalf("invalidations = %d, want 0", cache.invalidations)
	}
}

func TestCacheKeyIsDeterministicAndDistinct(t *testing.T) {
	first := CacheKey(model.TaskFilter{Keyword: "login", Status: "todo"})
	second := CacheKey(model.TaskFilter{Status: "todo", Keyword: "login", Page: 1, Limit: 10, Sort: model.DefaultSort})
	third := CacheKey(model.TaskFilter{Status: "done", Keyword: "login"})
	if first != second {
		t.Fatalf("logically identical filters produced different keys: %q != %q", first, second)
	}
	if first == third {
		t.Fatalf("different filters produced the same key: %q", first)
	}
}
