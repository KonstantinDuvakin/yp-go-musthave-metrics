package sender

import "testing"

func TestSendMetrics(t *testing.T) {
	type args struct {
		url string
	}
	tests := []struct {
		name string
		args args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			SendMetrics(tt.args.url)
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
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := UrlBuilder(tt.args.metricType, tt.args.name, tt.args.value); got != tt.want {
				t.Errorf("UrlBuilder() = %v, want %v", got, tt.want)
			}
		})
	}
}
