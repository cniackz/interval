// Copyright 2026 Cesar Celis
// SPDX-License-Identifier: Apache-2.0

package interval

import (
	"math"
	"math/big"
	"strconv"
	"strings"
)

// maxParseLen bounds the input of Parse, so that a hostile string cannot make it spend unbounded
// time or memory in math/big.
const maxParseLen = 512

// Parse reads an interval from text, a subset of the syntax of IEEE 1788-2015, clause 12.11:
//
//	"0.1"         a decimal number, enclosed in the tightest interval that contains it
//	"[a, b]"      two decimal numbers, a <= b; either may be -inf or +inf (as "inf", "Inf", ...)
//	"[a]"         the same as "a"
//	"[empty]"     the empty interval
//	"[entire]"    the whole real line
//
// Spaces around the numbers and brackets are allowed. A decimal such as 0.1 has no exact float64
// value, so the lower bound is rounded down and the upper bound rounded up: Parse("0.1") is
// [RD(0.1), RU(0.1)], two adjacent float64 values, and it contains the real 0.1. Decimal
// exponents ("1e-3") are accepted. Inputs longer than 512 bytes, NaN, a single infinite number
// and a > b give NaI and ErrInvalid.
func Parse(s string) (Interval, error) {
	if len(s) > maxParseLen {
		return nai(), ErrInvalid
	}
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "[") && strings.HasSuffix(s, "]") {
		body := strings.TrimSpace(s[1 : len(s)-1])
		switch strings.ToLower(body) {
		case "empty":
			return Empty(), nil
		case "entire":
			return Entire(), nil
		}
		if lo, hi, ok := strings.Cut(body, ","); ok {
			a, ok1 := parseBound(strings.TrimSpace(lo), false)
			b, ok2 := parseBound(strings.TrimSpace(hi), true)
			if !ok1 || !ok2 {
				return nai(), ErrInvalid
			}
			return New(a, b)
		}
		s = body
	}
	a, ok1 := parseBound(s, false)
	b, ok2 := parseBound(s, true)
	if !ok1 || !ok2 || math.IsInf(a, 0) && a == b { // a lone infinity is not a real number
		return nai(), ErrInvalid
	}
	return New(a, b)
}

// parseBound converts one decimal number to float64, rounded down (up == false) or up.
func parseBound(s string, up bool) (float64, bool) {
	neg := false
	t := s
	if t != "" && (t[0] == '+' || t[0] == '-') {
		neg = t[0] == '-'
		t = t[1:]
	}
	if l := strings.ToLower(t); l == "inf" || l == "infinity" {
		if neg {
			return math.Inf(-1), true
		}
		return math.Inf(1), true
	}
	digits, exp, ok := scanDecimal(t)
	if !ok {
		return 0, false
	}
	// Signed magnitude of the decimal, used to clamp out-of-range values before math/big sees
	// them: below 1e-400 every value rounds into [0, SmallestNonzeroFloat64], above 1e400 into
	// [MaxFloat64, +Inf].
	mag := exp + len(digits) // the value lies in [10^(mag-1), 10^mag)
	switch {
	case digits == "":
		return 0, true
	case mag < -400:
		switch {
		case up && neg:
			return 0, true
		case up:
			return math.SmallestNonzeroFloat64, true
		case neg:
			return -math.SmallestNonzeroFloat64, true
		}
		return 0, true
	case mag > 400:
		if neg {
			if up {
				return -math.MaxFloat64, true
			}
			return math.Inf(-1), true
		}
		if up {
			return math.Inf(1), true
		}
		return math.MaxFloat64, true
	}
	var r big.Rat
	if _, ok := r.SetString(digits + "e" + strconv.Itoa(exp)); !ok {
		return 0, false
	}
	if neg {
		r.Neg(&r)
	}
	f, _ := r.Float64() // nearest float64, subnormals included; may be +-Inf on overflow
	var d int
	if math.IsInf(f, 0) {
		d = -sign(f)
	} else {
		var g big.Rat
		g.SetFloat64(f)
		d = r.Cmp(&g) // sign of (exact - f)
	}
	if up {
		return ru(f, d), true
	}
	return rd(f, d), true
}

// scanDecimal splits digits[.digits][(e|E)[+-]digits] into the significant digits, with leading
// zeros removed, and the power of ten that scales them to the value. An all-zero number returns
// empty digits.
func scanDecimal(s string) (digits string, exp int, ok bool) {
	var b strings.Builder
	i, seen, frac := 0, false, 0
	for ; i < len(s) && s[i] >= '0' && s[i] <= '9'; i++ {
		b.WriteByte(s[i])
		seen = true
	}
	if i < len(s) && s[i] == '.' {
		i++
		for ; i < len(s) && s[i] >= '0' && s[i] <= '9'; i++ {
			b.WriteByte(s[i])
			frac++
			seen = true
		}
	}
	if !seen {
		return "", 0, false
	}
	e := 0
	if i < len(s) && (s[i] == 'e' || s[i] == 'E') {
		i++
		eneg := false
		if i < len(s) && (s[i] == '+' || s[i] == '-') {
			eneg = s[i] == '-'
			i++
		}
		start := i
		for ; i < len(s) && s[i] >= '0' && s[i] <= '9'; i++ {
			if e < 100000 { // saturate: anything this large is clamped by the caller
				e = e*10 + int(s[i]-'0')
			}
		}
		if i == start {
			return "", 0, false
		}
		if eneg {
			e = -e
		}
	}
	if i != len(s) {
		return "", 0, false
	}
	digits = strings.TrimLeft(b.String(), "0")
	exp = e - frac
	if digits == "" {
		return "", 0, true
	}
	// Trailing zeros only shift the exponent.
	trimmed := strings.TrimRight(digits, "0")
	exp += len(digits) - len(trimmed)
	return trimmed, exp, true
}

// String returns x as "[lo, hi]", "[empty]", "[entire]" or "[nai]". Each bound is printed as the
// shortest decimal that rounds to it, so Parse(x.String()) returns an interval that contains x.
func (x Interval) String() string {
	switch x.kind() {
	case ill:
		return "[nai]"
	case empty:
		return "[empty]"
	}
	if x.IsEntire() {
		return "[entire]"
	}
	return "[" + fmtBound(x.Lo) + ", " + fmtBound(x.Hi) + "]"
}

func fmtBound(f float64) string {
	switch {
	case math.IsInf(f, 1):
		return "+inf"
	case math.IsInf(f, -1):
		return "-inf"
	case f == 0:
		return "0"
	}
	return strconv.FormatFloat(f, 'g', -1, 64)
}
