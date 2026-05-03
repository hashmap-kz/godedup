package hash

import "testing"

func TestSimilarity(t *testing.T) {
	tests := []struct {
		name string
		a    []uint64
		b    []uint64
		want float64
	}{
		{name: "both empty", a: nil, b: nil, want: 1.0},
		{name: "one empty", a: []uint64{1}, b: nil, want: 0.0},
		{name: "identical", a: []uint64{1, 2, 3}, b: []uint64{1, 2, 3}, want: 1.0},
		{name: "one insertion", a: []uint64{1, 2, 3}, b: []uint64{1, 9, 2, 3}, want: 0.75},
		{name: "one substitution", a: []uint64{1, 2, 3, 4}, b: []uint64{1, 2, 9, 4}, want: 0.75},
		{name: "completely different same length", a: []uint64{1, 2, 3}, b: []uint64{4, 5, 6}, want: 0.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Similarity(&FuncInfo{StmtSeq: tt.a}, &FuncInfo{StmtSeq: tt.b})
			if got != tt.want {
				t.Fatalf("Similarity(%v, %v) = %v, want %v", tt.a, tt.b, got, tt.want)
			}
		})
	}
}

func TestEditDistance(t *testing.T) {
	tests := []struct {
		name string
		a    []uint64
		b    []uint64
		want int
	}{
		{name: "same", a: []uint64{1, 2, 3}, b: []uint64{1, 2, 3}, want: 0},
		{name: "insert", a: []uint64{1, 3}, b: []uint64{1, 2, 3}, want: 1},
		{name: "delete", a: []uint64{1, 2, 3}, b: []uint64{1, 3}, want: 1},
		{name: "substitute", a: []uint64{1, 2, 3}, b: []uint64{1, 9, 3}, want: 1},
		{name: "empty", a: nil, b: []uint64{1, 2}, want: 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := editDistance(tt.a, tt.b); got != tt.want {
				t.Fatalf("editDistance(%v, %v) = %d, want %d", tt.a, tt.b, got, tt.want)
			}
		})
	}
}
