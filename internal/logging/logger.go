package logging

import (
	"log/slog"
	"os"
	"strings"

	"github.com/lmittmann/tint"
	"github.com/mattn/go-isatty"
)

const bannerDivider = "=============================================================================="

func New() *slog.Logger {
	return slog.New(tint.NewHandler(os.Stdout, &tint.Options{
		NoColor: !isatty.IsTerminal(os.Stdout.Fd()),
	}))
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
