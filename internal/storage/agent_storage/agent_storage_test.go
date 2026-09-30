package agentstorage

import (
	"testing"

	models "github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/model"
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

func TestAgentStorage_CollectMetrics(t *testing.T) {
	type fields struct {
		Gauge   storage.GaugeMap
		Counter storage.CounterMap
	}
	tests := []struct {
		name         string
		fields       fields
		wantNil      bool
		wantGauges   storage.GaugeMap
		wantCounters storage.CounterMap
	}{
		{
			name: "empty storage returns nil",
			fields: fields{
				Gauge:   make(storage.GaugeMap),
				Counter: make(storage.CounterMap),
			},
			wantNil: true,
		},
		{
			name: "only gauges",
			fields: fields{
				Gauge:   storage.GaugeMap{"Alloc": 42.5, "HeapSys": 1.25},
				Counter: make(storage.CounterMap),
			},
			wantGauges:   storage.GaugeMap{"Alloc": 42.5, "HeapSys": 1.25},
			wantCounters: storage.CounterMap{},
		},
		{
			name: "only counters",
			fields: fields{
				Gauge:   make(storage.GaugeMap),
				Counter: storage.CounterMap{"PollCount": 3},
			},
			wantGauges:   storage.GaugeMap{},
			wantCounters: storage.CounterMap{"PollCount": 3},
		},
		{
			name: "gauges and counters",
			fields: fields{
				Gauge:   storage.GaugeMap{"Alloc": 42.5, "RandomValue": 0.5},
				Counter: storage.CounterMap{"PollCount": 7},
			},
			wantGauges:   storage.GaugeMap{"Alloc": 42.5, "RandomValue": 0.5},
			wantCounters: storage.CounterMap{"PollCount": 7},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			as := &AgentStorage{
				Gauge:   tt.fields.Gauge,
				Counter: tt.fields.Counter,
			}

			got := as.CollectMetrics()

			if tt.wantNil {
				if got != nil {
					t.Errorf("CollectMetrics() = %v, want nil", got)
				}
				return
			}

			if len(got) != len(tt.wantGauges)+len(tt.wantCounters) {
				t.Fatalf("CollectMetrics() returned %d metrics, want %d", len(got), len(tt.wantGauges)+len(tt.wantCounters))
			}

			gotGauges := make(storage.GaugeMap)
			gotCounters := make(storage.CounterMap)
			for _, m := range got {
				switch m.MType {
				case models.Gauge:
					if m.Value == nil || m.Delta != nil {
						t.Fatalf("gauge %q: want Value set and Delta nil, got Value=%v Delta=%v", m.ID, m.Value, m.Delta)
					}
					gotGauges[m.ID] = *m.Value
				case models.Counter:
					if m.Delta == nil || m.Value != nil {
						t.Fatalf("counter %q: want Delta set and Value nil, got Value=%v Delta=%v", m.ID, m.Value, m.Delta)
					}
					gotCounters[m.ID] = *m.Delta
				default:
					t.Fatalf("metric %q has unexpected type %q", m.ID, m.MType)
				}
			}

			for name, want := range tt.wantGauges {
				if got, ok := gotGauges[name]; !ok || got != want {
					t.Errorf("gauge %q = %v (present: %v), want %v", name, got, ok, want)
				}
			}
			for name, want := range tt.wantCounters {
				if got, ok := gotCounters[name]; !ok || got != want {
					t.Errorf("counter %q = %v (present: %v), want %v", name, got, ok, want)
				}
			}
		})
	}
}

func TestAgentStorage_CollectMetrics_IndependentOfStorage(t *testing.T) {
	as := NewAgentStorage()
	as.SetGauge("Alloc", 1.5)
	as.AddCounter("PollCount")

	batch := as.CollectMetrics()

	as.SetGauge("Alloc", 999)
	as.AddCounter("PollCount")

	for _, m := range batch {
		switch m.ID {
		case "Alloc":
			if *m.Value != 1.5 {
				t.Errorf("Alloc in batch = %v after storage change, want 1.5", *m.Value)
			}
		case "PollCount":
			if *m.Delta != 1 {
				t.Errorf("PollCount in batch = %v after storage change, want 1", *m.Delta)
			}
		}
	}
}
