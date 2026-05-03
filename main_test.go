package main

import (
	"reflect"
	"testing"
)

func TestTrimSuffix(t *testing.T) {
	tests := []struct {
		name   string
		s      string
		suffix string
		want   string
	}{
		{name: "has suffix", s: "./pkg/...", suffix: "/...", want: "./pkg"},
		{name: "without suffix", s: "./pkg", suffix: "/...", want: "./pkg"},
		{name: "shorter than suffix", s: ".", suffix: "/...", want: "."},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := trimSuffix(tt.s, tt.suffix); got != tt.want {
				t.Fatalf("trimSuffix(%q, %q) = %q, want %q", tt.s, tt.suffix, got, tt.want)
			}
		})
	}
}

func TestExpandPaths(t *testing.T) {
	tests := []struct {
		name string
		in   []string
		want []string
	}{
		{name: "dot slash ellipsis", in: []string{"./..."}, want: []string{"."}},
		{name: "ellipsis", in: []string{"..."}, want: []string{"."}},
		{name: "strip trailing ellipsis", in: []string{"./pkg/..."}, want: []string{"./pkg"}},
		{name: "mixed", in: []string{"./...", "./cmd/...", "internal"}, want: []string{".", "./cmd", "internal"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := expandPaths(tt.in); !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("expandPaths(%v) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}
