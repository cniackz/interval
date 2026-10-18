// Copyright 2026 Cesar Celis
// SPDX-License-Identifier: Apache-2.0

package interval

import (
	"math"
	"math/big"
	"math/rand/v2"
	"testing"
)

// TestTightness checks that, on point intervals, every basic operation returns exactly the
// directed roundings RD and RU of the exact result (IEEE 1788-2015, clause 12.9, tightest).
func TestTightness(t *testing.T) {
	r := rand.New(rand.NewPCG(1, 2))
	n := iterations(t, 200000)
	for _, op := range binops {
		for i := 0; i < n; i++ {
			a, b := randFinite(r), randFinite(r)
			ex := op.exact(ratOf(a), ratOf(b))
			got := op.iv(Point(a), Point(b))
			if ex == nil {
				if !got.IsEmpty() {
					t.Fatalf("%s(%v, %v) = %v, want Empty", op.name, a, b, got)
				}
				continue
			}
			if !isRD(got.Lo, ex) || !isRU(got.Hi, ex) {
				t.Fatalf("%s(%v, %v) = [%v, %v], want [RD, RU] of %v", op.name, a, b, got.Lo, got.Hi, ex.FloatString(40))
			}
		}
	}
	for i := 0; i < n; i++ {
		a := math.Abs(randFinite(r))
		if got := Point(a).Sqrt(); !sqrtTight(got, a) {
			t.Fatalf("Sqrt(%v) = %v, not tight", a, got)
		}
	}
}

// TestInclusion checks the fundamental theorem of interval arithmetic (Moore 1966, theorem 3.1):
// for members a of x and b of y, the exact a op b is a member of x op y.
func TestInclusion(t *testing.T) {
	r := rand.New(rand.NewPCG(3, 4))
	n := iterations(t, 100000)
	for _, op := range binops {
		for i := 0; i < n; i++ {
			x, y := randInterval(r), randInterval(r)
			a, b := randMember(r, x), randMember(r, y)
			ex := op.exact(ratOf(a), ratOf(b))
			if ex == nil {
				continue
			}
			got := op.iv(x, y)
			if !encloses(got, ex) {
				t.Fatalf("%s(%v, %v) = %v misses %v op %v", op.name, x, y, got, a, b)
			}
		}
	}
	for i := 0; i < n; i++ {
		x := randInterval(r)
		a := randMember(r, x)
		if got := x.Sqr(); !encloses(got, new(big.Rat).Mul(ratOf(a), ratOf(a))) {
			t.Fatalf("Sqr(%v) = %v misses %v^2", x, got, a)
		}
		if a >= 0 {
			if got := x.Sqrt(); !sqrtEncloses(got, a) {
				t.Fatalf("Sqrt(%v) = %v misses sqrt(%v)", x, got, a)
			}
		}
		k := r.IntN(13) - 6
		if a == 0 && k < 0 {
			continue
		}
		ex := new(big.Rat).SetInt64(1)
		for j := 0; j < abs(k); j++ {
			ex.Mul(ex, ratOf(a))
		}
		if k < 0 {
			ex.Inv(ex)
		}
		if got := x.Pown(k); !encloses(got, ex) {
			t.Fatalf("Pown(%v, %d) = %v misses %v^%d", x, k, got, a, k)
		}
	}
}

func abs(k int) int {
	if k < 0 {
		return -k
	}
	return k
}

// TestDivSplitInclusion checks that one of the two pieces holds every quotient, and that the
// pieces are inside the hull Div returns.
func TestDivSplitInclusion(t *testing.T) {
	r := rand.New(rand.NewPCG(5, 6))
	for i := 0; i < iterations(t, 100000); i++ {
		x, y := randInterval(r), randInterval(r)
		a, b := randMember(r, x), randMember(r, y)
		p, q := x.DivSplit(y)
		h := x.Div(y)
		if !p.Subset(h) || !q.Subset(h) {
			t.Fatalf("DivSplit(%v, %v) = %v, %v not inside Div = %v", x, y, p, q, h)
		}
		if b == 0 {
			continue
		}
		ex := new(big.Rat).Quo(ratOf(a), ratOf(b))
		if !encloses(p, ex) && !encloses(q, ex) {
			t.Fatalf("DivSplit(%v, %v) = %v, %v misses %v / %v", x, y, p, q, a, b)
		}
		if !q.IsEmpty() && !(p.Hi <= q.Lo) { // outward rounding may make the pieces touch
			t.Fatalf("DivSplit(%v, %v) = %v, %v: pieces out of order", x, y, p, q)
		}
	}
}

// TestIsotonicity checks inclusion isotonicity (Moore 1966, chapter 3): shrinking the operands
// never widens the result.
func TestIsotonicity(t *testing.T) {
	r := rand.New(rand.NewPCG(7, 8))
	for _, op := range binops {
		for i := 0; i < iterations(t, 50000); i++ {
			x, y := randInterval(r), randInterval(r)
			a, b := randMember(r, x), randMember(r, y)
			c, d := randMember(r, x), randMember(r, y)
			xs := Interval{min(a, c), max(a, c)}
			ys := Interval{min(b, d), max(b, d)}
			if got, want := op.iv(xs, ys), op.iv(x, y); !got.Subset(want) {
				t.Fatalf("%s: %v op %v = %v not inside %v op %v = %v", op.name, xs, ys, got, x, y, want)
			}
		}
	}
}

// TestAlgebra checks the identities that must hold exactly and the ones that hold only as
// inclusion (Moore 1966, chapter 2).
func TestAlgebra(t *testing.T) {
	r := rand.New(rand.NewPCG(9, 10))
	for i := 0; i < iterations(t, 50000); i++ {
		x, y, z := randInterval(r), randInterval(r), randInterval(r)
		if !x.Add(y).Equal(y.Add(x)) || !x.Mul(y).Equal(y.Mul(x)) {
			t.Fatalf("not commutative: %v, %v", x, y)
		}
		if !x.Neg().Neg().Equal(x) {
			t.Fatalf("Neg(Neg(%v)) = %v", x, x.Neg().Neg())
		}
		if !x.Sub(y).Equal(x.Add(y.Neg())) {
			t.Fatalf("%v - %v = %v, but %v + (-%v) = %v", x, y, x.Sub(y), x, y, x.Add(y.Neg()))
		}
		if !x.Hull(y).Equal(y.Hull(x)) || !x.Intersect(y).Equal(y.Intersect(x)) ||
			!x.Hull(y.Hull(z)).Equal(x.Hull(y).Hull(z)) ||
			!x.Intersect(y.Intersect(z)).Equal(x.Intersect(y).Intersect(z)) ||
			!x.Hull(x).Equal(x) || !x.Intersect(x).Equal(x) {
			t.Fatalf("set laws fail on %v, %v, %v", x, y, z)
		}
		if !x.Sub(x).Contains(0) {
			t.Fatalf("%v - itself = %v excludes 0", x, x.Sub(x))
		}
		if !x.Sqr().Subset(x.Mul(x)) {
			t.Fatalf("Sqr(%v) = %v not inside %v", x, x.Sqr(), x.Mul(x))
		}
		// Subdistributivity: x(y + z) is inside xy + xz, up to outward rounding, which can make
		// the left side one ulp wider at an endpoint. Check on small integers, where every
		// operation is exact, so the law holds exactly.
		xi, yi, zi := smallInt(r), smallInt(r), smallInt(r)
		if !xi.Mul(yi.Add(zi)).Subset(xi.Mul(yi).Add(xi.Mul(zi))) {
			t.Fatalf("subdistributivity fails on %v, %v, %v", xi, yi, zi)
		}
	}
	// The dependency problem, asserted so that nobody "fixes" it: equality does NOT hold.
	x := Interval{1, 2}
	if x.Sub(x).Equal(Interval{0, 0}) {
		t.Fatal("x - x became [0, 0]: the operations no longer treat operands as independent")
	}
	w, y, z := Interval{-1, 1}, Interval{1, 1}, Interval{-1, -1}
	if got, wide := w.Mul(y.Add(z)), w.Mul(y).Add(w.Mul(z)); got.Equal(wide) {
		t.Fatalf("subdistributivity is strict on this case, got equality %v", got)
	}
}

func smallInt(r *rand.Rand) Interval {
	a, b := float64(r.IntN(41)-20), float64(r.IntN(41)-20)
	return Interval{min(a, b), max(a, b)}
}

// TestWellFormed feeds every operation arbitrary bit patterns, ill-formed values included, and
// checks that each result is well-formed: non-empty, Empty or NaI.
func TestWellFormed(t *testing.T) {
	r := rand.New(rand.NewPCG(11, 12))
	pick := func() float64 {
		if r.IntN(3) == 0 {
			return []float64{math.NaN(), math.Inf(1), math.Inf(-1), 0, math.Copysign(0, -1)}[r.IntN(5)]
		}
		return math.Float64frombits(r.Uint64())
	}
	for i := 0; i < iterations(t, 200000); i++ {
		x, y := Interval{pick(), pick()}, Interval{pick(), pick()}
		p, q := x.DivSplit(y)
		for _, got := range []Interval{
			x.Add(y), x.Sub(y), x.Mul(y), x.Div(y), p, q, x.Neg(), x.Sqr(), x.Sqrt(),
			x.Pown(r.IntN(9) - 4), x.Intersect(y), x.Hull(y),
		} {
			if got.kind() == ill && !(math.IsNaN(got.Lo) && math.IsNaN(got.Hi)) {
				t.Fatalf("ill-formed result %#v from %#v, %#v", got, x, y)
			}
		}
		if (x.IsNaI() || y.IsNaI()) && !x.Add(y).IsNaI() {
			t.Fatalf("NaI did not propagate: %v + %v = %v", x, y, x.Add(y))
		}
	}
}

// TestTightnessExtremes checks tightness on operands spread over the whole exponent range, so that
// every guard between the error-free path and the exact math/big path is exercised from both
// sides: subnormals, values around 2^-969 and 2^-900, and values around 2^1000.
func TestTightnessExtremes(t *testing.T) {
	r := rand.New(rand.NewPCG(13, 14))
	exps := []int{-1074, -1060, -1030, -1022, -1000, -975, -970, -969, -950, -901, -900, -899, -600, -200, -60, -1, 0, 1, 60, 200, 600, 950, 999, 1000, 1001, 1020, 1023}
	pick := func() float64 {
		e := exps[r.IntN(len(exps))] + r.IntN(3) - 1
		f := math.Ldexp(1+r.Float64(), e)
		if r.IntN(2) == 0 {
			f = -f
		}
		if math.IsInf(f, 0) {
			f = math.Copysign(math.MaxFloat64, f)
		}
		return f
	}
	n := iterations(t, 100000)
	for _, op := range binops {
		for i := 0; i < n; i++ {
			a, b := pick(), pick()
			ex := op.exact(ratOf(a), ratOf(b))
			if ex == nil {
				continue
			}
			if got := op.iv(Point(a), Point(b)); !isRD(got.Lo, ex) || !isRU(got.Hi, ex) {
				t.Fatalf("%s(%v, %v) = [%v, %v], not tight", op.name, a, b, got.Lo, got.Hi)
			}
		}
	}
	for i := 0; i < n; i++ {
		a := math.Abs(pick())
		if got := Point(a).Sqrt(); !sqrtTight(got, a) {
			t.Fatalf("Sqrt(%v) = %v, not tight", a, got)
		}
	}
}

// TestHelpersInfinite pins the directed-rounding helpers on infinite operands, called directly.
// The public operations never ask for these bounds (a lower bound of +Inf or an upper bound of
// -Inf), so without this test the infinity checks in addDir, mulDir and divDir are not covered.
func TestHelpersInfinite(t *testing.T) {
	inf := math.Inf(1)
	cases := []struct {
		name      string
		got, want float64
	}{
		{"addDown(+Inf, 0)", addDown(inf, 0), inf},
		{"addDown(0, +Inf)", addDown(0, inf), inf},
		{"addUp(-Inf, 0)", addUp(-inf, 0), -inf},
		{"addUp(0, -Inf)", addUp(0, -inf), -inf},
		{"mulDown(+Inf, 2)", mulDown(inf, 2), inf},
		{"mulDown(2, +Inf)", mulDown(2, inf), inf},
		{"mulUp(-Inf, 2)", mulUp(-inf, 2), -inf},
		{"mulUp(2, -Inf)", mulUp(2, -inf), -inf},
		{"divDown(+Inf, 2)", divDown(inf, 2), inf},
		{"divUp(-Inf, 2)", divUp(-inf, 2), -inf},
		{"divDown(2, +Inf)", divDown(2, inf), 0},
		{"divUp(-2, +Inf)", divUp(-2, inf), 0},
	}
	for _, c := range cases {
		if c.got != c.want {
			t.Errorf("%s = %v, want %v", c.name, c.got, c.want)
		}
	}
}
