// Copyright (c) 2023, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package markus_test

import (
	"context"
	"fmt"
	"math/rand"
	"reflect"
	"runtime"
	"strconv"
	"sync"
	"testing"
	"time"

	"resenje.org/markus"
)

func TestVoting(t *testing.T) {

	type ballot[C comparable] struct {
		ballot markus.Ballot[C, uint64]
		unvote bool
	}
	for _, tc := range []struct {
		name    string
		choices []string
		ballots []ballot[string]
		result  []markus.Result[string]
		tie     bool
	}{
		{
			name:   "empty",
			result: []markus.Result[string]{},
		},
		{
			name:    "single option no votes",
			choices: []string{"A"},
			result: []markus.Result[string]{
				{Choice: "A", Index: 0, Wins: 0},
			},
		},
		{
			name:    "single option one vote",
			choices: []string{"A"},
			ballots: []ballot[string]{
				{ballot: markus.Ballot[string, uint64]{"A": 1}},
			},
			result: []markus.Result[string]{
				{Choice: "A", Index: 0, Wins: 0},
			},
		},
		{
			name:    "two options one vote",
			choices: []string{"A", "B"},
			ballots: []ballot[string]{
				{ballot: markus.Ballot[string, uint64]{"A": 1}},
			},
			result: []markus.Result[string]{
				{Choice: "A", Index: 0, Wins: 1},
				{Choice: "B", Index: 1, Wins: 0},
			},
		},
		{
			name:    "two options two votes",
			choices: []string{"A", "B"},
			ballots: []ballot[string]{
				{ballot: markus.Ballot[string, uint64]{"A": 1}},
				{ballot: markus.Ballot[string, uint64]{"A": 1, "B": 2}},
			},
			result: []markus.Result[string]{
				{Choice: "A", Index: 0, Wins: 1},
				{Choice: "B", Index: 1, Wins: 0},
			},
		},
		{
			name:    "three options three votes",
			choices: []string{"A", "B", "C"},
			ballots: []ballot[string]{
				{ballot: markus.Ballot[string, uint64]{"A": 1}},
				{ballot: markus.Ballot[string, uint64]{"A": 1, "B": 2}},
				{ballot: markus.Ballot[string, uint64]{"A": 1, "B": 2, "C": 3}},
			},
			result: []markus.Result[string]{
				{Choice: "A", Index: 0, Wins: 2},
				{Choice: "B", Index: 1, Wins: 1},
				{Choice: "C", Index: 2, Wins: 0},
			},
		},
		{
			name:    "tie",
			choices: []string{"A", "B", "C"},
			ballots: []ballot[string]{
				{ballot: markus.Ballot[string, uint64]{"A": 1}},
				{ballot: markus.Ballot[string, uint64]{"B": 1}},
			},
			result: []markus.Result[string]{
				{Choice: "A", Index: 0, Wins: 1},
				{Choice: "B", Index: 1, Wins: 1},
				{Choice: "C", Index: 2, Wins: 0},
			},
			tie: true,
		},
		{
			name:    "complex",
			choices: []string{"A", "B", "C", "D", "E"},
			ballots: []ballot[string]{
				{ballot: markus.Ballot[string, uint64]{"A": 1, "B": 1}},
				{ballot: markus.Ballot[string, uint64]{"B": 1, "C": 1, "A": 2}},
				{ballot: markus.Ballot[string, uint64]{"A": 1, "B": 2, "C": 2}},
				{ballot: markus.Ballot[string, uint64]{"A": 1, "B": 200, "C": 10}},
			},
			result: []markus.Result[string]{
				{Choice: "A", Index: 0, Wins: 4},
				{Choice: "B", Index: 1, Wins: 2},
				{Choice: "C", Index: 2, Wins: 2},
				{Choice: "D", Index: 3, Wins: 0},
				{Choice: "E", Index: 4, Wins: 0},
			},
		},
		{
			name:    "duplicate choice",
			choices: []string{"A", "B", "C", "B", "C", "C"},
			ballots: []ballot[string]{
				{ballot: markus.Ballot[string, uint64]{"A": 1, "C": 1}},
				{ballot: markus.Ballot[string, uint64]{"B": 1, "C": 1, "A": 2}},
				{ballot: markus.Ballot[string, uint64]{"B": 2, "C": 3}},
			},
			result: []markus.Result[string]{
				{Choice: "B", Index: 1, Wins: 1},
				{Choice: "C", Index: 2, Wins: 1},
				{Choice: "A", Index: 0, Wins: 0},
			},
			tie: true,
		},
		{
			name:    "example from wiki page",
			choices: []string{"A", "B", "C", "D", "E"},
			ballots: []ballot[string]{
				{ballot: markus.Ballot[string, uint64]{"A": 1, "C": 2, "B": 3, "E": 4, "D": 5}},
				{ballot: markus.Ballot[string, uint64]{"A": 1, "C": 2, "B": 3, "E": 4, "D": 5}},
				{ballot: markus.Ballot[string, uint64]{"A": 1, "C": 2, "B": 3, "E": 4, "D": 5}},
				{ballot: markus.Ballot[string, uint64]{"A": 1, "C": 2, "B": 3, "E": 4, "D": 5}},
				{ballot: markus.Ballot[string, uint64]{"A": 1, "C": 2, "B": 3, "E": 4, "D": 5}},

				{ballot: markus.Ballot[string, uint64]{"A": 1, "D": 2, "E": 3, "C": 4, "B": 5}},
				{ballot: markus.Ballot[string, uint64]{"A": 1, "D": 2, "E": 3, "C": 4, "B": 5}},
				{ballot: markus.Ballot[string, uint64]{"A": 1, "D": 2, "E": 3, "C": 4, "B": 5}},
				{ballot: markus.Ballot[string, uint64]{"A": 1, "D": 2, "E": 3, "C": 4, "B": 5}},
				{ballot: markus.Ballot[string, uint64]{"A": 1, "D": 2, "E": 3, "C": 4, "B": 5}},

				{ballot: markus.Ballot[string, uint64]{"B": 1, "E": 2, "D": 3, "A": 4, "C": 5}},
				{ballot: markus.Ballot[string, uint64]{"B": 1, "E": 2, "D": 3, "A": 4, "C": 5}},
				{ballot: markus.Ballot[string, uint64]{"B": 1, "E": 2, "D": 3, "A": 4, "C": 5}},
				{ballot: markus.Ballot[string, uint64]{"B": 1, "E": 2, "D": 3, "A": 4, "C": 5}},
				{ballot: markus.Ballot[string, uint64]{"B": 1, "E": 2, "D": 3, "A": 4, "C": 5}},
				{ballot: markus.Ballot[string, uint64]{"B": 1, "E": 2, "D": 3, "A": 4, "C": 5}},
				{ballot: markus.Ballot[string, uint64]{"B": 1, "E": 2, "D": 3, "A": 4, "C": 5}},
				{ballot: markus.Ballot[string, uint64]{"B": 1, "E": 2, "D": 3, "A": 4, "C": 5}},

				{ballot: markus.Ballot[string, uint64]{"C": 1, "A": 2, "B": 3, "E": 4, "D": 5}},
				{ballot: markus.Ballot[string, uint64]{"C": 1, "A": 2, "B": 3, "E": 4, "D": 5}},
				{ballot: markus.Ballot[string, uint64]{"C": 1, "A": 2, "B": 3, "E": 4, "D": 5}},

				{ballot: markus.Ballot[string, uint64]{"C": 1, "A": 2, "E": 3, "B": 4, "D": 5}},
				{ballot: markus.Ballot[string, uint64]{"C": 1, "A": 2, "E": 3, "B": 4, "D": 5}},
				{ballot: markus.Ballot[string, uint64]{"C": 1, "A": 2, "E": 3, "B": 4, "D": 5}},
				{ballot: markus.Ballot[string, uint64]{"C": 1, "A": 2, "E": 3, "B": 4, "D": 5}},
				{ballot: markus.Ballot[string, uint64]{"C": 1, "A": 2, "E": 3, "B": 4, "D": 5}},
				{ballot: markus.Ballot[string, uint64]{"C": 1, "A": 2, "E": 3, "B": 4, "D": 5}},
				{ballot: markus.Ballot[string, uint64]{"C": 1, "A": 2, "E": 3, "B": 4, "D": 5}},

				{ballot: markus.Ballot[string, uint64]{"C": 1, "B": 2, "A": 3, "D": 4, "E": 5}},
				{ballot: markus.Ballot[string, uint64]{"C": 1, "B": 2, "A": 3, "D": 4, "E": 5}},

				{ballot: markus.Ballot[string, uint64]{"D": 1, "C": 2, "E": 3, "B": 4, "A": 5}},
				{ballot: markus.Ballot[string, uint64]{"D": 1, "C": 2, "E": 3, "B": 4, "A": 5}},
				{ballot: markus.Ballot[string, uint64]{"D": 1, "C": 2, "E": 3, "B": 4, "A": 5}},
				{ballot: markus.Ballot[string, uint64]{"D": 1, "C": 2, "E": 3, "B": 4, "A": 5}},
				{ballot: markus.Ballot[string, uint64]{"D": 1, "C": 2, "E": 3, "B": 4, "A": 5}},
				{ballot: markus.Ballot[string, uint64]{"D": 1, "C": 2, "E": 3, "B": 4, "A": 5}},
				{ballot: markus.Ballot[string, uint64]{"D": 1, "C": 2, "E": 3, "B": 4, "A": 5}},

				{ballot: markus.Ballot[string, uint64]{"E": 1, "B": 2, "A": 3, "D": 4, "C": 5}},
				{ballot: markus.Ballot[string, uint64]{"E": 1, "B": 2, "A": 3, "D": 4, "C": 5}},
				{ballot: markus.Ballot[string, uint64]{"E": 1, "B": 2, "A": 3, "D": 4, "C": 5}},
				{ballot: markus.Ballot[string, uint64]{"E": 1, "B": 2, "A": 3, "D": 4, "C": 5}},
				{ballot: markus.Ballot[string, uint64]{"E": 1, "B": 2, "A": 3, "D": 4, "C": 5}},
				{ballot: markus.Ballot[string, uint64]{"E": 1, "B": 2, "A": 3, "D": 4, "C": 5}},
				{ballot: markus.Ballot[string, uint64]{"E": 1, "B": 2, "A": 3, "D": 4, "C": 5}},
				{ballot: markus.Ballot[string, uint64]{"E": 1, "B": 2, "A": 3, "D": 4, "C": 5}},
			},
			result: []markus.Result[string]{
				{Choice: "E", Index: 4, Wins: 4},
				{Choice: "A", Index: 0, Wins: 3},
				{Choice: "C", Index: 2, Wins: 2},
				{Choice: "B", Index: 1, Wins: 1},
				{Choice: "D", Index: 3, Wins: 0},
			},
		},
		{
			name:    "unvote single option one vote",
			choices: []string{"A"},
			ballots: []ballot[string]{
				{ballot: markus.Ballot[string, uint64]{"A": 1}},
				{ballot: markus.Ballot[string, uint64]{"A": 1}, unvote: true},
			},
			result: []markus.Result[string]{
				{Choice: "A", Index: 0, Wins: 0},
			},
		},
		{
			name:    "unvote two options one vote",
			choices: []string{"A", "B"},
			ballots: []ballot[string]{
				{ballot: markus.Ballot[string, uint64]{"A": 1}},
				{ballot: markus.Ballot[string, uint64]{"A": 1}, unvote: true},
			},
			result: []markus.Result[string]{
				{Choice: "A", Index: 0, Wins: 0},
				{Choice: "B", Index: 1, Wins: 0},
			},
			tie: true,
		},
		{
			name:    "unvote complex",
			choices: []string{"A", "B", "C", "D", "E"},
			ballots: []ballot[string]{
				{ballot: markus.Ballot[string, uint64]{"A": 1, "B": 1}},
				{ballot: markus.Ballot[string, uint64]{"B": 1, "C": 1, "A": 2}},
				{ballot: markus.Ballot[string, uint64]{"A": 1, "B": 2, "C": 2}},
				{ballot: markus.Ballot[string, uint64]{"A": 1, "B": 200, "C": 10}},
				{ballot: markus.Ballot[string, uint64]{"A": 1, "B": 2, "C": 2}, unvote: true},
			},
			result: []markus.Result[string]{
				{Choice: "A", Index: 0, Wins: 3},
				{Choice: "B", Index: 1, Wins: 2},
				{Choice: "C", Index: 2, Wins: 2},
				{Choice: "D", Index: 3, Wins: 0},
				{Choice: "E", Index: 4, Wins: 0},
			},
		},
		{
			name:    "multiple unvote complex",
			choices: []string{"A", "B", "C", "D", "E"},
			ballots: []ballot[string]{
				{ballot: markus.Ballot[string, uint64]{"A": 1, "B": 1}},
				{ballot: markus.Ballot[string, uint64]{"B": 1, "C": 1, "A": 2}},
				{ballot: markus.Ballot[string, uint64]{"A": 1, "B": 2, "C": 2}},
				{ballot: markus.Ballot[string, uint64]{"A": 1, "B": 1}, unvote: true},
				{ballot: markus.Ballot[string, uint64]{"A": 1, "B": 200, "C": 10}},
				{ballot: markus.Ballot[string, uint64]{"A": 1, "B": 2, "C": 2}, unvote: true},
				{ballot: markus.Ballot[string, uint64]{"B": 1, "C": 1, "A": 2}, unvote: true},
			},
			result: []markus.Result[string]{
				{Choice: "A", Index: 0, Wins: 4},
				{Choice: "C", Index: 2, Wins: 3},
				{Choice: "B", Index: 1, Wins: 2},
				{Choice: "D", Index: 3, Wins: 0},
				{Choice: "E", Index: 4, Wins: 0},
			},
		},
		{
			name:    "multiple unvote cancel compete vote",
			choices: []string{"A", "B"},
			ballots: []ballot[string]{
				{ballot: markus.Ballot[string, uint64]{"A": 1}},
				{ballot: markus.Ballot[string, uint64]{"A": 1}, unvote: true},
				{ballot: markus.Ballot[string, uint64]{"B": 1, "A": 2}},
				{ballot: markus.Ballot[string, uint64]{"B": 1, "A": 2}, unvote: true},
				{ballot: markus.Ballot[string, uint64]{"B": 1}},
			},
			result: []markus.Result[string]{
				{Choice: "B", Index: 1, Wins: 1},
				{Choice: "A", Index: 0, Wins: 0},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			v, err := markus.New[string, uint64](dir)
			if err != nil {
				t.Fatal(err)
			}
			defer v.Close()

			if err := v.Add(tc.choices...); err != nil {
				t.Fatal(err)
			}

			for _, b := range tc.ballots {
				if b.unvote {
					if err := v.Unvote(b.ballot); err != nil {
						t.Fatal(err)
					}
				} else {
					if err := v.Vote(b.ballot); err != nil {
						t.Fatal(err)
					}
				}
			}

			result, tie, _, err := v.ComputeSorted(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			if tie != tc.tie {
				t.Errorf("got tie %v, want %v", tie, tc.tie)
			}
			if !reflect.DeepEqual(result, tc.result) {
				t.Errorf("got result %+v, want %+v", result, tc.result)
			}
		})
	}
}

func TestVoting_persistance(t *testing.T) {
	dir := t.TempDir()
	v, err := markus.New[string, uint64](dir)
	if err != nil {
		t.Fatal(err)
	}
	defer v.Close()

	if err := v.Add("A", "B", "C", "D", "E"); err != nil {
		t.Fatal(err)
	}

	if err := v.Vote(markus.Ballot[string, uint64]{
		"D": 1,
		"B": 5,
	}); err != nil {
		t.Fatal(err)
	}

	if err := v.Vote(markus.Ballot[string, uint64]{
		"C": 1,
	}); err != nil {
		t.Fatal(err)
	}

	results, tie, staled, err := v.ComputeSorted(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	wantResults := []markus.Result[string]{
		{Choice: "D", Index: 3, Wins: 3},
		{Choice: "B", Index: 1, Wins: 2},
		{Choice: "C", Index: 2, Wins: 2},
		{Choice: "A", Index: 0, Wins: 0},
		{Choice: "E", Index: 4, Wins: 0},
	}
	assertEqual(t, "results", results, wantResults)
	wantTie := false
	assertEqual(t, "tie", tie, wantTie)
	wantStaled := false
	assertEqual(t, "staled", staled, wantStaled)

	if err := v.Close(); err != nil {
		t.Fatal(err)
	}

	v2, err := markus.New[string, uint64](dir)
	if err != nil {
		t.Fatal(err)
	}
	defer v2.Close()

	results, tie, staled, err = v2.ComputeSorted(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	assertEqual(t, "results", results, wantResults)
	assertEqual(t, "tie", tie, wantTie)
	assertEqual(t, "staled", staled, wantStaled)
}

func TestVoting_concurrency(t *testing.T) {
	v, err := markus.New[string, uint64](t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer v.Close()

	rand.Seed(time.Now().UnixNano())

	votingLog := make([]any, 0)
	votingLogMu := new(sync.Mutex)

	var (
		concurrency   = runtime.NumCPU()*2 + 1
		iterations    = 200
		choicesCount  = 100
		maxBallotSize = 5
	)

	t.Log("concurrency:", concurrency)

	sem := make(chan struct{}, concurrency)
	var wg sync.WaitGroup
	for i := 0; i < iterations; i++ {
		wg.Add(1)
		sem <- struct{}{}
		go func() {
			defer wg.Done()
			defer func() { <-sem }()

			choices := make([]string, rand.Intn(rand.Intn(maxBallotSize-1)+1))

			for i := range choices {
				choices[i] = fmt.Sprintf("%d", rand.Intn(choicesCount))
			}

			func() {
				votingLogMu.Lock()
				defer votingLogMu.Unlock()

				if err := v.Add(choices...); err != nil {
					t.Error(err)
				}

				votingLog = append(votingLog, choices)
			}()

			ballot := make(markus.Ballot[string, uint64])
			for _, c := range choices {
				ballot[c] = rand.Uint64() % uint64(len(choices)+1)
			}

			func() {
				votingLogMu.Lock()
				defer votingLogMu.Unlock()
				if err := v.Vote(ballot); err != nil {
					t.Error(err)
				}

				votingLog = append(votingLog, ballot)
			}()
		}()

		wg.Add(1)
		sem <- struct{}{}
		go func() {
			defer wg.Done()
			defer func() { <-sem }()
			_, _, _, err := v.ComputeSorted(context.Background())
			if err != nil {
				t.Error(err)
			}
		}()
	}

	wg.Wait()

	gotResults, gotTie, gotStaled, err := v.ComputeSorted(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	assertEqual(t, "staled", gotStaled, false)

	validation, err := markus.New[string, uint64](t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer v.Close()

	for _, m := range votingLog {
		switch m := m.(type) {
		case []string:
			if err := validation.Add(m...); err != nil {
				t.Fatal(err)
			}
		case markus.Ballot[string, uint64]:
			if err := validation.Vote(m); err != nil {
				t.Fatal(err)
			}
		default:
			t.Fatalf("unexpected type %T", m)
		}
	}

	wantResults, wantTie, wantStaled, err := validation.ComputeSorted(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	assertEqual(t, "staled", wantStaled, false)

	assertResultsWins(t, gotResults, wantResults)
	assertEqual(t, "tie", gotTie, wantTie)
}

func BenchmarkVoting_ComputeSorted(b *testing.B) {
	b.Log("creating voting...")
	rand.Seed(time.Now().UnixNano())

	const choicesCount = 1000

	choices := newChoices(choicesCount)

	dir := b.TempDir()
	v, err := markus.New[string, uint64](dir)
	if err != nil {
		b.Fatal(err)
	}
	defer v.Close()

	b.Log("adding choices...")
	if err := v.Add(choices...); err != nil {
		b.Fatal(err)
	}

	b.Log("voting...")
	for i := 0; i < 20; i++ {
		ballot := make(markus.Ballot[string, uint64])
		ballot[choices[rand.Uint64()%choicesCount]] = 1
		ballot[choices[rand.Uint64()%choicesCount]] = 1
		ballot[choices[rand.Uint64()%choicesCount]] = 2
		ballot[choices[rand.Uint64()%choicesCount]] = 3
		ballot[choices[rand.Uint64()%choicesCount]] = 20
		ballot[choices[rand.Uint64()%choicesCount]] = 20
		if err := v.Vote(ballot); err != nil {
			b.Fatal(err)
		}
	}

	b.Log("starting benchmark...")
	b.ResetTimer()

	for n := 0; n < b.N; n++ {
		_, _, _, _ = v.ComputeSorted(context.Background())
	}
}

func assertEqual[T any](t testing.TB, name string, got, want T) {
	t.Helper()

	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %s %+v, want %+v", name, got, want)
	}
}

func assertResultsWins[C comparable](t testing.TB, got, want []markus.Result[C]) {
	t.Helper()

	if len(got) != len(want) {
		t.Fatalf("got %d results, want %d", len(got), len(want))
	}

	for i, g := range got {
		w := want[i]
		if g.Choice != w.Choice {
			t.Errorf("got result %d choice %v, want %v", i, g.Choice, w.Choice)
		}
		if g.Wins != w.Wins {
			t.Errorf("got result %d wins %v, want %v", i, g.Wins, w.Wins)
		}
	}
}

func newChoices(count int) []string {
	choices := make([]string, 0, count)
	for i := 0; i < count; i++ {
		choices = append(choices, strconv.FormatInt(int64(i), 36))
	}
	return choices
}
