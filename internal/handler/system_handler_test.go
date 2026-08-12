package handler

import (
	"go-user-system/internal/buildinfo"
	"go-user-system/internal/response"
	"net/http"
	"testing"
)

func TestVersionHandlerReturnsBuildInformation(t *testing.T) {
	h := NewSystemHandler(buildinfo.Info{
		Version:   "v1.2.3",
		Commit:    "abc123",
		BuildTime: "2026-08-12T10:00:00Z",
	})

	recorder := performJSONRequest(h.VersionHandler, http.MethodGet, "/version", "")
	body := decodeResponse(t, recorder)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}
	if body.Code != response.CodeSuccess {
		t.Fatalf("expected code %d, got %d", response.CodeSuccess, body.Code)
	}
	data, ok := body.Data.(map[string]interface{})
	if !ok {
		t.Fatalf("expected map data, got %T", body.Data)
	}
	if data["version"] != "v1.2.3" || data["commit"] != "abc123" || data["build_time"] != "2026-08-12T10:00:00Z" {
		t.Fatalf("unexpected build information: %#v", data)
	}
}
