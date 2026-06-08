package model

import "testing"

func TestUserTableName(t *testing.T) {
	if got := (User{}).TableName(); got != "users" {
		t.Fatalf("expected table name users, got %s", got)
	}
}
