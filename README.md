# interval

Rigorous interval arithmetic on `float64` for Go, with the standard library only.

An `Interval{Lo, Hi}` stands for every real number between its bounds. Every operation returns an
interval that is **guaranteed to contain the exact result**, rounding included. Use it to carry
bounds you can trust through a computation: inputs known within a tolerance, a parameter known
only to lie in a range, or a check that a float64 result is not off by more than you think.

```go
free, _ := interval.Parse("[120, 125]") // units of free volume
rate, _ := interval.Parse("[3.1, 3.4]") // units per day
days := free.Div(rate)                   // [35.294..., 40.322...], guaranteed
```

## The problem

Plain `float64` code rounds every operation to the nearest representable value. If you compute a
bound with ordinary `/`, the result can land one unit in the last place inside the true bound, so
the "guaranteed" range can exclude the true value, and the error grows along a chain. Products and
quotients of ranges that cross zero or are unbounded need careful case analysis: taking the min
and max of the four endpoint products gives NaN for `0 * Inf` and for `Inf / Inf`.

This package solves both. Each operation satisfies the **inclusion property**: if `a` is in `x`
and `b` is in `y`, the exact real `a op b` is in `x.Op(y)` (Moore 1966; IEEE 1788-2015). It holds
through overflow, underflow, signed zeros and unbounded endpoints.

## How the bounds are computed

Go cannot switch the processor's rounding mode. The package computes each endpoint by round to
nearest and finds the exact sign of the rounding error with an error-free transformation: TwoSum
for addition and subtraction, and a fused multiply-add for multiplication, division and square
root. It moves an endpoint only when the result was inexact. So:

- `Add`, `Sub`, `Mul`, `Div` and `Sqrt` return the **tightest** `float64` bounds (the directed
  roundings of the exact result).
- **Exact results stay exact**: `[1, 1] + [1, 1]` is `[2, 2]`, not a range one ulp wide on each side.
- Near the underflow and overflow thresholds, where an error-free transformation can lose
  exactness, the sign comes from an exact comparison in `math/big`. Only that rare path allocates.

## API

| Group | Functions and methods |
|---|---|
| Build | `New(lo, hi)`, `Point(x)`, `Parse(s)`, `Empty()`, `Entire()`, or the literal `Interval{Lo, Hi}` |
| Arithmetic | `Add`, `Sub`, `Mul`, `Div`, `DivSplit`, `Neg`, `Sqr`, `Pown(n)`, `Sqrt` |
| Sets | `Intersect`, `Hull` |
| Predicates | `IsEmpty`, `IsEntire`, `IsNaI`, `Contains(v)`, `Subset`, `Disjoint`, `Equal` |
| Numbers | `Width`, `Mid`, `Rad`, `Mag`, `Mig`, `String` |

Behavior worth knowing:

- **Empty** is a value: `Sqrt([-2, -1])` and division by `[0, 0]` return it.
- **Unbounded** intervals use infinite endpoints. Infinity is never a member.
- **Division by a range that contains zero**: zero as an endpoint gives a half-unbounded result
  (`[1, 2] / [0, 1] = [1, +Inf]`); zero inside gives the hull, often `Entire`, and `DivSplit`
  returns the two pieces.
- **No panics.** An ill-formed value, such as `Interval{2, 1}` or a NaN bound, gives **NaI** ("not
  an interval"), which propagates through every operation. `New` and `Parse` also return
  `ErrInvalid`.
- **Decimal input**: `Point(0.1)` holds the `float64` nearest to 0.1, which is not 0.1.
  `Parse("0.1")` returns two adjacent `float64` values around the real 0.1.
- Compare with `Equal`, not `==`.
- The zero value is `[0, 0]`. Arithmetic, set and query methods do not allocate.

## Benchmarks

Apple silicon (darwin/arm64), Go 1.27.2, `go test -bench . -benchmem -count 10`, median of 10, operands
bounded and positive unless noted.

| Operation | ns/op | allocs/op |
|---|---|---|
| `Add` | 7.6 | 0 |
| `Sub` | 7.4 | 0 |
| `Mul` | 9.4 | 0 |
| `Mul`, both operands containing zero | 15.4 | 0 |
| `Div` | 10.7 | 0 |
| `Sqrt` | 7.6 | 0 |
| `Sqr` | 9.1 | 0 |
| `Pown(5)` | 29.4 | 0 |
| `Intersect` | 2.8 | 0 |
| `Mid` | 1.8 | 0 |
| `Parse("[0.1, 0.2]")` | 546 | 40 |
| `String` | 97 | 3 |
| plain `float64` add of both endpoints, for reference | 1.9 | 0 |

Rigor costs about 4x a plain `float64` addition.

## Known limits

**The dependency problem.** Interval arithmetic treats every occurrence of a variable as
independent. For `x = [-1, 2]`, `x.Sub(x)` is `[-3, 3]`, not `[0, 0]`, and `x.Mul(x)` is
`[-2, 4]` while the true range of `x * x` is `[0, 4]`. The result is still a valid enclosure, only
wider than necessary. To keep bounds tight: write each variable once where you can, use `Sqr` and
`Pown` for powers, or split an input range into pieces and take the `Hull` of the results.

**The wrapping effect.** Iterating a computation step by step on intervals can make the width grow
fast. This package is a building block for short chains of operations, not a set-valued
simulator.

**Worst case, not likelihood.** The bound covers every value consistent with the input bounds. It
says nothing about which values are likely, and it is only as good as the inputs.

**Pown is an enclosure, not the tightest bound.** `Pown` multiplies repeatedly and rounds
outward at each step, so for `n > 0` a bound can be a few ulps wider than the tightest one (an
attack measured up to about 20 ulps). For `n < 0` it divides one by `x.Pown(-n)`, which loses
more when that power overflows: `[2, 2].Pown(math.MinInt)` gives `[0, 2^-1024]` instead of
`[0, 2^-1074]`. `Sqr` is tight; for higher powers, the bound is a valid enclosure, a few ulps wide.

**Small edges.** `String` prints NaI as `"[nai]"`, and `Parse("[nai]")` gives back NaI together
with `ErrInvalid`, so NaI round-trips as an error, not as a value. `DivSplit` pieces can share the
endpoint zero when the divisor is unbounded (`[1, 2] / [-Inf, 1]` gives `[-Inf, 0]` and
`[1, +Inf]`).

**No elementary functions.** `exp`, `log` and the trigonometric functions are not provided,
because the Go `math` package documents no error bound for them.

## v1 follow-ups

- **The `huge` guard in TwoSum may be redundant.** `addDir` sends sums with an operand of magnitude
  2^1000 or more to the exact `math/big` comparison, as protection against overflow inside TwoSum.
  S. Boldo, S. Graillat and J.-M. Muller, "On the robustness of the 2Sum and Fast2Sum algorithms,"
  ACM Transactions on Mathematical Software 44(1), 2017, analyze when the intermediate operations
  of 2Sum can overflow, and their analysis suggests the guard is not needed; an independent audit found 0 differences against `math/big`
  on millions of inputs without the guard. Candidate for removal in v1, only together with a test
  that pins the overflow-range behavior against the exact oracle.
- **Pown tightness.** Compute `x^n` with one final directed rounding (or an exact `math/big`
  power on the slow path) so that `Pown` matches the tightness of the basic operations, and handle
  overflow of `x^-n` for negative `n` directly.
- **Parse allocations.** `Parse("[0.1, 0.2]")` allocates 40 times; most of that can go.

## Testing

- **Inclusion** is checked against an exact `math/big.Rat` oracle on random intervals and members,
  including unbounded intervals and values at the underflow and overflow thresholds.
- **Tightness** is checked exactly: each bound must be the largest (or smallest) `float64` on its
  side of the exact result, also across the whole exponent range.
- **Algebra**: commutativity, `x - y = x + (-y)`, the set laws, subdistributivity, inclusion
  isotonicity, and a test that the dependency problem is still there.
- **Special values**: a table from IEEE 1788-2015 (`[0, 0] * Entire = [0, 0]`,
  `[1, 2] / [-1, 0] = [-Inf, -1]`, `Pown(x, 0) = [1, 1]`, Empty and NaI propagation).
- **Fuzzing** of every operation over raw bit patterns (every NaN payload, both zeros,
  subnormals, infinities) and of `Parse`.
- **Mutation testing** of the case tables and the rounding code: `go run ./internal/mutants`.
- CI runs `go vet`, `gofumpt`, `staticcheck`, `go test -race` and a short fuzz run on Linux amd64,
  Linux arm64 and macOS, with the two latest Go releases. The arm64 runs guard against the
  compiler fusing a multiply and an add, which would break the rounding analysis.

## References

- R. E. Moore, *Interval Analysis*, Prentice-Hall, 1966.
- R. E. Moore, R. B. Kearfott, M. J. Cloud, *Introduction to Interval Analysis*, SIAM, 2009.
- IEEE Std 1788-2015, *IEEE Standard for Interval Arithmetic*; IEEE Std 1788.1-2017 (simplified).
- IEEE Std 754-2019, *IEEE Standard for Floating-Point Arithmetic*.
- T. Hickey, Q. Ju, M. H. van Emden, "Interval Arithmetic: From Principles to Implementation,"
  *Journal of the ACM* 48(5), 2001.
- O. Moller, "Quasi Double-Precision in Floating Point Addition," *BIT* 5, 1965.
- D. E. Knuth, *The Art of Computer Programming, Vol. 2*, 3rd ed., 1997.
- T. J. Dekker, "A Floating-Point Technique for Extending the Available Precision,"
  *Numerische Mathematik* 18, 1971.
- G. Bohlender, W. Walter, P. Kornerup, D. W. Matula, "Semantics for Exact Floating Point
  Operations," ARITH 10, 1991.
- S. Boldo, J.-M. Muller, "Exact and Approximated Error of the FMA," *IEEE Transactions on
  Computers* 60(2), 2011.
- J.-M. Muller et al., *Handbook of Floating-Point Arithmetic*, 2nd ed., 2018.

## License

Apache-2.0. See [LICENSE](LICENSE).
