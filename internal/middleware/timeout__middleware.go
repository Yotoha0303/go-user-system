package middleware

import (
	"encoding/json"
	"go-user-system/internal/response"
	"net/http"
	"time"
)

func TimeoutHandler(next http.Handler, timeout time.Duration) http.Handler {
	if timeout <= 0 {
		return next
	}

	body, err := json.Marshal(response.Response{
		Code: response.CodeRequestTimeout,
		Msg:  "request timeout",
	})

	if err != nil {
		body = []byte(`{"code":5002,"message":"request timeout"}`)
	}

	handler := http.TimeoutHandler(next, timeout, string(body))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := ensureRequestID(r)
		w.Header().Set(RequestIDHeader, requestID)
		w.Header().Set("Content-Type", "application/json;charset=utf-8")
		handler.ServeHTTP(w, r)
	})
}
