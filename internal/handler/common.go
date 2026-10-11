package handler

import (
	"log/slog"
	"net/http"
)

func setCacheControl(w http.ResponseWriter, value string) {
	w.Header().Set("Cache-Control", value)
}

func writeInternalError(w http.ResponseWriter, r *http.Request, logger *slog.Logger, err error) {
	logger.Error("internal error", "method", r.Method, "path", r.URL.Path, "err", err)
	http.Error(w, "internal server error", http.StatusInternalServerError)
}
