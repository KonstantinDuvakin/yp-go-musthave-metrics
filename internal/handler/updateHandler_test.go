package handler

import (
	"net/http"
	"reflect"
	"testing"

	"github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/storage"
)

func TestUpdateHandler(t *testing.T) {
	type args struct {
		storage *storage.MemStorage
	}
	tests := []struct {
		name string
		args args
		want http.HandlerFunc
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := UpdateHandler(tt.args.storage); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("UpdateHandler() = %v, want %v", got, tt.want)
			}
		})
	}
}
