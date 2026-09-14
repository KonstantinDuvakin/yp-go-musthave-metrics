package buildinfo

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFprintBuildInfo_AllSet(t *testing.T) {
	var buf bytes.Buffer
	PrintBuildInfo(&buf, "v1.2.3", "2026/09/14 15:00:00", "98a9d3e")

	require.Equal(t,
		"Build version: v1.2.3\nBuild date: 2026/09/14 15:00:00\nBuild commit: 98a9d3e\n",
		buf.String())
}

func TestFprintBuildInfo_AllEmpty(t *testing.T) {
	var buf bytes.Buffer
	PrintBuildInfo(&buf, "", "", "")

	require.Equal(t,
		"Build version: N/A\nBuild date: N/A\nBuild commit: N/A\n",
		buf.String(),
		"незаданные значения должны заменяться на N/A")
}

func TestFprintBuildInfo_PartiallySet(t *testing.T) {
	var buf bytes.Buffer
	PrintBuildInfo(&buf, "v1.0.0", "", "abc123")

	require.Equal(t,
		"Build version: v1.0.0\nBuild date: N/A\nBuild commit: abc123\n",
		buf.String(),
		"на N/A заменяются только пустые поля, остальные выводятся как есть")
}

func TestValueOrNA(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "пустая строка", in: "", want: NA},
		{name: "непустая строка", in: "v1", want: "v1"},
		{name: "пробел не считается пустым", in: " ", want: " "},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, valueOrNA(tt.in))
		})
	}
}
