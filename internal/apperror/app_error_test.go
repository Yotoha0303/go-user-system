package apperror

import (
	"errors"
	"testing"
)

func TestAppErrorUnwrapsCause(t *testing.T) {
	cause := errors.New("root cause")
	err := Wrap(500, 1001, "wrapped error", cause)

	if !errors.Is(err, cause) {
		t.Fatalf("expected wrapped error to match cause")
	}
	if err.Error() != "wrapped error" {
		t.Fatalf("expected error message wrapped error, got %s", err.Error())
	}
}

func TestFromErrorFindsAppError(t *testing.T) {
	appErr := New(400, 1001, "bad request")

	got, ok := FromError(appErr)

	if !ok {
		t.Fatal("expected FromError to return true")
	}
	if got != appErr {
		t.Fatal("expected original app error")
	}
}

func TestFromErrorRejectsPlainError(t *testing.T) {
	_, ok := FromError(errors.New("plain error"))

	if ok {
		t.Fatal("expected FromError to return false")
	}
}
