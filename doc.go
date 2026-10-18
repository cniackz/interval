// Copyright 2026 Cesar Celis
// SPDX-License-Identifier: Apache-2.0

// Package interval implements rigorous interval arithmetic on float64.
//
// An Interval [Lo, Hi] stands for every real number between its bounds. Each operation returns an
// interval that is guaranteed to contain the exact real result for every choice of members of
// its operands, including the effect of floating-point rounding. This is the inclusion property,
// the fundamental theorem of interval arithmetic (R. E. Moore, "Interval Analysis," 1966,
// theorem 3.1; IEEE 1788-2015, clause 10.6):
//
//	if a is in x and b is in y, then the exact a op b is in x.Op(y)
//
// It holds in finite precision: through overflow, underflow, signed zeros and unbounded
// endpoints. Use it to carry guaranteed bounds through a computation, for example from inputs
// known only within a tolerance to a derived quantity.
//
// # Model
//
// The package follows the set-based flavor of IEEE 1788-2015 (clause 7) with float64 bounds, as
// in IEEE 1788.1-2017. An interval is a closed connected set of reals; an infinite bound means the
// interval is unbounded on that side, and infinity itself is never a member. The empty set is an
// interval, and it is what an operation returns when no real result exists, such as the square
// root of [-2, -1] or a division by [0, 0]. Ill-formed input gives NaI ("not an interval"), which
// propagates through every operation. Nothing panics.
//
// # Rounding
//
// Go cannot switch the hardware rounding mode, so the package computes each endpoint by round to
// nearest and finds the exact sign of the rounding error with an error-free transformation:
// TwoSum for addition and subtraction (O. Moller, 1965; D. E. Knuth, TAOCP vol. 2, 4.2.2) and
// a fused multiply-add for multiplication, division and square root (T. J. Dekker, 1971;
// S. Boldo and J.-M. Muller, 2011). The lower bound is then rounded down and the upper bound
// rounded up only when the result was inexact. The basic operations Add, Sub, Mul, Div and Sqrt
// therefore return the tightest float64 bounds, and exact results stay exact: [1, 1] + [1, 1] is
// [2, 2]. Near the underflow and overflow thresholds, where an error-free transformation can lose
// exactness, the sign comes from an exact comparison in math/big instead; that path allocates.
//
// # The dependency problem
//
// Interval arithmetic treats every occurrence of a variable as independent. For x = [1, 2],
// x.Sub(x) is [-1, 1], not [0, 0], and x.Mul(x) is wider than x.Sqr() when x contains zero. The
// result is still a valid enclosure, only wider than the true range. When each variable occurs
// once in an expression, the natural interval extension gives the exact range up to rounding
// (Moore 1966, chapter 3), so rewrite expressions to use each variable once where possible, use
// Sqr and Pown for powers, or split the input intervals into pieces and take the hull of the
// results. The bound is also worst case: it says which results are possible, not which are likely.
//
// # Out of scope
//
// Elementary functions such as exp and log are not provided: the Go math package documents no
// error bound for them, and an enclosure built on them would rest on an unproven assumption.
// Decorations other than NaI, interval vectors and matrices, and precisions other than float64
// are also out of scope.
//
// # References
//
//   - R. E. Moore, "Interval Analysis," Prentice-Hall, 1966.
//   - R. E. Moore, R. B. Kearfott, M. J. Cloud, "Introduction to Interval Analysis," SIAM, 2009.
//   - IEEE Std 1788-2015, "IEEE Standard for Interval Arithmetic."
//   - IEEE Std 1788.1-2017, "IEEE Standard for Interval Arithmetic (Simplified)."
//   - IEEE Std 754-2019, "IEEE Standard for Floating-Point Arithmetic."
//   - T. Hickey, Q. Ju, M. H. van Emden, "Interval Arithmetic: From Principles to Implementation,"
//     Journal of the ACM 48(5), 2001.
//   - O. Moller, "Quasi Double-Precision in Floating Point Addition," BIT 5, 1965.
//   - D. E. Knuth, "The Art of Computer Programming, Vol. 2," 3rd ed., 1997.
//   - T. J. Dekker, "A Floating-Point Technique for Extending the Available Precision,"
//     Numerische Mathematik 18, 1971.
//   - G. Bohlender, W. Walter, P. Kornerup, D. W. Matula, "Semantics for Exact Floating Point
//     Operations," 10th IEEE Symposium on Computer Arithmetic, 1991.
//   - S. Boldo, J.-M. Muller, "Exact and Approximated Error of the FMA," IEEE Transactions on
//     Computers 60(2), 2011.
//   - J.-M. Muller et al., "Handbook of Floating-Point Arithmetic," 2nd ed., 2018.
package interval
