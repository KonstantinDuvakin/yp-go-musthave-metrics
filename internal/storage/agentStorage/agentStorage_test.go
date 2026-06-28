package agentStorage

import (
	"testing"

	"github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/storage"
)

func TestAgentStorage_AddCounter(t *testing.T) {
	type fields struct {
		Gauge   storage.GaugeMap
		Counter storage.CounterMap
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
				Gauge:   make(storage.GaugeMap),
				Counter: make(storage.CounterMap),
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
		Gauge   storage.GaugeMap
		Counter storage.CounterMap
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
				Gauge:   make(storage.GaugeMap),
				Counter: make(storage.CounterMap),
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
		Gauge   storage.GaugeMap
		Counter storage.CounterMap
	}
	tests := []struct {
		name   string
		fields fields
		wantG  storage.GaugeMap
		wantC  storage.CounterMap
	}{
		{
			name: "snapshot returns copy of maps",
			fields: fields{
				Gauge:   storage.GaugeMap{"g1": 1.5},
				Counter: storage.CounterMap{"c1": 2},
			},
			wantG: storage.GaugeMap{"g1": 1.5},
			wantC: storage.CounterMap{"c1": 2},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			as := &AgentStorage{
				Gauge:   tt.fields.Gauge,
				Counter: tt.fields.Counter,
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
