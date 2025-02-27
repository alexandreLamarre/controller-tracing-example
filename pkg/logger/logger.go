package logger

import (
	"log/slog"
	"os"
	"time"

	"github.com/lmittmann/tint"
	"github.com/sirupsen/logrus"
)

func init() {
	w := os.Stderr
	logger := slog.New(
		tint.NewHandler(w, &tint.Options{
			Level:      slog.LevelDebug,
			TimeFormat: time.Kitchen,
		}),
	)
	slog.SetDefault(logger)

	logrus.SetLevel(logrus.TraceLevel)
}
