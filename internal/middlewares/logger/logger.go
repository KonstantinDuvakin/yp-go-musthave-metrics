package logger

import (
	"net/http"
	"time"

	"go.uber.org/zap"
)

var Log = zap.NewNop()

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
		Log.Info("Outcoming Response", zap.Duration("duration", duration))
	})
}
