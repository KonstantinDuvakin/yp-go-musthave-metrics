// Package gzip предоставляет HTTP-middleware для прозрачного gzip-сжатия
// ответов и распаковки сжатых запросов.
package gzip

import (
	"compress/gzip"
	"io"
	"net/http"
	"strings"
	"sync"
)

type compressWriter struct {
	w  http.ResponseWriter
	zw *gzip.Writer
}

var writerPool = sync.Pool{
	New: func() any {
		return gzip.NewWriter(io.Discard)
	},
}

func newCompressWriter(w http.ResponseWriter) *compressWriter {
	zw := writerPool.Get().(*gzip.Writer)
	zw.Reset(w)
	return &compressWriter{
		w:  w,
		zw: zw,
	}
}

func (c *compressWriter) Header() http.Header {
	return c.w.Header()
}

func (c *compressWriter) Write(b []byte) (int, error) {
	return c.zw.Write(b)
}

func (c *compressWriter) WriteHeader(statusCode int) {
	if statusCode < 300 {
		c.w.Header().Set("Content-Encoding", "gzip")
	}
	c.w.WriteHeader(statusCode)
}

func (c *compressWriter) Close() error {
	err := c.zw.Close()
	writerPool.Put(c.zw)
	return err
}

type compressReader struct {
	r  io.ReadCloser
	zr *gzip.Reader
}

func newCompressReader(r io.ReadCloser) (*compressReader, error) {
	zr, err := gzip.NewReader(r)
	if err != nil {
		return nil, err
	}

	return &compressReader{
		r:  r,
		zr: zr,
	}, nil
}

func (c *compressReader) Read(p []byte) (n int, err error) {
	return c.zr.Read(p)
}

func (c *compressReader) Close() error {
	if err := c.r.Close(); err != nil {
		return err
	}
	return c.zr.Close()
}

// Middleware оборачивает h, добавляя поддержку gzip.
//
// Если клиент прислал заголовок Accept-Encoding: gzip, ответ сжимается.
// Если тело запроса пришло с Content-Encoding: gzip, оно прозрачно
// распаковывается перед передачей в h.
func Middleware(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ow := w

		acceptEncoding := r.Header.Get("Accept-Encoding")
		canSendEncoded := strings.Contains(acceptEncoding, "gzip")
		if canSendEncoded {
			cw := newCompressWriter(w)

			ow = cw

			defer cw.Close()
		}

		contentEncoding := r.Header.Get("Content-Encoding")
		shouldDecompress := strings.Contains(contentEncoding, "gzip")
		if shouldDecompress {
			cr, err := newCompressReader(r.Body)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			r.Body = cr

			defer cr.Close()
		}

		h.ServeHTTP(ow, r)
	})
}
