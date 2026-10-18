// Copyright 2026 Cesar Celis
// SPDX-License-Identifier: Apache-2.0

package interval

import (
	"math"
	"math/big"
)

// Directed rounding without control of the hardware rounding mode.
//
// Each primitive below computes r, the round-to-nearest result of one IEEE 754 operation, and d,
// the sign of (exact - r). From (r, d) the directed roundings follow:
//
//	RD(exact) = r if d >= 0, else the float64 just below r
//	RU(exact) = r if d <= 0, else the float64 just above r
//
// This is exact directed rounding, not a one-ulp widening: when r is exact both bounds equal r.
//
// The sign d comes from an error-free transformation when its exactness preconditions hold, and
// from an exact comparison in math/big otherwise (operands or results near the underflow or
// overflow thresholds). The math/big path allocates; the error-free path does not.
//
// References:
//   - TwoSum: O. Moller, "Quasi Double-Precision in Floating Point Addition," BIT 5, 1965;
//     D. E. Knuth, TAOCP vol. 2, 3rd ed., section 4.2.2, theorem B. Exact for any finite operands
//     whose sum does not overflow, subnormal results included.
//   - Product error by FMA: T. J. Dekker, "A Floating-Point Technique for Extending the Available
//     Precision," Numer. Math. 18, 1971; S. Boldo and J.-M. Muller, "Exact and Approximated Error
//     of the FMA," IEEE Trans. Computers 60(2), 2011. a*b - RN(a*b) is representable when
//     e_a + e_b >= e_min + p - 1 = -970, which |RN(a*b)| >= 2^-900 guarantees.
//   - Division remainder and square-root residual by FMA: G. Bohlender, W. Walter, P. Kornerup
//     and D. W. Matula, "Semantics for Exact Floating Point Operations," ARITH 10, 1991;
//     J.-M. Muller et al., "Handbook of Floating-Point Arithmetic," 2nd ed., 2018, section 4.4.
//     a - RN(a/b)*b and x - RN(sqrt(x))^2 are representable when no underflow occurs, which the
//     2^-900 and 2^1000 guards below ensure.
//
// Go may fuse x*y + z into one FMA ("Arithmetic operators" in the Go specification). Products
// whose rounding matters are therefore written as float64(x * y), which forces the rounding, and
// fused operations are always explicit math.FMA calls.

const (
	tiny = 0x1p-900  // below this magnitude an error-free transformation may underflow
	huge = 0x1p+1000 // above this magnitude an intermediate step may overflow
)

func sign(f float64) int {
	switch {
	case f > 0:
		return 1
	case f < 0:
		return -1
	}
	return 0
}

// rd and ru turn (r, d) into the directed roundings. They also cover overflow: a finite
// operation that rounds to +Inf has d = -1, so rd gives MaxFloat64 and ru keeps +Inf.
func rd(r float64, d int) float64 {
	if d < 0 {
		return math.Nextafter(r, math.Inf(-1))
	}
	return r
}

func ru(r float64, d int) float64 {
	if d > 0 {
		return math.Nextafter(r, math.Inf(1))
	}
	return r
}

// addDir returns RN(a+b) and the sign of the rounding error. The caller never passes opposite
// infinities (well-formed endpoints exclude them).
func addDir(a, b float64) (float64, int) {
	r := a + b
	if math.IsInf(a, 0) || math.IsInf(b, 0) {
		return r, 0
	}
	if math.IsInf(r, 0) {
		return r, -sign(r)
	}
	if math.Abs(a) < huge && math.Abs(b) < huge {
		bb := r - a
		e := (a - (r - bb)) + (b - bb)
		return r, sign(e)
	}
	var x, y, z, w big.Float
	x.SetFloat64(a)
	y.SetFloat64(b)
	z.SetPrec(2200).Add(&x, &y) // wide enough to hold any exact sum of two float64 values
	w.SetFloat64(r)
	return r, z.Cmp(&w)
}

// mulDir returns RN(a*b) and the sign of the rounding error, with the set-based rule that zero
// times an infinity is zero (IEEE 1788-2015, clause 10.5.3).
func mulDir(a, b float64) (float64, int) {
	if a == 0 || b == 0 {
		return 0, 0
	}
	r := float64(a * b)
	if math.IsInf(a, 0) || math.IsInf(b, 0) {
		return r, 0
	}
	if math.IsInf(r, 0) {
		return r, -sign(r)
	}
	if math.Abs(r) >= tiny {
		return r, sign(math.FMA(a, b, -r))
	}
	var x, y, z, w big.Float
	x.SetFloat64(a)
	y.SetFloat64(b)
	z.SetPrec(106).Mul(&x, &y) // a 53 by 53 bit product is exact in 106 bits
	w.SetFloat64(r)
	return r, z.Cmp(&w)
}

// divDir returns RN(a/b) and the sign of the rounding error. The caller never passes b == 0 or
// two infinities.
func divDir(a, b float64) (float64, int) {
	r := a / b
	if a == 0 || math.IsInf(a, 0) || math.IsInf(b, 0) {
		return r, 0
	}
	if math.IsInf(r, 0) {
		return r, -sign(r)
	}
	aa, ab := math.Abs(a), math.Abs(b)
	if aa >= tiny && aa <= huge && ab >= tiny && ab <= huge && math.Abs(r) >= tiny {
		// a/b - r = (a - r*b)/b, so the error has the sign of the remainder times the sign of b.
		return r, sign(math.FMA(-r, b, a)) * sign(b)
	}
	var x, y, z, w big.Float
	x.SetFloat64(r)
	y.SetFloat64(b)
	z.SetPrec(106).Mul(&x, &y)
	w.SetFloat64(a)
	return r, w.Cmp(&z) * sign(b)
}

// sqrtDir returns RN(sqrt(x)) and the sign of the rounding error, for x >= 0.
func sqrtDir(x float64) (float64, int) {
	r := math.Sqrt(x)
	if x == 0 || math.IsInf(x, 1) {
		return r, 0
	}
	if x >= tiny && x <= huge {
		return r, sign(math.FMA(-r, r, x))
	}
	var y, z, w big.Float
	y.SetFloat64(r)
	z.SetPrec(106).Mul(&y, &y)
	w.SetFloat64(x)
	return r, w.Cmp(&z)
}

func addDown(a, b float64) float64 { return rd(addDir(a, b)) }
func addUp(a, b float64) float64   { return ru(addDir(a, b)) }
func subDown(a, b float64) float64 { return rd(addDir(a, -b)) }
func subUp(a, b float64) float64   { return ru(addDir(a, -b)) }
func mulDown(a, b float64) float64 { return rd(mulDir(a, b)) }
func mulUp(a, b float64) float64   { return ru(mulDir(a, b)) }
func divDown(a, b float64) float64 { return rd(divDir(a, b)) }
func divUp(a, b float64) float64   { return ru(divDir(a, b)) }
func sqrtDown(x float64) float64   { return rd(sqrtDir(x)) }
func sqrtUp(x float64) float64     { return ru(sqrtDir(x)) }

// powDown and powUp bound a^n for a >= 0 and n >= 1 by binary exponentiation. Every factor is
// non-negative, so multiplying lower bounds rounded down gives a lower bound, and the same holds
// for upper bounds rounded up (Moore 1966, chapter 2, monotonicity of the product on [0, +Inf]).
func powDown(a float64, n uint64) float64 {
	r, b := 1.0, a
	for {
		if n&1 == 1 {
			r = mulDown(r, b)
		}
		n >>= 1
		if n == 0 {
			return r
		}
		b = mulDown(b, b)
	}
}

func powUp(a float64, n uint64) float64 {
	r, b := 1.0, a
	for {
		if n&1 == 1 {
			r = mulUp(r, b)
		}
		n >>= 1
		if n == 0 {
			return r
		}
		b = mulUp(b, b)
	}
}
