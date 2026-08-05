package rnld

import (
	"errors"
	"testing"
)

func TestAPIError(t *testing.T) {
	err := &APIError{StatusCode: 500, Message: "erro interno"}
	if err.StatusCode != 500 {
		t.Errorf("expected 500, got %d", err.StatusCode)
	}
	if err.Error() == "" {
		t.Error("expected non-empty error message")
	}

	var asErr error = err
	if asErr.Error() != err.Error() {
		t.Errorf("APIError should satisfy the error interface")
	}
}

func TestNotFoundError(t *testing.T) {
	err := newNotFoundError("")
	if err.StatusCode != 404 {
		t.Errorf("expected 404, got %d", err.StatusCode)
	}
	if err.Message != "Jogador não encontrado" {
		t.Errorf("unexpected default message: %q", err.Message)
	}

	var apiErr *APIError
	if !errors.As(error(err), &apiErr) {
		t.Error("NotFoundError should unwrap to APIError")
	}

	custom := newNotFoundError("player não encontrado")
	if custom.Message != "player não encontrado" {
		t.Errorf("unexpected custom message: %q", custom.Message)
	}
}

func TestUnauthorizedError(t *testing.T) {
	err := newUnauthorizedError("")
	if err.StatusCode != 403 {
		t.Errorf("expected 403, got %d", err.StatusCode)
	}

	var apiErr *APIError
	if !errors.As(error(err), &apiErr) {
		t.Error("UnauthorizedError should unwrap to APIError")
	}
}
