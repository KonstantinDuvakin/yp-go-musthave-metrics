package mem_storage

import (
	"testing"

	"github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/storage"
	"github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/storage/agent_storage"
)

func TestMemStorage_AddCounter(t *testing.T) {
	type fields struct {
		Gauge   storage.GaugeMap
		Counter storage.CounterMap
	}
	type args struct {
		field string
		value int64
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   int64
	}{
		{
			name: "add foo counter",
			fields: fields{
				Gauge:   make(storage.GaugeMap),
				Counter: make(storage.CounterMap),
			},
			args: args{
				field: "foo",
				value: 1,
			},
			want: 1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ms := &MemStorage{
				Gauge:   tt.fields.Gauge,
				Counter: tt.fields.Counter,
			}
			ms.AddCounter(tt.args.field, tt.args.value)
			if got := ms.Counter[tt.args.field]; got != tt.want {
				t.Errorf("AddCounter() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMemStorage_SetGauge(t *testing.T) {
	type fields struct {
		Gauge   storage.GaugeMap
		Counter storage.CounterMap
	}
	type args struct {
		field string
		value float64
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   float64
	}{
		{
			name: "add foo gauge field",
			fields: fields{
				Gauge:   make(storage.GaugeMap),
				Counter: make(storage.CounterMap),
			},
			args: args{
				field: "foo",
				value: 1.0,
			},
			want: 1.0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ms := &MemStorage{
				Gauge:   tt.fields.Gauge,
				Counter: tt.fields.Counter,
			}
			ms.SetGauge(tt.args.field, tt.args.value)
		})
	}
}

func TestNewAgentStorage(t *testing.T) {
	as := agent_storage.NewAgentStorage()
	if as == nil {
		t.Fatal("NewAgentStorage returned nil")
	}
	if as.Gauge == nil || as.Counter == nil {
		t.Fatal("Gauge/Counter maps must be initialized")
	}
}

func TestNewMemStorage(t *testing.T) {
	ms := NewMemStorage()
	if ms == nil {
		t.Fatal("NewAgentStorage returned nil")
	}
	if ms.Gauge == nil || ms.Counter == nil {
		t.Fatal("Gauge/Counter maps must be initialized")
	}
}
