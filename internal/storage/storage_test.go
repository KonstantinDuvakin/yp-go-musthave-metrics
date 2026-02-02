package storage

import (
	"sync"
	"testing"
)

func TestAgentStorage_AddCounter(t *testing.T) {
	type fields struct {
		Gauge   GaugeMap
		Counter CounterMap
		mu      sync.RWMutex
	}
	type args struct {
		name string
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
				Gauge:   make(GaugeMap),
				Counter: make(CounterMap),
				mu:      sync.RWMutex{},
			},
			args: args{
				name: "foo",
			},
			want: 1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			as := &AgentStorage{
				Gauge:   tt.fields.Gauge,
				Counter: tt.fields.Counter,
				mu:      tt.fields.mu,
			}
			as.AddCounter(tt.args.name)
			if as.Counter[tt.args.name] != tt.want {
				t.Errorf("AddCounter() got = %v, want %v", as.Counter[tt.args.name], tt.want)
			}
		})
	}
}

func TestAgentStorage_SetGauge(t *testing.T) {
	type fields struct {
		Gauge   GaugeMap
		Counter CounterMap
		mu      sync.RWMutex
	}
	type args struct {
		name  string
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
				Gauge:   make(GaugeMap),
				Counter: make(CounterMap),
				mu:      sync.RWMutex{},
			},
			args: args{
				name:  "foo",
				value: 1.0,
			},
			want: 1.0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			as := &AgentStorage{
				Gauge:   tt.fields.Gauge,
				Counter: tt.fields.Counter,
				mu:      tt.fields.mu,
			}
			as.SetGauge(tt.args.name, tt.args.value)
			if got := as.Gauge[tt.args.name]; got != tt.want {
				t.Errorf("SetGauge() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAgentStorage_Snapshot(t *testing.T) {
	type fields struct {
		Gauge   GaugeMap
		Counter CounterMap
		mu      sync.RWMutex
	}
	tests := []struct {
		name   string
		fields fields
		wantG  GaugeMap
		wantC  CounterMap
	}{
		{
			name: "snapshot returns copy of maps",
			fields: fields{
				Gauge:   GaugeMap{"g1": 1.5},
				Counter: CounterMap{"c1": 2},
			},
			wantG: GaugeMap{"g1": 1.5},
			wantC: CounterMap{"c1": 2},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			as := &AgentStorage{
				Gauge:   tt.fields.Gauge,
				Counter: tt.fields.Counter,
				mu:      tt.fields.mu,
			}
			gotG, gotC := as.Snapshot()
			gotG["g1"] = 999
			gotC["c1"] = 999
			if as.Gauge["g1"] == gotG["g1"] || as.Counter["c1"] == gotC["c1"] {
				t.Errorf("Snapshot() didn't make copy of AgentStorage")
			}
		})
	}
}

func TestMemStorage_AddCounter(t *testing.T) {
	type fields struct {
		Gauge   GaugeMap
		Counter CounterMap
		mu      sync.RWMutex
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
				Gauge:   make(GaugeMap),
				Counter: make(CounterMap),
				mu:      sync.RWMutex{},
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
				mu:      tt.fields.mu,
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
		Gauge   GaugeMap
		Counter CounterMap
		mu      sync.RWMutex
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
				Gauge:   make(GaugeMap),
				Counter: make(CounterMap),
				mu:      sync.RWMutex{},
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
				mu:      tt.fields.mu,
			}
			ms.SetGauge(tt.args.field, tt.args.value)
		})
	}
}

func TestNewAgentStorage(t *testing.T) {
	as := NewAgentStorage()
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
