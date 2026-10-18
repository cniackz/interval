// Copyright 2026 Cesar Celis
// SPDX-License-Identifier: Apache-2.0

package interval

import (
	"fmt"
	"math"
	"math/rand/v2"
	"os"
	"slices"
	"testing"
)

// TestMeasure produces the numbers of the sealed prediction. It is slow and runs only with
// INTERVAL_MEASURE=1:
//
//	INTERVAL_MEASURE=1 go test -run TestMeasure -v -timeout 1h
func TestMeasure(t *testing.T) {
	if os.Getenv("INTERVAL_MEASURE") == "" {
		t.Skip("set INTERVAL_MEASURE=1 to run")
	}
	r := rand.New(rand.NewPCG(2026, 1018))

	// Prediction 1: tightness on normal-range point intervals.
	normal := func(f float64) bool {
		a := math.Abs(f)
		return a >= 0x1p-1022 && a <= math.MaxFloat64
	}
	randNormal := func() float64 {
		for {
			if f := math.Float64frombits(r.Uint64()); normal(f) {
				return f
			}
		}
	}
	const n1 = 1000000
	for _, op := range binops {
		tight, total := 0, 0
		for total < n1 {
			a, b := randNormal(), randNormal()
			ex := op.exact(ratOf(a), ratOf(b))
			if ex == nil {
				continue
			}
			if f, _ := ex.Float64(); !normal(f) {
				continue
			}
			total++
			got := op.iv(Point(a), Point(b))
			if isRD(got.Lo, ex) && isRU(got.Hi, ex) {
				tight++
			}
		}
		t.Logf("P1 %s: %d of %d tight (%.4f%%)", op.name, tight, total, 100*float64(tight)/float64(total))
	}
	tight := 0
	for i := 0; i < n1; i++ {
		a := math.Abs(randNormal())
		if sqrtTight(Point(a).Sqrt(), a) {
			tight++
		}
	}
	t.Logf("P1 Sqrt: %d of %d tight (%.4f%%)", tight, n1, 100*float64(tight)/float64(n1))

	// Prediction 2: exact stays exact on integers in [-2^26, 2^26].
	for _, op := range binops[:3] {
		exact := 0
		for i := 0; i < n1; i++ {
			a := float64(r.Int64N(1<<27+1) - 1<<26)
			b := float64(r.Int64N(1<<27+1) - 1<<26)
			if got := op.iv(Point(a), Point(b)); got.Lo == got.Hi {
				exact++
			}
		}
		t.Logf("P2 %s: %d of %d exact (%.4f%%)", op.name, exact, n1, 100*float64(exact)/float64(n1))
	}

	// Prediction 5: width of the error-free path against one Nextafter step on every operation.
	const chains, steps = 10000, 100
	never, ratios := 0, make([]float64, 0, chains)
	for c := 0; c < chains; c++ {
		dec := func() Interval {
			x, err := Parse(fmt.Sprintf("%d.%02de%d", 1+r.IntN(9), r.IntN(100), r.IntN(3)-1))
			if err != nil {
				t.Fatal(err)
			}
			return x
		}
		a := dec()
		b := a
		for s := 0; s < steps; s++ {
			y := dec()
			switch r.IntN(3) {
			case 0:
				a, b = a.Add(y), nextAdd(b, y)
			case 1:
				a, b = a.Sub(y), nextSub(b, y)
			default:
				a, b = a.Mul(y), nextMul(b, y)
			}
		}
		wa, wb := a.Width(), b.Width()
		if wa <= wb {
			never++
		}
		if wb > 0 && !math.IsInf(wb, 0) {
			ratios = append(ratios, wa/wb)
		}
	}
	slices.Sort(ratios)
	t.Logf("P5 error-free width <= Nextafter width in %d of %d chains (%.2f%%); median ratio %.4f over %d chains",
		never, chains, 100*float64(never)/chains, ratios[len(ratios)/2], len(ratios))
}

// The Nextafter-only reference: round to nearest, then widen each endpoint by one step, always.
func down1(f float64) float64 { return math.Nextafter(f, math.Inf(-1)) }
func up1(f float64) float64   { return math.Nextafter(f, math.Inf(1)) }

func nextAdd(x, y Interval) Interval { return Interval{down1(x.Lo + y.Lo), up1(x.Hi + y.Hi)} }
func nextSub(x, y Interval) Interval { return Interval{down1(x.Lo - y.Hi), up1(x.Hi - y.Lo)} }
func nextMul(x, y Interval) Interval {
	p := []float64{x.Lo * y.Lo, x.Lo * y.Hi, x.Hi * y.Lo, x.Hi * y.Hi}
	return Interval{down1(slices.Min(p)), up1(slices.Max(p))}
}
