package crypto

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/agent/sender"
	cryptohelper "github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/helpers/crypto"
	"github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/middlewares/gzip"
	"github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/middlewares/hash"
	models "github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/model"
)

func newKey(t *testing.T) *rsa.PrivateKey {
	t.Helper()

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	return key
}

// echoHandler возвращает тело запроса как есть и запоминает, был ли вызван.
func echoHandler(called *bool) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		*called = true
		_, _ = io.Copy(w, r.Body)
	})
}

func doRequest(h http.Handler, body []byte, encrypted bool) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/updates", bytes.NewReader(body))
	if encrypted {
		req.Header.Set("X-Encrypted", "true")
	}

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	return rec
}

func TestMiddleware(t *testing.T) {
	key := newKey(t)
	otherKey := newKey(t)

	plain := []byte(`[{"id":"Alloc","type":"gauge","value":1.5}]`)

	encrypted, err := cryptohelper.EncryptBodyWithKey(plain, &key.PublicKey)
	require.NoError(t, err)

	foreign, err := cryptohelper.EncryptBodyWithKey(plain, &otherKey.PublicKey)
	require.NoError(t, err)

	tampered := bytes.Clone(encrypted)
	tampered[len(tampered)-1] ^= 0x01

	tooLarge, err := cryptohelper.EncryptBodyWithKey(bytes.Repeat([]byte("a"), 1<<20), &key.PublicKey)
	require.NoError(t, err)

	tests := []struct {
		name       string
		key        *rsa.PrivateKey
		body       []byte
		encrypted  bool
		wantCode   int
		wantBody   []byte
		wantCalled bool
	}{
		{
			name:       "decrypts encrypted body",
			key:        key,
			body:       encrypted,
			encrypted:  true,
			wantCode:   http.StatusOK,
			wantBody:   plain,
			wantCalled: true,
		},
		{
			name:       "passes plain body through",
			key:        key,
			body:       plain,
			wantCode:   http.StatusOK,
			wantBody:   plain,
			wantCalled: true,
		},
		{
			name:      "body shorter than RSA block",
			key:       key,
			body:      make([]byte, 10),
			encrypted: true,
			wantCode:  http.StatusBadRequest,
		},
		{
			name:      "body without nonce and tag",
			key:       key,
			body:      encrypted[:key.Size()+4],
			encrypted: true,
			wantCode:  http.StatusBadRequest,
		},
		{
			name:      "foreign public key",
			key:       key,
			body:      foreign,
			encrypted: true,
			wantCode:  http.StatusBadRequest,
		},
		{
			name:      "tampered body",
			key:       key,
			body:      tampered,
			encrypted: true,
			wantCode:  http.StatusBadRequest,
		},
		{
			name:      "body larger than limit",
			key:       key,
			body:      tooLarge,
			encrypted: true,
			wantCode:  http.StatusRequestEntityTooLarge,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var called bool
			h := Middleware(tc.key)(echoHandler(&called))

			rec := doRequest(h, tc.body, tc.encrypted)

			assert.Equal(t, tc.wantCode, rec.Code)
			assert.Equal(t, tc.wantCalled, called, "вызов следующего хендлера")
			if tc.wantBody != nil {
				assert.Equal(t, tc.wantBody, rec.Body.Bytes())
			}
		})
	}
}

// Полная цепочка как на сервере: агент сжимает, подписывает и шифрует тело,
// сервер расшифровывает, распаковывает и проверяет подпись.
func TestMiddleware_WithAgentChain(t *testing.T) {
	const hashKey = "secret"
	key := newKey(t)

	var got []byte
	r := chi.NewRouter()
	r.Use(Middleware(key))
	r.Use(gzip.Middleware)
	r.Use(hash.Middleware(hashKey))
	r.Post("/updates", func(w http.ResponseWriter, r *http.Request) {
		got, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
	})

	srv := httptest.NewServer(r)
	defer srv.Close()

	value := 42.5
	batch := []models.Metrics{{ID: "Alloc", MType: models.Gauge, Value: &value}}

	s := sender.NewSender(strings.TrimPrefix(srv.URL, "http://"))
	err := s.SendMetricsBatch(context.Background(), batch, hashKey, &key.PublicKey)
	require.NoError(t, err)

	assert.JSONEq(t, `[{"id":"Alloc","type":"gauge","value":42.5}]`, string(got))
}
