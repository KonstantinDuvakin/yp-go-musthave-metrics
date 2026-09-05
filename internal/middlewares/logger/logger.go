// Package logger предоставляет общий структурированный логгер приложения и
// HTTP-middleware для логирования запросов.
package logger

import (
	"net/http"
	"time"

	"go.uber.org/zap"
)

// Log — глобальный логгер приложения. До вызова [InitializeLogger] это
// no-op логгер, который ничего не пишет.
var Log = zap.NewNop()

// InitializeLogger настраивает глобальный [Log] на продакшн-конфигурацию с
// уровнем логирования level (например, "info", "debug"). Возвращает ошибку
// при некорректном уровне или сбое инициализации.
func InitializeLogger(level string) error {
	lvl, err := zap.ParseAtomicLevel(level)
	if err != nil {
		return err
	}

	cfg := zap.NewProductionConfig()

	cfg.Level = lvl

	zl, err := cfg.Build()
	if err != nil {
		return err
	}

	Log = zl

	return nil
}

type (
	responseData struct {
		status int
		size   int
	}
	loggerResponseWriter struct {
		http.ResponseWriter
		responseData *responseData
	}
)

func (lrw *loggerResponseWriter) Write(b []byte) (int, error) {
	size, err := lrw.ResponseWriter.Write(b)
	lrw.responseData.size += size
	return size, err
}

func (lrw *loggerResponseWriter) WriteHeader(statusCode int) {
	lrw.ResponseWriter.WriteHeader(statusCode)
	lrw.responseData.status = statusCode
}

// RequestLogger оборачивает h, логируя каждый запрос: HTTP-метод, URI и
// длительность обработки, а также статус и размер ответа — через [Log].
func RequestLogger(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		rd := &responseData{
			status: 0,
			size:   0,
		}

		lrw := &loggerResponseWriter{
			ResponseWriter: w,
			responseData:   rd,
		}

		h.ServeHTTP(lrw, r)

		duration := time.Since(start)

		Log.Info("Incoming Request",
			zap.String("HTTP method", r.Method),
			zap.String("uri", r.RequestURI),
			zap.Duration("duration", duration),
		)
		Log.Info("Outcoming Response",
			zap.Duration("duration", duration),
			zap.Int("status", rd.status),
			zap.Int("size", rd.size),
		)
	})
}
