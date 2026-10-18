// Copyright 2026 Cesar Celis
// SPDX-License-Identifier: Apache-2.0

package interval

import "math"

// Add returns x + y = {a + b : a in x, b in y}, rounded outward:
// [a, b] + [c, d] = [RD(a+c), RU(b+d)] (Moore 1966, chapter 2).
func (x Interval) Add(y Interval) Interval {
	if r, done := pre2(x, y); done {
		return r
	}
	return mk(addDown(x.Lo, y.Lo), addUp(x.Hi, y.Hi))
}

// Sub returns x - y = {a - b : a in x, b in y}, rounded outward:
// [a, b] - [c, d] = [RD(a-d), RU(b-c)] (Moore 1966, chapter 2).
func (x Interval) Sub(y Interval) Interval {
	if r, done := pre2(x, y); done {
		return r
	}
	return mk(subDown(x.Lo, y.Hi), subUp(x.Hi, y.Lo))
}

// Neg returns -x = [-Hi, -Lo]. Negation is exact.
func (x Interval) Neg() Interval {
	switch x.kind() {
	case ill:
		return nai()
	case empty:
		return Empty()
	}
	return mk(-x.Hi, -x.Lo)
}

// class is the sign class of a non-empty interval, as in Hickey, Ju and van Emden,
// "Interval Arithmetic: From Principles to Implementation," J. ACM 48(5), 2001, section 4.
type class uint8

const (
	clZ class = iota // [0, 0]
	clN              // Hi <= 0, Lo < 0
	clM              // Lo < 0 < Hi
	clP              // Lo >= 0, Hi > 0
)

func classOf(x Interval) class {
	switch {
	case x.Lo == 0 && x.Hi == 0:
		return clZ
	case x.Hi <= 0:
		return clN
	case x.Lo >= 0:
		return clP
	}
	return clM
}

// Mul returns x * y = {a * b : a in x, b in y}, rounded outward.
//
// The endpoints come from the nine-case sign table of Hickey, Ju and van Emden (2001,
// section 4), so only two endpoint products are computed except when both operands contain zero.
// Endpoint products use the set-based rule 0 * (+-Inf) = 0, because every real times zero is
// zero (IEEE 1788-2015, clause 10.5.3); for example [0, 0] * Entire = [0, 0].
func (x Interval) Mul(y Interval) Interval {
	if r, done := pre2(x, y); done {
		return r
	}
	cx, cy := classOf(x), classOf(y)
	if cx == clZ || cy == clZ {
		return mk(0, 0)
	}
	a, b, c, d := x.Lo, x.Hi, y.Lo, y.Hi
	switch cx {
	case clP:
		switch cy {
		case clP:
			return mk(mulDown(a, c), mulUp(b, d))
		case clM:
			return mk(mulDown(b, c), mulUp(b, d))
		default: // clN
			return mk(mulDown(b, c), mulUp(a, d))
		}
	case clM:
		switch cy {
		case clP:
			return mk(mulDown(a, d), mulUp(b, d))
		case clM:
			return mk(min(mulDown(a, d), mulDown(b, c)), max(mulUp(a, c), mulUp(b, d)))
		default: // clN
			return mk(mulDown(b, c), mulUp(a, c))
		}
	default: // clN
		switch cy {
		case clP:
			return mk(mulDown(a, d), mulUp(b, c))
		case clM:
			return mk(mulDown(a, d), mulUp(a, c))
		default: // clN
			return mk(mulDown(b, d), mulUp(a, c))
		}
	}
}

// Div returns the smallest interval that contains x / y = {a / b : a in x, b in y, b != 0},
// rounded outward. The result is Empty when y is [0, 0], because no real b is available. When zero
// is an endpoint of y the result is unbounded on one side ([1, 2] / [0, 1] = [1, +Inf],
// [1, 2] / [-1, 0] = [-Inf, -1]). When zero is inside y the set x / y can have two pieces and Div
// returns their hull, often Entire; DivSplit returns the pieces.
//
// The case table is that of Hickey, Ju and van Emden (2001, section 5) and IEEE 1788-2015,
// clause 10.5.3. No case divides zero by zero or infinity by infinity.
func (x Interval) Div(y Interval) Interval {
	if r, done := pre2(x, y); done {
		return r
	}
	cy := classOf(y)
	if cy == clZ {
		return Empty()
	}
	cx := classOf(x)
	if cx == clZ {
		return mk(0, 0)
	}
	a, b, c, d := x.Lo, x.Hi, y.Lo, y.Hi
	inf := math.Inf(1)
	switch {
	case cy == clM:
		return Entire()
	case c > 0: // y positive, zero excluded
		switch cx {
		case clP:
			return mk(divDown(a, d), divUp(b, c))
		case clM:
			return mk(divDown(a, c), divUp(b, c))
		default: // clN
			return mk(divDown(a, c), divUp(b, d))
		}
	case d < 0: // y negative, zero excluded
		switch cx {
		case clP:
			return mk(divDown(b, d), divUp(a, c))
		case clM:
			return mk(divDown(b, d), divUp(a, d))
		default: // clN
			return mk(divDown(b, c), divUp(a, d))
		}
	case c == 0: // y = [0, d] with d > 0
		switch {
		case cx == clM:
			return Entire()
		case a > 0:
			return mk(divDown(a, d), inf)
		case a == 0: // x = [0, b], b > 0
			return mk(0, inf)
		case b == 0: // x = [a, 0], a < 0
			return mk(-inf, 0)
		default: // b < 0
			return mk(-inf, divUp(b, d))
		}
	default: // y = [c, 0] with c < 0
		switch {
		case cx == clM:
			return Entire()
		case a > 0:
			return mk(-inf, divUp(a, c))
		case a == 0:
			return mk(-inf, 0)
		case b == 0:
			return mk(0, inf)
		default: // b < 0
			return mk(divDown(b, c), inf)
		}
	}
}

// DivSplit returns x / y as at most two intervals, the two-output division of
// IEEE 1788-2015, clause 10.5.5 (mulRevToPair), used by interval Newton methods (Moore, Kearfott
// and Cloud, "Introduction to Interval Analysis," SIAM 2009, chapter 8). When x excludes zero and
// zero is strictly inside y, the first result is the lower piece and the second the upper piece;
// for example [1, 2] / [-1, 1] gives [-Inf, -1] and [1, +Inf]. The real pieces are disjoint;
// after outward rounding they can share an endpoint when a quotient underflows. In every other case the first result
// is x.Div(y) and the second is Empty (or NaI for NaI input).
func (x Interval) DivSplit(y Interval) (Interval, Interval) {
	if r, done := pre2(x, y); done {
		return r, r
	}
	if classOf(y) == clM {
		inf := math.Inf(1)
		a, b, c, d := x.Lo, x.Hi, y.Lo, y.Hi
		switch {
		case a > 0:
			return mk(-inf, divUp(a, c)), mk(divDown(a, d), inf)
		case b < 0:
			return mk(-inf, divUp(b, d)), mk(divDown(b, c), inf)
		}
	}
	return x.Div(y), Empty()
}

// Sqr returns {a * a : a in x}. It is tighter than x.Mul(x), which treats the two factors as
// independent: [-1, 1].Sqr() is [0, 1] while [-1, 1].Mul([-1, 1]) is [-1, 1].
// Empty gives Empty and NaI gives NaI.
func (x Interval) Sqr() Interval {
	switch x.kind() {
	case ill:
		return nai()
	case empty:
		return Empty()
	}
	lo, hi := x.Mig(), x.Mag()
	return mk(mulDown(lo, lo), mulUp(hi, hi))
}

// Pown returns {a^n : a in x} for an integer n, rounded outward (IEEE 1788-2015, clause 9.1,
// pown). Pown(0) is [1, 1] for every non-empty x, including [0, 0]. For n > 0 the result is tight
// up to the rounding of each multiplication; for n < 0 it is [1, 1] divided by x.Pown(-n), so
// [0, 0].Pown(-1) is Empty and [-1, 1].Pown(-2) is [1, +Inf]. Empty gives Empty and NaI gives NaI
// for every n.
func (x Interval) Pown(n int) Interval {
	switch x.kind() {
	case ill:
		return nai()
	case empty:
		return Empty()
	}
	switch {
	case n == 0:
		return mk(1, 1)
	case n > 0:
		return x.powPos(uint64(n))
	}
	// -n overflows for the most negative int, so compute |n| as an unsigned value.
	return mk(1, 1).Div(x.powPos(uint64(-(n + 1)) + 1))
}

// powPos returns x^n for a non-empty x and n >= 1, using monotonicity on each sign piece.
func (x Interval) powPos(n uint64) Interval {
	if n%2 == 0 {
		lo, hi := x.Mig(), x.Mag()
		return mk(powDown(lo, n), powUp(hi, n))
	}
	var lo, hi float64
	if x.Lo >= 0 {
		lo = powDown(x.Lo, n)
	} else {
		lo = -powUp(-x.Lo, n)
	}
	if x.Hi >= 0 {
		hi = powUp(x.Hi, n)
	} else {
		hi = -powDown(-x.Hi, n)
	}
	return mk(lo, hi)
}

// Sqrt returns {sqrt(a) : a in x, a >= 0}, rounded outward: the domain is restricted to
// [0, +Inf], so Sqrt([-1, 4]) is [0, 2] and Sqrt([-2, -1]) is Empty (IEEE 1788-2015, clause 9.1).
// IEEE 754 makes the square root correctly rounded, so the bounds are tight.
func (x Interval) Sqrt() Interval {
	switch x.kind() {
	case ill:
		return nai()
	case empty:
		return Empty()
	}
	if x.Hi < 0 {
		return Empty()
	}
	return mk(sqrtDown(max(x.Lo, 0)), sqrtUp(x.Hi))
}
