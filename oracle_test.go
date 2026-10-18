// Copyright 2026 Cesar Celis
// SPDX-License-Identifier: Apache-2.0

package interval

import (
	"math"
	"math/big"
	"math/rand/v2"
	"testing"
)

// The oracle: exact rational arithmetic in math/big, independent of the code under test.

func ratOf(f float64) *big.Rat { return new(big.Rat).SetFloat64(f) }

// cmpFR compares a float64, possibly infinite, with an exact rational.
func cmpFR(f float64, r *big.Rat) int {
	switch {
	case math.IsInf(f, -1):
		return -1
	case math.IsInf(f, 1):
		return 1
	}
	return ratOf(f).Cmp(r)
}

// isRD reports whether lo is the largest float64 (or -Inf) not above exact.
func isRD(lo float64, exact *big.Rat) bool {
	return cmpFR(lo, exact) <= 0 && cmpFR(math.Nextafter(lo, math.Inf(1)), exact) > 0
}

// isRU reports whether hi is the smallest float64 (or +Inf) not below exact.
func isRU(hi float64, exact *big.Rat) bool {
	return cmpFR(hi, exact) >= 0 && cmpFR(math.Nextafter(hi, math.Inf(-1)), exact) < 0
}

// encloses reports whether lo <= exact <= hi.
func encloses(x Interval, exact *big.Rat) bool {
	return cmpFR(x.Lo, exact) <= 0 && cmpFR(x.Hi, exact) >= 0
}

type binop struct {
	name  string
	iv    func(x, y Interval) Interval
	exact func(a, b *big.Rat) *big.Rat // nil result means undefined (division by zero)
}

var binops = []binop{
	{"Add", Interval.Add, func(a, b *big.Rat) *big.Rat { return new(big.Rat).Add(a, b) }},
	{"Sub", Interval.Sub, func(a, b *big.Rat) *big.Rat { return new(big.Rat).Sub(a, b) }},
	{"Mul", Interval.Mul, func(a, b *big.Rat) *big.Rat { return new(big.Rat).Mul(a, b) }},
	{"Div", Interval.Div, func(a, b *big.Rat) *big.Rat {
		if b.Sign() == 0 {
			return nil
		}
		return new(big.Rat).Quo(a, b)
	}},
}

// sqrtBounds checks that lo is RD(sqrt(a)) and hi is RU(sqrt(a)) by squaring exactly.
func sqrtTight(x Interval, a float64) bool {
	ra := ratOf(a)
	sq := func(f float64) *big.Rat { r := ratOf(f); return r.Mul(r, r) }
	lo, hi := x.Lo, x.Hi
	if lo < 0 || hi < 0 || math.IsInf(lo, 0) || math.IsInf(hi, 0) {
		return false
	}
	nl := math.Nextafter(lo, math.Inf(1))
	ph := math.Nextafter(hi, math.Inf(-1))
	return sq(lo).Cmp(ra) <= 0 && (math.IsInf(nl, 1) || sq(nl).Cmp(ra) > 0) &&
		sq(hi).Cmp(ra) >= 0 && (ph < 0 || sq(ph).Cmp(ra) < 0)
}

func sqrtEncloses(x Interval, a float64) bool {
	if x.Lo < 0 {
		return false
	}
	ra := ratOf(a)
	sq := func(f float64) *big.Rat { r := ratOf(f); return r.Mul(r, r) }
	return sq(x.Lo).Cmp(ra) <= 0 && (math.IsInf(x.Hi, 1) || sq(x.Hi).Cmp(ra) >= 0)
}

// Generators.

var specials = []float64{
	0, math.Copysign(0, -1), 1, -1, 2, 0.5, 3, 0.1, -0.1,
	math.MaxFloat64, -math.MaxFloat64, math.SmallestNonzeroFloat64, -math.SmallestNonzeroFloat64,
	0x1p-1022, -0x1p-1022, 0x1p-900, 0x1p1000, 0x1p-969, 0x1p1023, 0x1.fffffffffffffp1023,
	1 + 0x1p-52, 1 - 0x1p-53, 0x1p53, 0x1p53 + 2,
}

// randFinite draws a finite float64 from a mix of distributions: uniform bit patterns, small
// integers (where exact results are common), values near the underflow and overflow thresholds,
// and special values.
func randFinite(r *rand.Rand) float64 {
	for {
		var f float64
		switch r.IntN(5) {
		case 0, 1:
			f = math.Float64frombits(r.Uint64())
		case 2:
			f = float64(r.IntN(2001) - 1000)
		case 3:
			f = math.Ldexp(r.Float64()+0.5, r.IntN(200)-100)
			if r.IntN(2) == 0 {
				f = -f
			}
		default:
			if r.IntN(2) == 0 {
				f = specials[r.IntN(len(specials))]
			} else {
				f = math.Ldexp(r.Float64()+0.5, []int{-1074, -1060, -1022, -1000, -969, -900, 1000, 1020, 1023}[r.IntN(9)])
			}
		}
		if !math.IsNaN(f) && !math.IsInf(f, 0) {
			return f
		}
	}
}

// randInterval draws a non-empty well-formed interval, sometimes a point, sometimes unbounded.
func randInterval(r *rand.Rand) Interval {
	a, b := randFinite(r), randFinite(r)
	if r.IntN(4) == 0 {
		b = a
	}
	if a > b {
		a, b = b, a
	}
	switch r.IntN(10) {
	case 0:
		a = math.Inf(-1)
	case 1:
		b = math.Inf(1)
	}
	return Interval{a, b}
}

// randMember draws a finite member of a non-empty interval.
func randMember(r *rand.Rand, x Interval) float64 {
	lo, hi := x.Lo, x.Hi
	if math.IsInf(lo, -1) {
		lo = min(hi, 0) - math.Abs(randFinite(r))
		if math.IsInf(lo, 0) || math.IsInf(hi, 1) {
			lo = -math.MaxFloat64
		}
	}
	if math.IsInf(hi, 1) {
		hi = max(lo, 0) + math.Abs(randFinite(r))
		if math.IsInf(hi, 0) {
			hi = math.MaxFloat64
		}
	}
	switch r.IntN(4) {
	case 0:
		return lo
	case 1:
		return hi
	}
	m := lo + r.Float64()*(hi-lo)
	if math.IsNaN(m) || math.IsInf(m, 0) {
		m = lo/2 + hi/2
	}
	return min(max(m, lo), hi)
}

func iterations(t testing.TB, n int) int {
	if testing.Short() {
		return n / 10
	}
	return n
}
