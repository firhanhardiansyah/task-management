package repository

import (
	"reflect"
	"strings"
	"testing"

	"task-management/backend/internal/model"
)

func TestBuildListQueryIncludesSearchAndSoftDelete(t *testing.T) {
	query, countQuery, args := buildListQuery(model.TaskFilter{
		Status: "todo", Keyword: "login", Assignee: "12", Page: 1, Limit: 10, Sort: "title_asc",
	})

	for _, fragment := range []string{"deleted_at IS NULL", "status = ?", "title LIKE ?", "description LIKE ?", "assignee = ?", "ORDER BY title ASC", "LIMIT ? OFFSET ?"} {
		if !strings.Contains(query, fragment) {
			t.Fatalf("query does not contain %q: %s", fragment, query)
		}
	}
	if !strings.Contains(countQuery, "deleted_at IS NULL") {
		t.Fatalf("count query must exclude soft-deleted tasks: %s", countQuery)
	}
	wantArgs := []any{"todo", "%login%", "%login%", "12"}
	if !reflect.DeepEqual(args, wantArgs) {
		t.Fatalf("arguments = %#v, want %#v", args, wantArgs)
	}
}

func TestBuildListQueryUsesWhitelistedSort(t *testing.T) {
	query, _, _ := buildListQuery(model.TaskFilter{Sort: "title; DROP TABLE tasks", Page: 1, Limit: 10})
	if strings.Contains(query, "DROP") || !strings.Contains(query, "ORDER BY created_at DESC") {
		t.Fatalf("unsafe sort was not replaced with default: %s", query)
	}
}
