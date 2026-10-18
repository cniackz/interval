// Copyright 2026 Cesar Celis
// SPDX-License-Identifier: Apache-2.0

package interval

import (
	"errors"
	"math"
)

// Interval is a closed interval [Lo, Hi] of real numbers. Compare intervals with Equal, not ==:
// == is false for two NaI values and distinguishes -0 from +0, which a set does not.
//
// A well-formed Interval is one of:
//   - non-empty: Lo <= Hi, Lo != +Inf, Hi != -Inf and no NaN. An infinite endpoint means the
//     interval is unbounded on that side; infinity itself is never a member.
//   - Empty: Lo == +Inf and Hi == -Inf, the canonical empty set.
//   - NaI ("not an interval"): both endpoints NaN.
//
// Any other value, such as Interval{2, 1}, is ill-formed. Every operation maps ill-formed input to
// NaI, so a bad literal can produce NaI but never a wrong bound. NaI propagates through every
// operation and takes precedence over Empty (IEEE 1788-2015, clause 12.2, the "ill" decoration).
//
// The zero value is [0, 0], a well-formed point interval.
//
// Results store a zero lower bound as -0 and a zero upper bound as +0, the convention of
// IEEE 1788-2015, clause 12.12.8, for inf and sup. The sign of zero carries no other meaning.
type Interval struct {
	Lo, Hi float64
}

// ErrInvalid is returned by New and Parse for bounds that do not form an interval.
var ErrInvalid = errors.New("interval: invalid bounds")

// New returns [lo, hi]. It returns NaI and ErrInvalid if lo > hi, if either bound is NaN, if
// lo == +Inf or if hi == -Inf. Use Empty for the empty set.
func New(lo, hi float64) (Interval, error) {
	x := Interval{lo, hi}
	if x.kind() != nonEmpty {
		return nai(), ErrInvalid
	}
	return mk(lo, hi), nil
}

// Point returns the degenerate interval [x, x]. A NaN or infinite x gives NaI, because infinity
// is not a real number. Point(0.1) holds the float64 nearest to 0.1, not the real 0.1: use Parse
// for decimal input.
func Point(x float64) Interval {
	if math.IsNaN(x) || math.IsInf(x, 0) {
		return nai()
	}
	return mk(x, x)
}

// Empty returns the empty interval.
func Empty() Interval { return Interval{math.Inf(1), math.Inf(-1)} }

// Entire returns the whole real line, [-Inf, +Inf].
func Entire() Interval { return Interval{math.Inf(-1), math.Inf(1)} }

func nai() Interval { return Interval{math.NaN(), math.NaN()} }

// mk builds a result and normalizes the sign of zero endpoints.
func mk(lo, hi float64) Interval {
	if lo == 0 {
		lo = math.Copysign(0, -1)
	}
	if hi == 0 {
		hi = 0
	}
	return Interval{lo, hi}
}

type kind uint8

const (
	ill kind = iota
	empty
	nonEmpty
)

func (x Interval) kind() kind {
	if x.Lo <= x.Hi && !math.IsInf(x.Lo, 1) && !math.IsInf(x.Hi, -1) {
		return nonEmpty // the comparison is false for NaN
	}
	if math.IsInf(x.Lo, 1) && math.IsInf(x.Hi, -1) {
		return empty
	}
	return ill
}

// IsEmpty reports whether x is the empty interval.
func (x Interval) IsEmpty() bool { return x.kind() == empty }

// IsEntire reports whether x is the whole real line.
func (x Interval) IsEntire() bool { return math.IsInf(x.Lo, -1) && math.IsInf(x.Hi, 1) }

// IsNaI reports whether x is NaI or any other ill-formed value.
func (x Interval) IsNaI() bool { return x.kind() == ill }

// Contains reports whether the real number v is a member of x. It is false for NaN and for the
// infinities, which are not real numbers.
func (x Interval) Contains(v float64) bool {
	return x.kind() == nonEmpty && x.Lo <= v && v <= x.Hi && !math.IsInf(v, 0)
}

// Subset reports whether every member of x is a member of y. The empty set is a subset of every
// interval.
func (x Interval) Subset(y Interval) bool {
	kx, ky := x.kind(), y.kind()
	switch {
	case kx == ill || ky == ill:
		return false
	case kx == empty:
		return true
	case ky == empty:
		return false
	}
	return y.Lo <= x.Lo && x.Hi <= y.Hi
}

// Disjoint reports whether x and y have no member in common.
func (x Interval) Disjoint(y Interval) bool {
	kx, ky := x.kind(), y.kind()
	switch {
	case kx == ill || ky == ill:
		return false
	case kx == empty || ky == empty:
		return true
	}
	return x.Hi < y.Lo || y.Hi < x.Lo
}

// Equal reports whether x and y are the same set. It is false when either is NaI.
func (x Interval) Equal(y Interval) bool {
	kx, ky := x.kind(), y.kind()
	if kx == ill || ky == ill || kx != ky {
		return false
	}
	return kx == empty || (x.Lo == y.Lo && x.Hi == y.Hi)
}

// Intersect returns the set of reals in both x and y.
func (x Interval) Intersect(y Interval) Interval {
	if r, done := pre2(x, y); done {
		return r
	}
	lo, hi := max(x.Lo, y.Lo), min(x.Hi, y.Hi)
	if lo > hi {
		return Empty()
	}
	return mk(lo, hi)
}

// Hull returns the smallest interval that contains both x and y. The empty set is its identity.
func (x Interval) Hull(y Interval) Interval {
	kx, ky := x.kind(), y.kind()
	switch {
	case kx == ill || ky == ill:
		return nai()
	case kx == empty:
		return y
	case ky == empty:
		return x
	}
	return mk(min(x.Lo, y.Lo), max(x.Hi, y.Hi))
}

// Width returns an upper bound of Hi - Lo (the difference rounded up). It is +Inf for an
// unbounded interval and NaN for Empty and NaI (IEEE 1788-2015, clause 10.5.9).
func (x Interval) Width() float64 {
	if x.kind() != nonEmpty {
		return math.NaN()
	}
	return subUp(x.Hi, x.Lo)
}

// Mid returns a float64 member of x close to its midpoint. Following IEEE 1788-2015,
// clause 12.12.8, Mid(Entire) is 0, Mid([-Inf, b]) is -MaxFloat64 and Mid([a, +Inf]) is
// +MaxFloat64. It is NaN for Empty and NaI.
func (x Interval) Mid() float64 {
	if x.kind() != nonEmpty {
		return math.NaN()
	}
	lo, hi := x.Lo, x.Hi
	switch {
	case math.IsInf(lo, -1) && math.IsInf(hi, 1):
		return 0
	case math.IsInf(lo, -1):
		return -math.MaxFloat64
	case math.IsInf(hi, 1):
		return math.MaxFloat64
	case lo == -hi:
		return 0
	}
	m := (lo + hi) / 2
	if math.IsInf(m, 0) {
		m = lo/2 + hi/2
	}
	return min(max(m, lo), hi)
}

// Rad returns r such that [Mid()-r, Mid()+r] contains x, rounded up. It is +Inf for an unbounded
// interval and NaN for Empty and NaI.
func (x Interval) Rad() float64 {
	if x.kind() != nonEmpty {
		return math.NaN()
	}
	m := x.Mid()
	return max(subUp(m, x.Lo), subUp(x.Hi, m))
}

// Mag returns the largest absolute value of a member of x, or NaN for Empty and NaI.
func (x Interval) Mag() float64 {
	if x.kind() != nonEmpty {
		return math.NaN()
	}
	return max(math.Abs(x.Lo), math.Abs(x.Hi))
}

// Mig returns the smallest absolute value of a member of x, or NaN for Empty and NaI.
func (x Interval) Mig() float64 {
	if x.kind() != nonEmpty {
		return math.NaN()
	}
	if x.Lo <= 0 && 0 <= x.Hi {
		return 0
	}
	return min(math.Abs(x.Lo), math.Abs(x.Hi))
}

// pre2 handles NaI and Empty operands of a binary operation. When done is true, r is the result.
func pre2(x, y Interval) (r Interval, done bool) {
	kx, ky := x.kind(), y.kind()
	if kx == ill || ky == ill {
		return nai(), true
	}
	if kx == empty || ky == empty {
		return Empty(), true
	}
	return Interval{}, false
}
