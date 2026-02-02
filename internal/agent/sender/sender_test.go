package sender

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/handler"
	"github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/storage"
)

func TestSendMetrics(t *testing.T) {
	type want struct {
		code        int
		response    string
		contentType string
	}
	type args struct {
		url string
	}
	tests := []struct {
		name string
		args args
		want want
	}{
		{
			name: "positive counter test #1",
			args: args{
				url: fmt.Sprintf("%supdate/counter/counter/1", serverURL),
			},
			want: want{
				code:        200,
				response:    `{"status":"ok"}`,
				contentType: "application/json",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, tt.args.url, nil)
			w := httptest.NewRecorder()
			ms := storage.NewMemStorage()
			h := handler.UpdateHandler(ms)
			h(w, request)
			res := w.Result()
			t.Log("res: ", res)
			if got := res.StatusCode; got != tt.want.code {
				t.Errorf("SendMetrics() = %d, want %d", got, tt.want.code)
			}
		})
	}
}

func TestUrlBuilder(t *testing.T) {
	type args struct {
		metricType string
		name       string
		value      string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "empty url",
			args: args{
				metricType: "",
				name:       "",
				value:      "",
			},
			want: fmt.Sprintf("%supdate///", serverURL),
		},
		{
			name: "counter url",
			args: args{
				metricType: "counter",
				name:       "counter",
				value:      "1",
			},
			want: fmt.Sprintf("%supdate/counter/counter/1", serverURL),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := UrlBuilder(tt.args.metricType, tt.args.name, tt.args.value); got != tt.want {
				t.Errorf("UrlBuilder() = %v, want %v", got, tt.want)
			}
		})
	}
}
