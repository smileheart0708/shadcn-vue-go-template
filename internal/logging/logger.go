package logging

import (
	"context"
	"log/slog"
	"os"
	"strings"

	"github.com/lmittmann/tint"
	"github.com/mattn/go-isatty"
)

const bannerDivider = "=============================================================================="

func New() *slog.Logger {
	logger, _ := NewWithStream(StreamOptions{})
	return logger
}

func NewWithStream(options StreamOptions) (*slog.Logger, *Stream) {
	stream := NewStream(options)
	consoleHandler := tint.NewHandler(os.Stdout, &tint.Options{
		NoColor: !isatty.IsTerminal(os.Stdout.Fd()),
	})

	logger := slog.New(newFanoutHandler(consoleHandler, NewStreamHandler(stream, "app")))
	return logger, stream
}

func LogStartupBanner(logger *slog.Logger, listenAddr string, dataDir string) {
	if logger == nil {
		logger = slog.Default()
	}

	logger.Info(bannerDivider)
	logger.Info("listen: http://localhost" + strings.TrimSpace(listenAddr))
	logger.Info("data_dir: " + strings.TrimSpace(dataDir))
	logger.Info(bannerDivider)
}

type fanoutHandler struct {
	handlers []slog.Handler
}

func newFanoutHandler(handlers ...slog.Handler) slog.Handler {
	filtered := make([]slog.Handler, 0, len(handlers))
	for _, handler := range handlers {
		if handler != nil {
			filtered = append(filtered, handler)
		}
	}
	return &fanoutHandler{handlers: filtered}
}

func (h *fanoutHandler) Enabled(ctx context.Context, level slog.Level) bool {
	for _, handler := range h.handlers {
		if handler.Enabled(ctx, level) {
			return true
		}
	}
	return false
}

func (h *fanoutHandler) Handle(ctx context.Context, record slog.Record) error {
	var firstErr error
	for _, handler := range h.handlers {
		if !handler.Enabled(ctx, record.Level) {
			continue
		}
		if err := handler.Handle(ctx, record.Clone()); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

func (h *fanoutHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	handlers := make([]slog.Handler, 0, len(h.handlers))
	for _, handler := range h.handlers {
		handlers = append(handlers, handler.WithAttrs(attrs))
	}
	return &fanoutHandler{handlers: handlers}
}

func (h *fanoutHandler) WithGroup(name string) slog.Handler {
	handlers := make([]slog.Handler, 0, len(h.handlers))
	for _, handler := range h.handlers {
		handlers = append(handlers, handler.WithGroup(name))
	}
	return &fanoutHandler{handlers: handlers}
}
