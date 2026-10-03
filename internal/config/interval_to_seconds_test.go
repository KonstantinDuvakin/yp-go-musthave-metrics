package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIntervalToSeconds(t *testing.T) {
	tests := []struct {
		name     string
		interval string
		want     float64
		wantErr  bool
	}{
		{name: "seconds", interval: "1s", want: 1},
		{name: "milliseconds", interval: "500ms", want: 0.5},
		{name: "combined", interval: "1m30s", want: 90},
		{name: "zero", interval: "0s", want: 0},
		{name: "negative is not rejected here", interval: "-1s", want: -1},
		{name: "number without unit", interval: "10", wantErr: true},
		{name: "garbage", interval: "abc", wantErr: true},
		{name: "empty", interval: "", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := intervalToSeconds(tt.interval, "some_field")

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "some_field")
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}
