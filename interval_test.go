// Copyright 2026 Cesar Celis
// SPDX-License-Identifier: Apache-2.0

package interval

import (
	"errors"
	"math"
	"math/big"
	"testing"
)

var (
	inf  = math.Inf(1)
	ninf = math.Inf(-1)
)

func iv(lo, hi float64) Interval { return Interval{lo, hi} }

// TestSpecialValues is the hand-written table from IEEE 1788-2015, clauses 9 and 10.
func TestSpecialValues(t *testing.T) {
	E, N, R := Empty(), nai(), Entire()
	cases := []struct {
		name      string
		got, want Interval
	}{
		{"0*Entire", iv(0, 0).Mul(R), iv(0, 0)},
		{"Entire*0", R.Mul(iv(0, 0)), iv(0, 0)},
		{"[0,1]*[1,Inf]", iv(0, 1).Mul(iv(1, inf)), iv(0, inf)},
		{"[-1,0]*[-Inf,-1]", iv(-1, 0).Mul(iv(ninf, -1)), iv(0, inf)},
		{"[0,1]*[-Inf,-1]", iv(0, 1).Mul(iv(ninf, -1)), iv(ninf, 0)},
		{"[-1,1]*[-1,1]", iv(-1, 1).Mul(iv(-1, 1)), iv(-1, 1)},
		{"[-2,3]*[-5,7]", iv(-2, 3).Mul(iv(-5, 7)), iv(-15, 21)},
		{"[2,3]*[-5,-4]", iv(2, 3).Mul(iv(-5, -4)), iv(-15, -8)},
		{"[-3,-2]*[-5,-4]", iv(-3, -2).Mul(iv(-5, -4)), iv(8, 15)},
		{"[1,2]/[0,0]", iv(1, 2).Div(iv(0, 0)), E},
		{"[-1,1]/[0,0]", iv(-1, 1).Div(iv(0, 0)), E},
		{"[0,0]/[0,0]", iv(0, 0).Div(iv(0, 0)), E},
		{"[0,0]/[1,2]", iv(0, 0).Div(iv(1, 2)), iv(0, 0)},
		{"[1,2]/[0,1]", iv(1, 2).Div(iv(0, 1)), iv(1, inf)},
		{"[1,2]/[-1,0]", iv(1, 2).Div(iv(-1, 0)), iv(ninf, -1)},
		{"[-2,-1]/[0,1]", iv(-2, -1).Div(iv(0, 1)), iv(ninf, -1)},
		{"[-2,-1]/[-1,0]", iv(-2, -1).Div(iv(-1, 0)), iv(1, inf)},
		{"[0,2]/[0,1]", iv(0, 2).Div(iv(0, 1)), iv(0, inf)},
		{"[0,2]/[-1,0]", iv(0, 2).Div(iv(-1, 0)), iv(ninf, 0)},
		{"[-2,0]/[0,1]", iv(-2, 0).Div(iv(0, 1)), iv(ninf, 0)},
		{"[-2,0]/[-1,0]", iv(-2, 0).Div(iv(-1, 0)), iv(0, inf)},
		{"[1,2]/[-1,1]", iv(1, 2).Div(iv(-1, 1)), R},
		{"[-1,1]/[0,1]", iv(-1, 1).Div(iv(0, 1)), R},
		{"[1,2]/[4,8]", iv(1, 2).Div(iv(4, 8)), iv(0.125, 0.5)},
		{"[1,Inf]/[1,Inf]", iv(1, inf).Div(iv(1, inf)), iv(0, inf)},
		{"[-Inf,-1]/[-Inf,-1]", iv(ninf, -1).Div(iv(ninf, -1)), iv(0, inf)},
		{"[1,2]/[-Inf,-1]", iv(1, 2).Div(iv(ninf, -1)), iv(-2, 0)},
		{"Sqrt[-1,4]", iv(-1, 4).Sqrt(), iv(0, 2)},
		{"Sqrt[-2,-1]", iv(-2, -1).Sqrt(), E},
		{"Sqrt[0,Inf]", iv(0, inf).Sqrt(), iv(0, inf)},
		{"Sqr[-1,1]", iv(-1, 1).Sqr(), iv(0, 1)},
		{"Sqr[-3,2]", iv(-3, 2).Sqr(), iv(0, 9)},
		{"Sqr[-3,-2]", iv(-3, -2).Sqr(), iv(4, 9)},
		{"Pown[0,0]^0", iv(0, 0).Pown(0), iv(1, 1)},
		{"Pown Entire^0", R.Pown(0), iv(1, 1)},
		{"Pown[-2,3]^3", iv(-2, 3).Pown(3), iv(-8, 27)},
		{"Pown[-2,3]^2", iv(-2, 3).Pown(2), iv(0, 9)},
		{"Pown[-3,-2]^3", iv(-3, -2).Pown(3), iv(-27, -8)},
		{"Pown[0,0]^-1", iv(0, 0).Pown(-1), E},
		{"Pown[-1,1]^-2", iv(-1, 1).Pown(-2), iv(1, inf)},
		{"Pown[-1,1]^-1", iv(-1, 1).Pown(-1), R},
		{"Pown[2,4]^-1", iv(2, 4).Pown(-1), iv(0.25, 0.5)},
		{"Pown[2,4]^-2", iv(2, 4).Pown(-2), iv(0.0625, 0.25)},
		{"Pown[0.5,0.5]^MaxInt", iv(0.5, 0.5).Pown(math.MaxInt), iv(0, math.SmallestNonzeroFloat64)},
		{"Pown[-Inf,-1]^3", iv(ninf, -1).Pown(3), iv(ninf, -1)},
		{"Empty+x", E.Add(iv(1, 2)), E},
		{"Empty*0", E.Mul(iv(0, 0)), E},
		{"Pown Empty^0", E.Pown(0), E},
		{"Sqr Empty", E.Sqr(), E},
		{"Neg Empty", E.Neg(), E},
		{"-Inf + -Inf", iv(ninf, 1).Add(iv(ninf, 1)), iv(ninf, 2)},
		{"[-Inf,1]-[0,Inf]", iv(ninf, 1).Sub(iv(0, inf)), iv(ninf, 1)},
		{"Max+Max", iv(math.MaxFloat64, math.MaxFloat64).Add(iv(math.MaxFloat64, math.MaxFloat64)), iv(math.MaxFloat64, inf)},
		{"-Max-Max", iv(-math.MaxFloat64, -math.MaxFloat64).Sub(iv(math.MaxFloat64, math.MaxFloat64)), iv(ninf, -math.MaxFloat64)},
		{"Max*2", iv(math.MaxFloat64, math.MaxFloat64).Mul(iv(2, 2)), iv(math.MaxFloat64, inf)},
		{"Max/0.5", iv(math.MaxFloat64, math.MaxFloat64).Div(iv(0.5, 0.5)), iv(math.MaxFloat64, inf)},
		{"tiny*tiny", iv(0x1p-1074, 0x1p-1074).Mul(iv(0x1p-1074, 0x1p-1074)), iv(0, 0x1p-1074)},
		{"tiny/2", iv(0x1p-1074, 0x1p-1074).Div(iv(2, 2)), iv(0, 0x1p-1074)},
		{"Intersect disjoint", iv(1, 2).Intersect(iv(3, 4)), E},
		{"Intersect touch", iv(1, 2).Intersect(iv(2, 4)), iv(2, 2)},
		{"Hull", iv(1, 2).Hull(iv(5, 6)), iv(1, 6)},
		{"Hull Empty", E.Hull(iv(5, 6)), iv(5, 6)},
		{"1+1 exact", iv(1, 1).Add(iv(1, 1)), iv(2, 2)},
		{"0.5*0.5 exact", iv(0.5, 0.5).Mul(iv(0.5, 0.5)), iv(0.25, 0.25)},
		{"Sqrt 4 exact", iv(4, 4).Sqrt(), iv(2, 2)},
	}
	for _, c := range cases {
		if !c.got.Equal(c.want) {
			t.Errorf("%s = %v (%#v), want %v", c.name, c.got, c.got, c.want)
		}
	}
	// Extreme exponents: 2^MinInt underflows toward zero, 2^MaxInt overflows.
	if got := iv(2, 2).Pown(math.MinInt); got.Lo != 0 || !(got.Hi > 0 && got.Hi < 1e-300) {
		t.Errorf("Pown([2, 2], MinInt) = %#v, want [0, tiny]", got)
	}
	if got := iv(2, 2).Pown(math.MaxInt); !got.Equal(iv(math.MaxFloat64, inf)) {
		t.Errorf("Pown([2, 2], MaxInt) = %#v, want [MaxFloat64, +Inf]", got)
	}
	// NaI absorbs, and wins over Empty.
	for _, got := range []Interval{N.Add(E), E.Mul(N), N.Div(iv(1, 2)), N.Sqrt(), N.Sqr(), N.Pown(0), N.Neg(), N.Hull(E), N.Intersect(R)} {
		if !got.IsNaI() {
			t.Errorf("NaI did not absorb: %#v", got)
		}
	}
	// One third is not exact: the bounds must be adjacent float64 values around it.
	third := iv(1, 1).Div(iv(3, 3))
	if math.Nextafter(third.Lo, inf) != third.Hi || !encloses(third, big.NewRat(1, 3)) {
		t.Errorf("1/3 = %#v, want two adjacent float64 values around 1/3", third)
	}
}

func TestIllFormedInput(t *testing.T) {
	for _, x := range []Interval{{2, 1}, {math.NaN(), 1}, {1, math.NaN()}, {inf, inf}, {ninf, ninf}, {inf, 1}, {1, ninf}} {
		if !x.IsNaI() || !x.Add(iv(1, 1)).IsNaI() || x.Contains(1) || x.Equal(x) || x.Subset(Entire()) {
			t.Errorf("ill-formed %#v not handled as NaI", x)
		}
	}
}

func TestConstructors(t *testing.T) {
	if x, err := New(1, 2); err != nil || !x.Equal(iv(1, 2)) {
		t.Errorf("New(1, 2) = %v, %v", x, err)
	}
	for _, b := range [][2]float64{{2, 1}, {math.NaN(), 1}, {inf, inf}, {ninf, ninf}, {inf, ninf}} {
		if x, err := New(b[0], b[1]); !errors.Is(err, ErrInvalid) || !x.IsNaI() {
			t.Errorf("New(%v, %v) = %v, %v, want NaI and ErrInvalid", b[0], b[1], x, err)
		}
	}
	if x, err := New(ninf, inf); err != nil || !x.IsEntire() {
		t.Errorf("New(-Inf, +Inf) = %v, %v", x, err)
	}
	if !Point(inf).IsNaI() || !Point(math.NaN()).IsNaI() || !Point(3).Equal(iv(3, 3)) {
		t.Error("Point mishandles special values")
	}
	var zero Interval
	if zero.kind() != nonEmpty || !zero.Equal(iv(0, 0)) {
		t.Error("the zero value is not [0, 0]")
	}
	if got := iv(0, 0).Add(iv(0, 0)); !math.Signbit(got.Lo) || math.Signbit(got.Hi) {
		t.Errorf("zero endpoints not normalized to [-0, +0]: %#v", got)
	}
}

func TestPredicates(t *testing.T) {
	E := Empty()
	checks := []struct {
		name string
		got  bool
	}{
		{"Contains", iv(1, 2).Contains(1.5)},
		{"Contains Lo", iv(1, 2).Contains(1)},
		{"!Contains", !iv(1, 2).Contains(3)},
		{"!Contains Inf", !Entire().Contains(inf)},
		{"!Contains NaN", !Entire().Contains(math.NaN())},
		{"!Empty Contains", !E.Contains(0)},
		{"Subset", iv(1, 2).Subset(iv(0, 3))},
		{"Empty Subset", E.Subset(iv(0, 3))},
		{"!Subset Empty", !iv(1, 2).Subset(E)},
		{"Disjoint", iv(1, 2).Disjoint(iv(3, 4))},
		{"!Disjoint touch", !iv(1, 2).Disjoint(iv(2, 4))},
		{"Disjoint Empty", E.Disjoint(iv(1, 2))},
		{"Equal -0 +0", iv(math.Copysign(0, -1), 1).Equal(iv(0, 1))},
		{"Equal Empty", E.Equal(Empty())},
		{"IsEntire", Entire().IsEntire()},
		{"IsEmpty", E.IsEmpty() && !iv(1, 1).IsEmpty()},
	}
	for _, c := range checks {
		if !c.got {
			t.Errorf("%s failed", c.name)
		}
	}
}

func TestQueries(t *testing.T) {
	same := func(a, b float64) bool { return a == b || math.IsNaN(a) && math.IsNaN(b) }
	cases := []struct {
		x                         Interval
		width, mid, rad, mag, mig float64
	}{
		{iv(1, 3), 2, 2, 1, 3, 1},
		{iv(-3, 2), 5, -0.5, 2.5, 3, 0},
		{iv(-2, -1), 1, -1.5, 0.5, 2, 1},
		{Entire(), inf, 0, inf, inf, 0},
		{iv(ninf, 2), inf, -math.MaxFloat64, inf, inf, 0},
		{iv(1, inf), inf, math.MaxFloat64, inf, inf, 1},
		{iv(-math.MaxFloat64, math.MaxFloat64), inf, 0, math.MaxFloat64, math.MaxFloat64, 0},
		{Empty(), math.NaN(), math.NaN(), math.NaN(), math.NaN(), math.NaN()},
		{nai(), math.NaN(), math.NaN(), math.NaN(), math.NaN(), math.NaN()},
	}
	for _, c := range cases {
		if !same(c.x.Width(), c.width) || !same(c.x.Mid(), c.mid) || !same(c.x.Rad(), c.rad) ||
			!same(c.x.Mag(), c.mag) || !same(c.x.Mig(), c.mig) {
			t.Errorf("%v: width %v mid %v rad %v mag %v mig %v, want %v %v %v %v %v", c.x,
				c.x.Width(), c.x.Mid(), c.x.Rad(), c.x.Mag(), c.x.Mig(), c.width, c.mid, c.rad, c.mag, c.mig)
		}
	}
	// Width rounds up: 1 - (-2^-60) is not representable.
	if w := iv(-0x1p-60, 1).Width(); w <= 1 {
		t.Errorf("Width not rounded up: %v", w)
	}
}

func TestParse(t *testing.T) {
	tenth, err := Parse("0.1")
	if err != nil || math.Nextafter(tenth.Lo, inf) != tenth.Hi || !encloses(tenth, big.NewRat(1, 10)) {
		t.Errorf("Parse(0.1) = %#v, %v", tenth, err)
	}
	ok := []struct {
		in   string
		want Interval
	}{
		{"[1, 2]", iv(1, 2)},
		{" [ -1.5 , 2e3 ] ", iv(-1.5, 2000)},
		{"[3]", iv(3, 3)},
		{"0.25", iv(0.25, 0.25)},
		{"-0", iv(0, 0)},
		{"[empty]", Empty()},
		{"[Entire]", Entire()},
		{"[-inf, 2]", iv(ninf, 2)},
		{"[1, +Infinity]", iv(1, inf)},
		{"1e500", iv(math.MaxFloat64, inf)},
		{"-1e500", iv(ninf, -math.MaxFloat64)},
		{"1e-500", iv(0, math.SmallestNonzeroFloat64)},
		{"-1e-500", iv(-math.SmallestNonzeroFloat64, 0)},
		{"4.9406564584124654e-324", iv(0, math.SmallestNonzeroFloat64)},
		{"0x1", Interval{}}, // placeholder, rejected below
	}
	for _, c := range ok[:len(ok)-1] {
		got, err := Parse(c.in)
		if err != nil || !got.Equal(c.want) {
			t.Errorf("Parse(%q) = %v, %v, want %v", c.in, got, err, c.want)
		}
	}
	for _, in := range []string{"", "[]", "[2, 1]", "nan", "inf", "[inf]", "[-inf]", "1.2.3", "1e", "[1, 2", "abc", "[1,2,3]", "0x1", string(make([]byte, 600))} {
		if got, err := Parse(in); !errors.Is(err, ErrInvalid) || !got.IsNaI() {
			t.Errorf("Parse(%q) = %v, %v, want NaI and ErrInvalid", in, got, err)
		}
	}
}

func TestStringRoundTrip(t *testing.T) {
	for _, x := range []Interval{iv(1, 2), iv(0.1, 0.3), iv(ninf, 2), iv(-1, inf), Entire(), Empty(), iv(math.SmallestNonzeroFloat64, math.MaxFloat64), iv(1, 1).Div(iv(3, 3))} {
		back, err := Parse(x.String())
		if err != nil || !x.Subset(back) {
			t.Errorf("Parse(%q) = %v, %v, does not contain %v", x.String(), back, err, x)
		}
	}
	if s := nai().String(); s != "[nai]" {
		t.Errorf("NaI prints as %q", s)
	}
}
