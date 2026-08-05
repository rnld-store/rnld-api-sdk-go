package rnld

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestHttpGetSerializesQueryAndHeaders(t *testing.T) {
	var gotQuery url.Values
	var gotHeaders http.Header

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.Query()
		gotHeaders = r.Header
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":true}`))
	}))
	defer srv.Close()

	var out struct {
		Status bool `json:"status"`
	}
	err := httpGet(context.Background(), http.DefaultClient, srv.URL+"/whitelists/query", testAPIKey, map[string]string{
		"guild_id": "123",
		"wl_id":    "WPLP",
		"ignored":  "",
	}, &out)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !out.Status {
		t.Error("expected status true")
	}

	if gotQuery.Get("guild_id") != "123" || gotQuery.Get("wl_id") != "WPLP" {
		t.Errorf("unexpected query: %+v", gotQuery)
	}
	if gotQuery.Has("ignored") {
		t.Errorf("expected empty value to be omitted, got %+v", gotQuery)
	}
	if gotHeaders.Get("x-api-key") != testAPIKey {
		t.Errorf("unexpected x-api-key header: %q", gotHeaders.Get("x-api-key"))
	}
	if gotHeaders.Get("Content-Type") != "application/json" {
		t.Errorf("unexpected Content-Type header: %q", gotHeaders.Get("Content-Type"))
	}
}

func TestHttpPatchSendsBodyAndHeaders(t *testing.T) {
	srv, captured := startServer(t, 200, map[string]any{"status": true})

	var out struct {
		Status bool `json:"status"`
	}
	err := httpPatch(context.Background(), http.DefaultClient, srv.URL+"/whitelists/liberar", testAPIKey, map[string]any{
		"guild_id": "123", "wl_id": "WPLP",
	}, &out)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !out.Status {
		t.Error("expected status true")
	}
	if captured.Method != http.MethodPatch {
		t.Errorf("expected PATCH, got %s", captured.Method)
	}
	if captured.Body["wl_id"] != "WPLP" {
		t.Errorf("unexpected body: %+v", captured.Body)
	}
}

func TestHttpPatchPropagatesUnauthorized(t *testing.T) {
	srv, _ := startServer(t, 403, map[string]any{"mensagem": "Acesso negado"})

	err := httpPatch(context.Background(), http.DefaultClient, srv.URL, testAPIKey, map[string]any{}, nil)
	var unauthorized *UnauthorizedError
	if !errors.As(err, &unauthorized) {
		t.Fatalf("expected UnauthorizedError, got %v", err)
	}
}

func TestHttpPatchPropagatesNotFound(t *testing.T) {
	srv, _ := startServer(t, 404, map[string]any{"mensagem": "Não encontrado"})

	err := httpPatch(context.Background(), http.DefaultClient, srv.URL, testAPIKey, map[string]any{}, nil)
	var notFound *NotFoundError
	if !errors.As(err, &notFound) {
		t.Fatalf("expected NotFoundError, got %v", err)
	}
}

func TestHttpPatchPropagatesAPIErrorOnInvalidJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
		w.Write([]byte("not json"))
	}))
	defer srv.Close()

	err := httpPatch(context.Background(), http.DefaultClient, srv.URL, testAPIKey, map[string]any{}, nil)
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected APIError, got %v", err)
	}
}
