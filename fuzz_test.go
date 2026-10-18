// Copyright 2026 Cesar Celis
// SPDX-License-Identifier: Apache-2.0

package interval

import (
	"math"
	"math/big"
	"strings"
	"testing"
)

// The binary fuzz targets take six raw bit patterns: two intervals and one sample point in each.
// Reinterpreting uint64 values reaches every NaN payload, both zeros, the subnormals and both
// infinities. Each run checks that the operation does not panic, that the result is well-formed,
// and, when the inputs are well-formed and the points are finite members, the inclusion property
// against the exact big.Rat oracle.

func seedBinary(f *testing.F) {
	vals := append([]float64{math.Inf(1), math.Inf(-1), math.NaN()}, specials...)
	for i, a := range vals {
		b := vals[(i*7+3)%len(vals)]
		c := vals[(i*5+1)%len(vals)]
		d := vals[(i*3+2)%len(vals)]
		f.Add(bits(min(a, b)), bits(max(a, b)), bits(min(c, d)), bits(max(c, d)), bits(a), bits(c))
	}
}

func bits(f float64) uint64 { return math.Float64bits(f) }

func wellFormed(t *testing.T, got Interval, ctx ...any) {
	t.Helper()
	if got.kind() == ill && !(math.IsNaN(got.Lo) && math.IsNaN(got.Hi)) {
		t.Fatalf("ill-formed result %#v from %v", got, ctx)
	}
}

func fuzzBinary(f *testing.F, op binop) {
	seedBinary(f)
	f.Fuzz(func(t *testing.T, xl, xh, yl, yh, pa, pb uint64) {
		x := Interval{math.Float64frombits(xl), math.Float64frombits(xh)}
		y := Interval{math.Float64frombits(yl), math.Float64frombits(yh)}
		a, b := math.Float64frombits(pa), math.Float64frombits(pb)
		got := op.iv(x, y)
		wellFormed(t, got, x, y)
		if !x.Contains(a) || !y.Contains(b) {
			return
		}
		ex := op.exact(ratOf(a), ratOf(b))
		if ex == nil {
			return
		}
		if !encloses(got, ex) {
			t.Fatalf("%s(%v, %v) = %v misses %v op %v", op.name, x, y, got, a, b)
		}
	})
}

func FuzzAdd(f *testing.F) { fuzzBinary(f, binops[0]) }
func FuzzSub(f *testing.F) { fuzzBinary(f, binops[1]) }
func FuzzMul(f *testing.F) { fuzzBinary(f, binops[2]) }
func FuzzDiv(f *testing.F) { fuzzBinary(f, binops[3]) }

func FuzzDivSplit(f *testing.F) {
	seedBinary(f)
	f.Fuzz(func(t *testing.T, xl, xh, yl, yh, pa, pb uint64) {
		x := Interval{math.Float64frombits(xl), math.Float64frombits(xh)}
		y := Interval{math.Float64frombits(yl), math.Float64frombits(yh)}
		a, b := math.Float64frombits(pa), math.Float64frombits(pb)
		p, q := x.DivSplit(y)
		wellFormed(t, p, x, y)
		wellFormed(t, q, x, y)
		if !x.Contains(a) || !y.Contains(b) || b == 0 {
			return
		}
		ex := new(big.Rat).Quo(ratOf(a), ratOf(b))
		if !encloses(p, ex) && !encloses(q, ex) {
			t.Fatalf("DivSplit(%v, %v) = %v, %v misses %v / %v", x, y, p, q, a, b)
		}
	})
}

// FuzzUnary covers Sqrt, Sqr, Pown and Neg on one interval, one sample point and an exponent.
func FuzzUnary(f *testing.F) {
	for _, a := range specials {
		f.Add(bits(min(a, -a)), bits(max(a, -a)), bits(a), int8(3))
	}
	f.Add(bits(math.NaN()), bits(1), bits(1), int8(-2))
	f.Fuzz(func(t *testing.T, xl, xh, pa uint64, n int8) {
		x := Interval{math.Float64frombits(xl), math.Float64frombits(xh)}
		a := math.Float64frombits(pa)
		k := int(n % 9)
		sqrt, sqr, pw, neg := x.Sqrt(), x.Sqr(), x.Pown(k), x.Neg()
		for _, got := range []Interval{sqrt, sqr, pw, neg} {
			wellFormed(t, got, x, k)
		}
		if !x.Contains(a) {
			return
		}
		ra := ratOf(a)
		if a >= 0 && !sqrtEncloses(sqrt, a) {
			t.Fatalf("Sqrt(%v) = %v misses sqrt(%v)", x, sqrt, a)
		}
		if !encloses(sqr, new(big.Rat).Mul(ra, ra)) {
			t.Fatalf("Sqr(%v) = %v misses %v^2", x, sqr, a)
		}
		if !encloses(neg, new(big.Rat).Neg(ra)) {
			t.Fatalf("Neg(%v) = %v misses -%v", x, neg, a)
		}
		if a == 0 && k < 0 {
			return
		}
		ex := big.NewRat(1, 1)
		for j := 0; j < abs(k); j++ {
			ex.Mul(ex, ra)
		}
		if k < 0 {
			ex.Inv(ex)
		}
		if !encloses(pw, ex) {
			t.Fatalf("Pown(%v, %d) = %v misses %v^%d", x, k, pw, a, k)
		}
	})
}

// FuzzParse checks that Parse never panics, that every result is well-formed, and that a
// successful parse of a plain decimal contains its exact rational value.
func FuzzParse(f *testing.F) {
	for _, s := range []string{"0.1", "[1, 2]", "[-inf, 3.5e10]", "[empty]", "1e-400", "-1e400", "[2, 1]", "nan", "007.500e+3"} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) {
		got, err := Parse(s)
		wellFormed(t, got, s)
		if err != nil {
			if !got.IsNaI() {
				t.Fatalf("Parse(%q) failed with %v but returned %v", s, err, got)
			}
			return
		}
		if got.IsNaI() {
			t.Fatalf("Parse(%q) returned NaI without an error", s)
		}
		// A plain decimal (no brackets, no infinity): its exact value must be inside.
		if strings.ContainsAny(s, "[]iInN") || len(s) > 100 {
			return
		}
		if i := strings.IndexAny(s, "eE"); i >= 0 && len(strings.TrimLeft(s[i+1:], "+-")) > 3 {
			return // keep the oracle fast: math/big would expand a huge power of ten
		}
		r, ok := new(big.Rat).SetString(strings.TrimSpace(s))
		if ok && !encloses(got, r) {
			t.Fatalf("Parse(%q) = %v misses the exact value %v", s, got, r.FloatString(30))
		}
	})
}
