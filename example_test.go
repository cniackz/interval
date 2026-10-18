// Copyright 2026 Cesar Celis
// SPDX-License-Identifier: Apache-2.0

package interval_test

import (
	"fmt"

	"github.com/cniackz/interval"
)

// A reservoir has between 120 and 125 units of free volume, and it fills at between 3.1 and 3.4
// units per day. The time until it is full is guaranteed to lie in the result, rounding included.
func Example() {
	free, _ := interval.Parse("[120, 125]")
	rate, _ := interval.Parse("[3.1, 3.4]")
	days := free.Div(rate)
	fmt.Printf("between %.4f and %.4f days\n", days.Lo, days.Hi)
	fmt.Println(days.Contains(36), days.Contains(41))
	// Output:
	// between 35.2941 and 40.3226 days
	// true false
}

// A decimal such as 0.1 has no exact float64. Parse encloses the real value between two adjacent
// float64 values; Point would keep only the nearest one.
func ExampleParse() {
	x, _ := interval.Parse("0.1")
	fmt.Println(x.Lo < x.Hi, x.Lo <= 0.1 && 0.1 <= x.Hi)
	fmt.Printf("%.20f\n%.20f\n", x.Lo, x.Hi)
	// Output:
	// true true
	// 0.09999999999999999167
	// 0.10000000000000000555
}

// Exact results stay exact: the bounds move only when rounding happened.
func ExampleInterval_Add() {
	one := interval.Point(1)
	fmt.Println(one.Add(one))
	third := one.Div(interval.Point(3))
	fmt.Println(third.Lo < third.Hi)
	// Output:
	// [2, 2]
	// true
}

// Division by an interval that has zero as an endpoint is unbounded on one side; with zero
// inside, Div returns the hull and DivSplit the two pieces.
func ExampleInterval_DivSplit() {
	x := interval.Point(1)
	fmt.Println(x.Div(interval.Interval{Lo: 0, Hi: 2}))
	fmt.Println(x.Div(interval.Interval{Lo: -1, Hi: 2}))
	lo, hi := x.DivSplit(interval.Interval{Lo: -1, Hi: 2})
	fmt.Println(lo, hi)
	// Output:
	// [0.5, +inf]
	// [entire]
	// [-inf, -1] [0.5, +inf]
}

// The dependency problem: every occurrence of a variable is treated as independent, so the result
// is a valid enclosure but can be wider than the true range.
func Example_dependency() {
	x := interval.Interval{Lo: -1, Hi: 2}
	fmt.Println(x.Sub(x)) // the true range of x - x is [0, 0]
	fmt.Println(x.Mul(x)) // the true range of x * x is [0, 4]
	fmt.Println(x.Sqr())  // Sqr knows both factors are the same variable
	// Output:
	// [-3, 3]
	// [-2, 4]
	// [0, 4]
}

// Ill-formed input never panics: it gives NaI, which propagates.
func ExampleInterval_IsNaI() {
	bad := interval.Interval{Lo: 2, Hi: 1}
	r := bad.Add(interval.Point(1))
	fmt.Println(r.IsNaI(), r)
	_, err := interval.New(2, 1)
	fmt.Println(err)
	// Output:
	// true [nai]
	// interval: invalid bounds
}
