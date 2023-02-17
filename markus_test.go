// Copyright (c) 2023, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package markus_test

import (
	"context"
	"math/rand"
	"reflect"
	"runtime"
	"sync"
	"testing"
	"time"

	"resenje.org/markus"
)

func TestVoting(t *testing.T) {

	type ballot[T markus.Type] struct {
		vote   markus.Ballot[T]
		unvote markus.Record
	}
	for _, tc := range []struct {
		name         string
		choicesCount uint64
		ballots      []ballot[uint16]
		result       []markus.Result
		tie          bool
	}{
		{
			name:   "empty",
			result: []markus.Result{},
		},
		{
			name:         "single option no votes",
			choicesCount: 1,
			result: []markus.Result{
				{Index: 0, Wins: 0},
			},
		},
		{
			name:         "single option one vote",
			choicesCount: 1,
			ballots: []ballot[uint16]{
				{vote: markus.Ballot[uint16]{0: 1}},
			},
			result: []markus.Result{
				{Index: 0, Wins: 0},
			},
		},
		{
			name:         "two options one vote",
			choicesCount: 2,
			ballots: []ballot[uint16]{
				{vote: markus.Ballot[uint16]{0: 1}},
			},
			result: []markus.Result{
				{Index: 0, Wins: 1},
				{Index: 1, Wins: 0},
			},
		},
		{
			name:         "two options two votes",
			choicesCount: 2,
			ballots: []ballot[uint16]{
				{vote: markus.Ballot[uint16]{0: 1}},
				{vote: markus.Ballot[uint16]{0: 1, 1: 2}},
			},
			result: []markus.Result{
				{Index: 0, Wins: 1},
				{Index: 1, Wins: 0},
			},
		},
		{
			name:         "three options three votes",
			choicesCount: 3,
			ballots: []ballot[uint16]{
				{vote: markus.Ballot[uint16]{0: 1}},
				{vote: markus.Ballot[uint16]{0: 1, 1: 2}},
				{vote: markus.Ballot[uint16]{0: 1, 1: 2, 2: 3}},
			},
			result: []markus.Result{
				{Index: 0, Wins: 2},
				{Index: 1, Wins: 1},
				{Index: 2, Wins: 0},
			},
		},
		{
			name:         "tie",
			choicesCount: 3,
			ballots: []ballot[uint16]{
				{vote: markus.Ballot[uint16]{0: 1}},
				{vote: markus.Ballot[uint16]{1: 1}},
			},
			result: []markus.Result{
				{Index: 0, Wins: 1},
				{Index: 1, Wins: 1},
				{Index: 2, Wins: 0},
			},
			tie: true,
		},
		{
			name:         "complex",
			choicesCount: 5,
			ballots: []ballot[uint16]{
				{vote: markus.Ballot[uint16]{0: 1, 1: 1}},
				{vote: markus.Ballot[uint16]{1: 1, 2: 1, 0: 2}},
				{vote: markus.Ballot[uint16]{0: 1, 1: 2, 2: 2}},
				{vote: markus.Ballot[uint16]{0: 1, 1: 200, 2: 10}},
			},
			result: []markus.Result{
				{Index: 0, Wins: 4},
				{Index: 1, Wins: 2},
				{Index: 2, Wins: 2},
				{Index: 3, Wins: 0},
				{Index: 4, Wins: 0},
			},
		},
		{
			name:         "example from wiki page",
			choicesCount: 5,
			ballots: []ballot[uint16]{
				{vote: markus.Ballot[uint16]{0: 1, 2: 2, 1: 3, 4: 4, 3: 5}},
				{vote: markus.Ballot[uint16]{0: 1, 2: 2, 1: 3, 4: 4, 3: 5}},
				{vote: markus.Ballot[uint16]{0: 1, 2: 2, 1: 3, 4: 4, 3: 5}},
				{vote: markus.Ballot[uint16]{0: 1, 2: 2, 1: 3, 4: 4, 3: 5}},
				{vote: markus.Ballot[uint16]{0: 1, 2: 2, 1: 3, 4: 4, 3: 5}},

				{vote: markus.Ballot[uint16]{0: 1, 3: 2, 4: 3, 2: 4, 1: 5}},
				{vote: markus.Ballot[uint16]{0: 1, 3: 2, 4: 3, 2: 4, 1: 5}},
				{vote: markus.Ballot[uint16]{0: 1, 3: 2, 4: 3, 2: 4, 1: 5}},
				{vote: markus.Ballot[uint16]{0: 1, 3: 2, 4: 3, 2: 4, 1: 5}},
				{vote: markus.Ballot[uint16]{0: 1, 3: 2, 4: 3, 2: 4, 1: 5}},

				{vote: markus.Ballot[uint16]{1: 1, 4: 2, 3: 3, 0: 4, 2: 5}},
				{vote: markus.Ballot[uint16]{1: 1, 4: 2, 3: 3, 0: 4, 2: 5}},
				{vote: markus.Ballot[uint16]{1: 1, 4: 2, 3: 3, 0: 4, 2: 5}},
				{vote: markus.Ballot[uint16]{1: 1, 4: 2, 3: 3, 0: 4, 2: 5}},
				{vote: markus.Ballot[uint16]{1: 1, 4: 2, 3: 3, 0: 4, 2: 5}},
				{vote: markus.Ballot[uint16]{1: 1, 4: 2, 3: 3, 0: 4, 2: 5}},
				{vote: markus.Ballot[uint16]{1: 1, 4: 2, 3: 3, 0: 4, 2: 5}},
				{vote: markus.Ballot[uint16]{1: 1, 4: 2, 3: 3, 0: 4, 2: 5}},

				{vote: markus.Ballot[uint16]{2: 1, 0: 2, 1: 3, 4: 4, 3: 5}},
				{vote: markus.Ballot[uint16]{2: 1, 0: 2, 1: 3, 4: 4, 3: 5}},
				{vote: markus.Ballot[uint16]{2: 1, 0: 2, 1: 3, 4: 4, 3: 5}},

				{vote: markus.Ballot[uint16]{2: 1, 0: 2, 4: 3, 1: 4, 3: 5}},
				{vote: markus.Ballot[uint16]{2: 1, 0: 2, 4: 3, 1: 4, 3: 5}},
				{vote: markus.Ballot[uint16]{2: 1, 0: 2, 4: 3, 1: 4, 3: 5}},
				{vote: markus.Ballot[uint16]{2: 1, 0: 2, 4: 3, 1: 4, 3: 5}},
				{vote: markus.Ballot[uint16]{2: 1, 0: 2, 4: 3, 1: 4, 3: 5}},
				{vote: markus.Ballot[uint16]{2: 1, 0: 2, 4: 3, 1: 4, 3: 5}},
				{vote: markus.Ballot[uint16]{2: 1, 0: 2, 4: 3, 1: 4, 3: 5}},

				{vote: markus.Ballot[uint16]{2: 1, 1: 2, 0: 3, 3: 4, 4: 5}},
				{vote: markus.Ballot[uint16]{2: 1, 1: 2, 0: 3, 3: 4, 4: 5}},

				{vote: markus.Ballot[uint16]{3: 1, 2: 2, 4: 3, 1: 4, 0: 5}},
				{vote: markus.Ballot[uint16]{3: 1, 2: 2, 4: 3, 1: 4, 0: 5}},
				{vote: markus.Ballot[uint16]{3: 1, 2: 2, 4: 3, 1: 4, 0: 5}},
				{vote: markus.Ballot[uint16]{3: 1, 2: 2, 4: 3, 1: 4, 0: 5}},
				{vote: markus.Ballot[uint16]{3: 1, 2: 2, 4: 3, 1: 4, 0: 5}},
				{vote: markus.Ballot[uint16]{3: 1, 2: 2, 4: 3, 1: 4, 0: 5}},
				{vote: markus.Ballot[uint16]{3: 1, 2: 2, 4: 3, 1: 4, 0: 5}},

				{vote: markus.Ballot[uint16]{4: 1, 1: 2, 0: 3, 3: 4, 2: 5}},
				{vote: markus.Ballot[uint16]{4: 1, 1: 2, 0: 3, 3: 4, 2: 5}},
				{vote: markus.Ballot[uint16]{4: 1, 1: 2, 0: 3, 3: 4, 2: 5}},
				{vote: markus.Ballot[uint16]{4: 1, 1: 2, 0: 3, 3: 4, 2: 5}},
				{vote: markus.Ballot[uint16]{4: 1, 1: 2, 0: 3, 3: 4, 2: 5}},
				{vote: markus.Ballot[uint16]{4: 1, 1: 2, 0: 3, 3: 4, 2: 5}},
				{vote: markus.Ballot[uint16]{4: 1, 1: 2, 0: 3, 3: 4, 2: 5}},
				{vote: markus.Ballot[uint16]{4: 1, 1: 2, 0: 3, 3: 4, 2: 5}},
			},
			result: []markus.Result{
				{Index: 4, Wins: 4},
				{Index: 0, Wins: 3},
				{Index: 2, Wins: 2},
				{Index: 1, Wins: 1},
				{Index: 3, Wins: 0},
			},
		},
		{
			name:         "unvote single option one vote",
			choicesCount: 1,
			ballots: []ballot[uint16]{
				{vote: markus.Ballot[uint16]{0: 1}},
				{unvote: markus.Record{
					Ranks: [][]uint64{{0}},
					Size:  2,
				}},
			},
			result: []markus.Result{
				{Index: 0, Wins: 0},
			},
		},
		{
			name:         "unvote two options one vote",
			choicesCount: 2,
			ballots: []ballot[uint16]{
				{vote: markus.Ballot[uint16]{0: 1}},
				{unvote: markus.Record{
					Ranks: [][]uint64{{0}},
					Size:  2,
				}},
			},
			result: []markus.Result{
				{Index: 0, Wins: 0},
				{Index: 1, Wins: 0},
			},
			tie: true,
		},
		{
			name:         "unvote complex",
			choicesCount: 5,
			ballots: []ballot[uint16]{
				{vote: markus.Ballot[uint16]{0: 1, 1: 1}},
				{vote: markus.Ballot[uint16]{1: 1, 2: 1, 0: 2}},
				{vote: markus.Ballot[uint16]{0: 1, 1: 2, 2: 2}},
				{vote: markus.Ballot[uint16]{0: 1, 1: 200, 2: 10}},
				{unvote: markus.Record{
					Ranks: [][]uint64{{0}, {1, 2}},
					Size:  5,
				}},
			},
			result: []markus.Result{
				{Index: 0, Wins: 3},
				{Index: 1, Wins: 2},
				{Index: 2, Wins: 2},
				{Index: 3, Wins: 0},
				{Index: 4, Wins: 0},
			},
		},
		{
			name:         "multiple unvote complex",
			choicesCount: 5,
			ballots: []ballot[uint16]{
				{vote: markus.Ballot[uint16]{0: 1, 1: 1}},
				{vote: markus.Ballot[uint16]{1: 1, 2: 1, 0: 2}},
				{vote: markus.Ballot[uint16]{0: 1, 1: 2, 2: 2}},
				{unvote: markus.Record{
					Ranks: [][]uint64{{0, 1}},
					Size:  5,
				}},
				{vote: markus.Ballot[uint16]{0: 1, 1: 200, 2: 10}},
				{unvote: markus.Record{
					Ranks: [][]uint64{{0}, {1, 2}},
					Size:  5,
				}},
				{unvote: markus.Record{
					Ranks: [][]uint64{{1, 2}, {0}},
					Size:  5,
				}},
			},
			result: []markus.Result{
				{Index: 0, Wins: 4},
				{Index: 2, Wins: 3},
				{Index: 1, Wins: 2},
				{Index: 3, Wins: 0},
				{Index: 4, Wins: 0},
			},
		},
		{
			name:         "multiple unvote cancel compete vote",
			choicesCount: 2,
			ballots: []ballot[uint16]{
				{vote: markus.Ballot[uint16]{0: 1}},
				{unvote: markus.Record{
					Ranks: [][]uint64{{0}},
					Size:  2,
				}},
				{vote: markus.Ballot[uint16]{1: 1, 0: 2}},
				{unvote: markus.Record{
					Ranks: [][]uint64{{1}, {0}},
					Size:  2,
				}},
				{vote: markus.Ballot[uint16]{1: 1}},
			},
			result: []markus.Result{
				{Index: 1, Wins: 1},
				{Index: 0, Wins: 0},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			v := newMarkusVoting[uint16](t)

			if _, _, err := v.Add(tc.choicesCount); err != nil {
				t.Fatal(err)
			}

			for _, b := range tc.ballots {
				if b.unvote.Size > 0 {
					if err := v.Unvote(b.unvote); err != nil {
						t.Fatal(err)
					}
				} else {
					if _, err := v.Vote(b.vote); err != nil {
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

func TestVoting_Unvote_afterAdd(t *testing.T) {
	v := newMarkusVoting[uint64](t)

	if _, _, err := v.Add(3); err != nil {
		t.Fatal(err)
	}

	ballot := markus.Ballot[uint64]{0: 1, 1: 2}
	record, err := v.Vote(ballot)
	if err != nil {
		t.Fatal(err)
	}

	if _, _, err := v.Add(1); err != nil {
		t.Fatal(err)
	}

	gotResults, tie, stale, err := v.ComputeSorted(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	assertEqual(t, "gotResults", gotResults, []markus.Result{
		{Index: 0, Wins: 2},
		{Index: 1, Wins: 1},
		{Index: 2, Wins: 0},
		{Index: 3, Wins: 0},
	})
	assertEqual(t, "tie", tie, false)
	assertEqual(t, "stale", stale, false)

	if err := v.Unvote(record); err != nil {
		t.Fatal(err)
	}

	gotResults, tie, stale, err = v.ComputeSorted(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	assertEqual(t, "gotResults", gotResults, []markus.Result{
		{Index: 0, Wins: 0},
		{Index: 1, Wins: 0},
		{Index: 2, Wins: 0},
		{Index: 3, Wins: 0},
	})
	assertEqual(t, "tie", tie, true)
	assertEqual(t, "stale", stale, false)
}

func TestVoting_persistance(t *testing.T) {
	dir := t.TempDir()
	v, err := markus.New[uint16](dir)
	if err != nil {
		t.Fatal(err)
	}
	defer v.Close()

	if _, _, err := v.Add(5); err != nil {
		t.Fatal(err)
	}

	if _, err := v.Vote(markus.Ballot[uint16]{
		3: 1,
		1: 5,
	}); err != nil {
		t.Fatal(err)
	}

	if _, err := v.Vote(markus.Ballot[uint16]{
		2: 1,
	}); err != nil {
		t.Fatal(err)
	}

	results, tie, staled, err := v.ComputeSorted(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	wantResults := []markus.Result{
		{Index: 3, Wins: 3},
		{Index: 1, Wins: 2},
		{Index: 2, Wins: 2},
		{Index: 0, Wins: 0},
		{Index: 4, Wins: 0},
	}
	assertEqual(t, "results", results, wantResults)
	wantTie := false
	assertEqual(t, "tie", tie, wantTie)
	wantStaled := false
	assertEqual(t, "staled", staled, wantStaled)

	if err := v.Close(); err != nil {
		t.Fatal(err)
	}

	v2, err := markus.New[uint16](dir)
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
	v := newMarkusVoting[uint64](t)

	votingLog := make([]any, 0)
	votingLogMu := new(sync.Mutex)

	var (
		concurrency          = runtime.NumCPU()*2 + 1
		iterations           = 100
		choicesCount  uint64 = 100
		maxBallotSize        = 25
	)

	t.Log("concurrency:", concurrency)

	addTimes := make([]time.Duration, 0)
	addTimesMu := new(sync.Mutex)
	voteTimes := make([]time.Duration, 0)
	voteTimesMu := new(sync.Mutex)
	computeTimes := make([]time.Duration, 0)
	computeTimesMu := new(sync.Mutex)

	sem := make(chan struct{}, concurrency)
	var wg sync.WaitGroup
	for i := 0; i < iterations; i++ {
		if i%(iterations/100) == 0 {
			t.Logf("progress: %.0f %%", float64(i)/float64(iterations)*100)
			addTimesMu.Lock()
			t.Logf("add choices: %v per choice", avgDuration(addTimes))
			addTimes = addTimes[:0]
			addTimesMu.Unlock()

			voteTimesMu.Lock()
			t.Logf("vote: %v per choice", avgDuration(voteTimes))
			voteTimes = voteTimes[:0]
			voteTimesMu.Unlock()

			computeTimesMu.Lock()
			t.Logf("compute: %v per choice", avgDuration(computeTimes))
			computeTimes = computeTimes[:0]
			computeTimesMu.Unlock()
		}
		wg.Add(1)
		sem <- struct{}{}
		go func() {
			defer wg.Done()
			defer func() { <-sem }()

			votingLogMu.Lock()
			defer votingLogMu.Unlock()

			choices := make([]uint64, rand.Intn(rand.Intn(maxBallotSize)+1)+1)

			var maxChoice uint64
			for i := range choices {
				c := rand.Uint64() % choicesCount
				choices[i] = c
				if c > maxChoice {
					maxChoice = c
				}
			}

			size := v.Size()
			func() {

				count := maxChoice - size + 1
				if count <= 0 {
					return
				}

				start := time.Now()
				if _, _, err := v.Add(count); err != nil {
					t.Error(err)
				}
				if size > 0 {
					addTimesMu.Lock()
					addTimes = append(addTimes, time.Since(start)/time.Duration(size))
					addTimesMu.Unlock()
				}

				votingLog = append(votingLog, count)
			}()

			ballot := make(markus.Ballot[uint64])
			for _, c := range choices {
				ballot[c] = rand.Uint64() % uint64(len(choices)+1)
			}

			func() {

				start := time.Now()
				if _, err := v.Vote(ballot); err != nil {
					t.Error(err)
				}
				if size > 0 {
					voteTimesMu.Lock()
					voteTimes = append(voteTimes, time.Since(start)/time.Duration(size))
					voteTimesMu.Unlock()
				}

				votingLog = append(votingLog, ballot)
			}()
		}()

		wg.Add(1)
		sem <- struct{}{}
		go func() {
			defer wg.Done()
			defer func() { <-sem }()

			start := time.Now()
			results, _, stale, err := v.ComputeSorted(context.Background())
			if err != nil {
				t.Error(err)
			}
			if len(results) == 0 {
				return
			}
			if stale {
				t.Log("stale")
			}
			computeTimesMu.Lock()
			computeTimes = append(computeTimes, time.Since(start)/time.Duration(len(results)))
			computeTimesMu.Unlock()
		}()
	}

	wg.Wait()

	gotResults, gotTie, gotStaled, err := v.ComputeSorted(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	assertEqual(t, "staled", gotStaled, false)

	validation, err := markus.New[uint64](t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer v.Close()

	for _, m := range votingLog {
		switch m := m.(type) {
		case uint64:
			if _, _, err := validation.Add(m); err != nil {
				t.Fatal(err)
			}
		case markus.Ballot[uint64]:
			if _, err := validation.Vote(m); err != nil {
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

	assertEqual(t, "results", gotResults, wantResults)
	assertEqual(t, "tie", gotTie, wantTie)
}

func BenchmarkVoting_ComputeSorted(b *testing.B) {
	b.Log("creating voting...")

	const choicesCount = 1000

	v := newMarkusVoting[uint64](b)

	b.Log("adding choices...")
	if _, _, err := v.Add(choicesCount); err != nil {
		b.Fatal(err)
	}

	b.Log("voting...")
	for i := 0; i < 20; i++ {
		ballot := make(markus.Ballot[uint64])
		ballot[rand.Uint64()%choicesCount] = 1
		ballot[rand.Uint64()%choicesCount] = 1
		ballot[rand.Uint64()%choicesCount] = 2
		ballot[rand.Uint64()%choicesCount] = 3
		ballot[rand.Uint64()%choicesCount] = 20
		ballot[rand.Uint64()%choicesCount] = 20
		if _, err := v.Vote(ballot); err != nil {
			b.Fatal(err)
		}
	}

	b.Log("starting benchmark...")
	b.ResetTimer()

	for n := 0; n < b.N; n++ {
		_, _, _, _ = v.ComputeSorted(context.Background())
	}
}

func BenchmarkVoting_Vote(b *testing.B) {
	b.Log("creating voting...")

	const choicesCount = 100

	v := newMarkusVoting[uint64](b)

	b.Log("adding choices...")
	if _, _, err := v.Add(choicesCount); err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()

	for n := 0; n < b.N; n++ {
		if _, err := v.Vote(markus.Ballot[uint64]{
			choicesCount / 2: 1,
		}); err != nil {
			b.Fatal(err)
		}
	}
}

func assertEqual[T any](t testing.TB, name string, got, want T) {
	t.Helper()

	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %s %+v, want %+v", name, got, want)
	}
}

func avgDuration(durations []time.Duration) time.Duration {
	if len(durations) == 0 {
		return 0
	}
	var sum time.Duration
	for _, d := range durations {
		sum += d
	}
	return sum / time.Duration(len(durations))
}

func newMarkusVoting[T markus.Type](t testing.TB) *markus.Voting[T] {
	t.Helper()

	dir := t.TempDir()

	v, err := markus.New[T](dir)
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() {
		if err := v.Close(); err != nil {
			t.Error(err)
		}
	})
	return v
}
