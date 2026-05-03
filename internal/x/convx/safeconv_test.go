package convx

import (
	"go/token"
	"testing"
)

func TestToUint64SignedIntegers(t *testing.T) {
	tests := []struct {
		name string
		in   int
		want uint64
	}{
		{name: "positive", in: 42, want: 42},
		{name: "zero", in: 0, want: 0},
		{name: "negative clamps to zero", in: -7, want: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ToUint64(tt.in); got != tt.want {
				t.Fatalf("ToUint64(%d) = %d, want %d", tt.in, got, tt.want)
			}
		})
	}
}

func TestToUint64AcceptsTokenToken(t *testing.T) {
	if got := ToUint64(token.ADD); got != uint64(token.ADD) {
		t.Fatalf("ToUint64(token.ADD) = %d, want %d", got, token.ADD)
	}
}
