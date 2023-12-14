package log

import (
	"github.com/Graylog2/go-gelf/gelf"
	sloggraylog "github.com/samber/slog-graylog/v2"
	"gorm.io/gorm/logger"
	"io"
	"log"
	"log/slog"
	"os"
	"time"
)

func NewAppLogger() *AppLogger {
	addr := os.Getenv("LOG_CHANNEL")
	gelfWriter, err := gelf.NewWriter(addr)

	if err != nil {
		log.Print("Cant init logger " + err.Error())
	}

	l := AppLogger{
		logger: slog.New(sloggraylog.Option{Level: slog.LevelDebug, Writer: gelfWriter}.NewGraylogHandler()),
		writer: gelfWriter,
	}

	log.SetOutput(io.MultiWriter(os.Stderr, gelfWriter))

	return &l
}

type AppLogger struct {
	logger *slog.Logger
	writer *gelf.Writer
}

func (l *AppLogger) Writer() *gelf.Writer {
	return l.writer
}

func (l *AppLogger) GetGormLogger() logger.Interface {
	gl := logger.New(log.New(io.MultiWriter(os.Stderr, l.writer), "\r\n", log.LstdFlags), // io writer
		logger.Config{
			SlowThreshold:             time.Second,   // Slow SQL threshold
			LogLevel:                  logger.Silent, // Log level
			IgnoreRecordNotFoundError: false,         // Ignore ErrRecordNotFound error for logger
			ParameterizedQueries:      false,         // Don't include params in the SQL log
			Colorful:                  false,         // Disable color
		})
	return gl
}
