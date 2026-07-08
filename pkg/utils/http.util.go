package utils

import (
	"encoding/json"
	"errors"
	"net/http"
)

// WriteSuccess writes a JSON response with the given status code and payload.
func WriteSuccess(w http.ResponseWriter, statusCode int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	if data != nil {
		_ = json.NewEncoder(w).Encode(data)
	}
}

// WriteError writes a JSON error response. If err is (or wraps) a *CustomError,
// its StatusCode and Message are used; otherwise it defaults to a 500 with a
// generic message, since a plain error's text may leak internal details.
func WriteError(w http.ResponseWriter, err error) {
	statusCode := http.StatusInternalServerError
	message := "something went wrong"

	var customErr *CustomError
	if errors.As(err, &customErr) {
		statusCode = customErr.StatusCode
		message = customErr.Message
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": message})
}

func Wrap(handler http.Handler, middleware ...func(http.Handler) http.Handler) http.Handler {
	for i := len(middleware) - 1; i >= 0; i-- {
		handler = middleware[i](handler)
	}
	return handler
}
