// Copyright 2026 Cesar Celis
// SPDX-License-Identifier: Apache-2.0

// Command mutants runs mutation testing on the case tables and the rounding code of the
// interval package: it changes one token at a time in arith.go and round.go, runs the tests,
// and reports the mutants that the tests did not kill. It uses the standard library only and is
// not part of the package API.
//
// Usage, from the module root:
//
//	go run ./internal/mutants [-min 95] [-files arith.go,round.go]
//
// A mutant that does not compile is discarded. The command exits with status 1 when the share of
// killed mutants is below -min percent.
package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"go/scanner"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"
)

// swaps lists the replacements tried for each token text. Operators flip a comparison or an
// arithmetic operation; identifiers swap a rounding direction, an endpoint or an operand.
var swaps = map[string][]string{
	"<": {"<="}, "<=": {"<"}, ">": {">="}, ">=": {">"}, "==": {"!="}, "!=": {"=="},
	"+": {"-"}, "-": {"+"},
	"rd": {"ru"}, "ru": {"rd"},
	"addDown": {"addUp"}, "addUp": {"addDown"},
	"subDown": {"subUp"}, "subUp": {"subDown"},
	"mulDown": {"mulUp"}, "mulUp": {"mulDown"},
	"divDown": {"divUp"}, "divUp": {"divDown"},
	"sqrtDown": {"sqrtUp"}, "sqrtUp": {"sqrtDown"},
	"powDown": {"powUp"}, "powUp": {"powDown"},
	"Lo": {"Hi"}, "Hi": {"Lo"},
	"a": {"b"}, "b": {"a"}, "c": {"d"}, "d": {"c"},
	"min": {"max"}, "max": {"min"},
	"clP": {"clN"}, "clN": {"clP"},
}

type mutant struct {
	file, desc string
	key        string // file | trimmed source line | token -> replacement
	src        []byte
}

// loadEquivalent reads equivalent.txt next to this command.
func loadEquivalent(path string) map[string]bool {
	eq := map[string]bool{}
	data, err := os.ReadFile(path)
	if err != nil {
		return eq
	}
	for _, line := range strings.Split(string(data), "\n") {
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		f := strings.Split(line, " | ")
		if len(f) >= 3 {
			eq[f[0]+" | "+f[1]+" | "+f[2]] = true
		}
	}
	return eq
}

func main() {
	minPct := flag.Float64("min", 0, "fail when fewer than this percent of mutants are killed")
	files := flag.String("files", "arith.go,round.go", "comma-separated files to mutate")
	eqPath := flag.String("equivalent", "internal/mutants/equivalent.txt", "file of mutants declared equivalent")
	flag.Parse()
	eq := loadEquivalent(*eqPath)

	var ms []mutant
	for _, f := range strings.Split(*files, ",") {
		src, err := os.ReadFile(f)
		if err != nil {
			fatal(err)
		}
		ms = append(ms, generate(f, src)...)
	}
	fmt.Printf("%d candidate mutants\n", len(ms))

	type result struct {
		m       mutant
		outcome string // killed, survived, invalid
	}
	results := make([]result, len(ms))
	var wg sync.WaitGroup
	sem := make(chan struct{}, max(1, runtime.NumCPU()/2))
	for i, m := range ms {
		wg.Add(1)
		sem <- struct{}{}
		go func() {
			defer wg.Done()
			defer func() { <-sem }()
			results[i] = result{m, run(m)}
		}()
	}
	wg.Wait()

	killed, survived, equivalent, killedEq := 0, 0, 0, 0
	var surv []string
	for _, r := range results {
		isEq := eq[r.m.key]
		switch {
		case r.outcome == "killed" && isEq:
			killedEq++ // declared equivalent but killed: the declaration is wrong
			fmt.Println("KILLED BUT DECLARED EQUIVALENT", r.m.file+":"+r.m.desc)
		case r.outcome == "killed":
			killed++
		case r.outcome == "survived" && isEq:
			equivalent++
		case r.outcome == "survived":
			survived++
			surv = append(surv, r.m.file+":"+r.m.desc)
		}
	}
	sort.Strings(surv)
	for _, s := range surv {
		fmt.Println("SURVIVED", s)
	}
	compiling := killed + survived + equivalent + killedEq
	raw := 100 * float64(killed+killedEq) / float64(max(compiling, 1))
	pct := 100 * float64(killed+killedEq) / float64(max(compiling-equivalent, 1))
	fmt.Printf("raw: killed %d of %d compiling mutants (%.1f%%); %d did not compile\n", killed+killedEq, compiling, raw, len(ms)-compiling)
	fmt.Printf("adjusted: %d declared equivalent and surviving, killed %d of %d (%.1f%%)\n", equivalent, killed+killedEq, compiling-equivalent, pct)
	if pct < *minPct || killedEq > 0 {
		os.Exit(1)
	}
}

// generate returns one mutant per mutable token occurrence. Comments are not tokens, and tokens on
// lines that declare a function signature are skipped, so parameter lists stay intact.
func generate(name string, src []byte) []mutant {
	fset := token.NewFileSet()
	file := fset.AddFile(name, fset.Base(), len(src))
	var s scanner.Scanner
	s.Init(file, src, nil, 0)
	lines := bytes.Split(src, []byte("\n"))
	var out []mutant
	for {
		pos, tok, lit := s.Scan()
		if tok == token.EOF {
			break
		}
		text := lit
		if text == "" {
			text = tok.String()
		}
		repl, ok := swaps[text]
		if !ok || (tok != token.IDENT && !tok.IsOperator()) {
			continue
		}
		p := fset.Position(pos)
		line := string(lines[p.Line-1])
		if strings.HasPrefix(strings.TrimSpace(line), "func ") || strings.HasPrefix(strings.TrimSpace(line), "//") {
			continue
		}
		off := file.Offset(pos)
		for _, r := range repl {
			b := make([]byte, 0, len(src)+4)
			b = append(b, src[:off]...)
			b = append(b, r...)
			b = append(b, src[off+len(text):]...)
			key := name + " | " + strings.TrimSpace(line) + " | " + text + " -> " + r
			out = append(out, mutant{name, fmt.Sprintf("%d:%d %s -> %s", p.Line, p.Column, text, r), key, b})
		}
	}
	return out
}

// run copies the module to a temporary directory with the mutated file and runs the short tests.
func run(m mutant) string {
	dir, err := os.MkdirTemp("", "mutant")
	if err != nil {
		fatal(err)
	}
	defer os.RemoveAll(dir)
	entries, err := os.ReadDir(".")
	if err != nil {
		fatal(err)
	}
	for _, e := range entries {
		n := e.Name()
		if e.IsDir() || !(strings.HasSuffix(n, ".go") || n == "go.mod") {
			continue
		}
		src, err := os.ReadFile(n)
		if err != nil {
			fatal(err)
		}
		if n == m.file {
			src = m.src
		}
		if err := os.WriteFile(filepath.Join(dir, n), src, 0o644); err != nil {
			fatal(err)
		}
	}
	if exec.Command("go", "-C", dir, "build", ".").Run() != nil {
		return "invalid"
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, "go", "-C", dir, "test", "-short", "-count=1", "-run", "Test|Example", ".")
	if cmd.Run() != nil {
		return "killed"
	}
	return "survived"
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(2)
}
