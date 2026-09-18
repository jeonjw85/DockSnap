package main

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func Test_formatElapsed_formats_duration_boundaries(t *testing.T) {
	tests := []struct {
		name string
		in   time.Duration
		want string
	}{
		{name: "negative", in: -time.Millisecond, want: "0ms"},
		{name: "milliseconds", in: 12 * time.Millisecond, want: "12ms"},
		{name: "seconds", in: 1200 * time.Millisecond, want: "1.2s"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, formatElapsed(tt.in))
		})
	}
}

func Test_formatLine_formats_product_output(t *testing.T) {
	require.Equal(t, "seed1 1 volumes 5B 12ms", formatLine("seed1", 1, 5, 12*time.Millisecond))
}
