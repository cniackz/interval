// Copyright 2026 Cesar Celis
// SPDX-License-Identifier: Apache-2.0

package interval

import "testing"

// Benchmarks use bounded positive operands, the common case. sink keeps results alive.
var (
	sink  Interval
	sinkF float64
	x0    = Interval{1.1, 1.3}
	y0    = Interval{2.7, 3.9}
)

func BenchmarkAdd(b *testing.B) {
	for b.Loop() {
		sink = x0.Add(y0)
	}
}

func BenchmarkSub(b *testing.B) {
	for b.Loop() {
		sink = x0.Sub(y0)
	}
}

func BenchmarkMul(b *testing.B) {
	for b.Loop() {
		sink = x0.Mul(y0)
	}
}

func BenchmarkMulMixed(b *testing.B) {
	x, y := Interval{-1.1, 1.3}, Interval{-2.7, 3.9}
	for b.Loop() {
		sink = x.Mul(y)
	}
}

func BenchmarkDiv(b *testing.B) {
	for b.Loop() {
		sink = x0.Div(y0)
	}
}

func BenchmarkSqrt(b *testing.B) {
	for b.Loop() {
		sink = y0.Sqrt()
	}
}

func BenchmarkSqr(b *testing.B) {
	for b.Loop() {
		sink = x0.Sqr()
	}
}

func BenchmarkPown5(b *testing.B) {
	for b.Loop() {
		sink = x0.Pown(5)
	}
}

func BenchmarkIntersect(b *testing.B) {
	for b.Loop() {
		sink = x0.Intersect(y0)
	}
}

func BenchmarkMid(b *testing.B) {
	for b.Loop() {
		sinkF = y0.Mid()
	}
}

func BenchmarkParse(b *testing.B) {
	for b.Loop() {
		sink, _ = Parse("[0.1, 0.2]")
	}
}

func BenchmarkString(b *testing.B) {
	x, _ := Parse("[0.1, 0.2]")
	var s string
	for b.Loop() {
		s = x.String()
	}
	_ = s
}

// BenchmarkPlainAdd is the reference for the cost of rigor: two float64 additions.
func BenchmarkPlainAdd(b *testing.B) {
	var lo, hi float64
	for b.Loop() {
		lo, hi = x0.Lo+y0.Lo, x0.Hi+y0.Hi
	}
	sinkF = lo + hi
}

// TestZeroAllocs pins the zero-allocation promise for the arithmetic, set and query methods.
func TestZeroAllocs(t *testing.T) {
	if testing.CoverMode() != "" {
		t.Skip("coverage instrumentation allocates")
	}
	fns := map[string]func(){
		"Add":       func() { sink = x0.Add(y0) },
		"Sub":       func() { sink = x0.Sub(y0) },
		"Mul":       func() { sink = x0.Mul(y0) },
		"Div":       func() { sink = x0.Div(y0) },
		"DivSplit":  func() { sink, _ = x0.DivSplit(Interval{-1, 2}) },
		"Sqrt":      func() { sink = y0.Sqrt() },
		"Sqr":       func() { sink = x0.Sqr() },
		"Pown":      func() { sink = x0.Pown(-5) },
		"Neg":       func() { sink = x0.Neg() },
		"Intersect": func() { sink = x0.Intersect(y0) },
		"Hull":      func() { sink = x0.Hull(y0) },
		"Width":     func() { sinkF = y0.Width() },
		"Mid":       func() { sinkF = y0.Mid() },
		"Rad":       func() { sinkF = y0.Rad() },
		"Subset":    func() { _ = x0.Subset(y0) },
	}
	for name, fn := range fns {
		if n := testing.AllocsPerRun(100, fn); n != 0 {
			t.Errorf("%s allocates %v times per call", name, n)
		}
	}
}
