// Copyright (c) 2023, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package markus_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"os"
	"reflect"
	"runtime"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"resenje.org/markus"
)

var verbose = strings.ToLower(os.Getenv("VERBOSE")) == "true"

func TestVoting(t *testing.T) {

	type ballot struct {
		vote   markus.Ballot
		unvote markus.Record
	}
	for _, tc := range []struct {
		name         string
		choicesCount uint64
		ballots      []ballot
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
				{Index: 0, Wins: 0, Strength: 0, Advantage: 0},
			},
		},
		{
			name:         "single option one vote",
			choicesCount: 1,
			ballots: []ballot{
				{vote: markus.Ballot{0: 1}},
			},
			result: []markus.Result{
				{Index: 0, Wins: 0, Strength: 0, Advantage: 0},
			},
		},
		{
			name:         "two options one vote",
			choicesCount: 2,
			ballots: []ballot{
				{vote: markus.Ballot{0: 1}},
			},
			result: []markus.Result{
				{Index: 0, Wins: 1, Strength: 1, Advantage: 1},
				{Index: 1, Wins: 0, Strength: 0, Advantage: 0},
			},
		},
		{
			name:         "two options two votes",
			choicesCount: 2,
			ballots: []ballot{
				{vote: markus.Ballot{0: 1}},
				{vote: markus.Ballot{0: 1, 1: 2}},
			},
			result: []markus.Result{
				{Index: 0, Wins: 1, Strength: 2, Advantage: 2},
				{Index: 1, Wins: 0, Strength: 0, Advantage: 0}},
		},
		{
			name:         "three options three votes",
			choicesCount: 3,
			ballots: []ballot{
				{vote: markus.Ballot{0: 1}},
				{vote: markus.Ballot{0: 1, 1: 2}},
				{vote: markus.Ballot{0: 1, 1: 2, 2: 3}},
			},
			result: []markus.Result{
				{Index: 0, Wins: 2, Strength: 6, Advantage: 6},
				{Index: 1, Wins: 1, Strength: 2, Advantage: 2},
				{Index: 2, Wins: 0, Strength: 0, Advantage: 0},
			},
		},
		{
			name:         "tie",
			choicesCount: 3,
			ballots: []ballot{
				{vote: markus.Ballot{0: 1}},
				{vote: markus.Ballot{1: 1}},
			},
			result: []markus.Result{
				{Index: 0, Wins: 1, Strength: 1, Advantage: 1},
				{Index: 1, Wins: 1, Strength: 1, Advantage: 1},
				{Index: 2, Wins: 0, Strength: 0, Advantage: 0},
			},
			tie: true,
		},
		{
			name:         "complex",
			choicesCount: 5,
			ballots: []ballot{
				{vote: markus.Ballot{0: 1, 1: 1}},
				{vote: markus.Ballot{1: 1, 2: 1, 0: 2}},
				{vote: markus.Ballot{0: 1, 1: 2, 2: 2}},
				{vote: markus.Ballot{0: 1, 1: 200, 2: 10}},
			},
			result: []markus.Result{
				{Index: 0, Wins: 4, Strength: 13, Advantage: 13},
				{Index: 1, Wins: 2, Strength: 8, Advantage: 8},
				{Index: 2, Wins: 2, Strength: 6, Advantage: 6},
				{Index: 3, Wins: 0, Strength: 0, Advantage: 0},
				{Index: 4, Wins: 0, Strength: 0, Advantage: 0},
			},
		},
		{
			name:         "example from wiki page",
			choicesCount: 5,
			ballots: []ballot{
				{vote: markus.Ballot{0: 1, 2: 2, 1: 3, 4: 4, 3: 5}},
				{vote: markus.Ballot{0: 1, 2: 2, 1: 3, 4: 4, 3: 5}},
				{vote: markus.Ballot{0: 1, 2: 2, 1: 3, 4: 4, 3: 5}},
				{vote: markus.Ballot{0: 1, 2: 2, 1: 3, 4: 4, 3: 5}},
				{vote: markus.Ballot{0: 1, 2: 2, 1: 3, 4: 4, 3: 5}},

				{vote: markus.Ballot{0: 1, 3: 2, 4: 3, 2: 4, 1: 5}},
				{vote: markus.Ballot{0: 1, 3: 2, 4: 3, 2: 4, 1: 5}},
				{vote: markus.Ballot{0: 1, 3: 2, 4: 3, 2: 4, 1: 5}},
				{vote: markus.Ballot{0: 1, 3: 2, 4: 3, 2: 4, 1: 5}},
				{vote: markus.Ballot{0: 1, 3: 2, 4: 3, 2: 4, 1: 5}},

				{vote: markus.Ballot{1: 1, 4: 2, 3: 3, 0: 4, 2: 5}},
				{vote: markus.Ballot{1: 1, 4: 2, 3: 3, 0: 4, 2: 5}},
				{vote: markus.Ballot{1: 1, 4: 2, 3: 3, 0: 4, 2: 5}},
				{vote: markus.Ballot{1: 1, 4: 2, 3: 3, 0: 4, 2: 5}},
				{vote: markus.Ballot{1: 1, 4: 2, 3: 3, 0: 4, 2: 5}},
				{vote: markus.Ballot{1: 1, 4: 2, 3: 3, 0: 4, 2: 5}},
				{vote: markus.Ballot{1: 1, 4: 2, 3: 3, 0: 4, 2: 5}},
				{vote: markus.Ballot{1: 1, 4: 2, 3: 3, 0: 4, 2: 5}},

				{vote: markus.Ballot{2: 1, 0: 2, 1: 3, 4: 4, 3: 5}},
				{vote: markus.Ballot{2: 1, 0: 2, 1: 3, 4: 4, 3: 5}},
				{vote: markus.Ballot{2: 1, 0: 2, 1: 3, 4: 4, 3: 5}},

				{vote: markus.Ballot{2: 1, 0: 2, 4: 3, 1: 4, 3: 5}},
				{vote: markus.Ballot{2: 1, 0: 2, 4: 3, 1: 4, 3: 5}},
				{vote: markus.Ballot{2: 1, 0: 2, 4: 3, 1: 4, 3: 5}},
				{vote: markus.Ballot{2: 1, 0: 2, 4: 3, 1: 4, 3: 5}},
				{vote: markus.Ballot{2: 1, 0: 2, 4: 3, 1: 4, 3: 5}},
				{vote: markus.Ballot{2: 1, 0: 2, 4: 3, 1: 4, 3: 5}},
				{vote: markus.Ballot{2: 1, 0: 2, 4: 3, 1: 4, 3: 5}},

				{vote: markus.Ballot{2: 1, 1: 2, 0: 3, 3: 4, 4: 5}},
				{vote: markus.Ballot{2: 1, 1: 2, 0: 3, 3: 4, 4: 5}},

				{vote: markus.Ballot{3: 1, 2: 2, 4: 3, 1: 4, 0: 5}},
				{vote: markus.Ballot{3: 1, 2: 2, 4: 3, 1: 4, 0: 5}},
				{vote: markus.Ballot{3: 1, 2: 2, 4: 3, 1: 4, 0: 5}},
				{vote: markus.Ballot{3: 1, 2: 2, 4: 3, 1: 4, 0: 5}},
				{vote: markus.Ballot{3: 1, 2: 2, 4: 3, 1: 4, 0: 5}},
				{vote: markus.Ballot{3: 1, 2: 2, 4: 3, 1: 4, 0: 5}},
				{vote: markus.Ballot{3: 1, 2: 2, 4: 3, 1: 4, 0: 5}},

				{vote: markus.Ballot{4: 1, 1: 2, 0: 3, 3: 4, 2: 5}},
				{vote: markus.Ballot{4: 1, 1: 2, 0: 3, 3: 4, 2: 5}},
				{vote: markus.Ballot{4: 1, 1: 2, 0: 3, 3: 4, 2: 5}},
				{vote: markus.Ballot{4: 1, 1: 2, 0: 3, 3: 4, 2: 5}},
				{vote: markus.Ballot{4: 1, 1: 2, 0: 3, 3: 4, 2: 5}},
				{vote: markus.Ballot{4: 1, 1: 2, 0: 3, 3: 4, 2: 5}},
				{vote: markus.Ballot{4: 1, 1: 2, 0: 3, 3: 4, 2: 5}},
				{vote: markus.Ballot{4: 1, 1: 2, 0: 3, 3: 4, 2: 5}},
			},
			result: []markus.Result{
				{Index: 4, Wins: 4, Strength: 112, Advantage: 16},
				{Index: 0, Wins: 3, Strength: 86, Advantage: 11},
				{Index: 2, Wins: 2, Strength: 58, Advantage: 2},
				{Index: 1, Wins: 1, Strength: 33, Advantage: 5},
				{Index: 3, Wins: 0, Strength: 0, Advantage: 0},
			},
		},
		{
			name:         "unvote single option one vote",
			choicesCount: 1,
			ballots: []ballot{
				{vote: markus.Ballot{0: 1}},
				{unvote: [][]uint64{{0}}},
			},
			result: []markus.Result{
				{Index: 0, Wins: 0, Strength: 0, Advantage: 0},
			},
		},
		{
			name:         "unvote two options one vote",
			choicesCount: 2,
			ballots: []ballot{
				{vote: markus.Ballot{0: 1}},
				{unvote: [][]uint64{{0}, {1}}},
			},
			result: []markus.Result{
				{Index: 0, Wins: 0, Strength: 0, Advantage: 0},
				{Index: 1, Wins: 0, Strength: 0, Advantage: 0},
			},
			tie: true,
		},
		{
			name:         "unvote complex",
			choicesCount: 5,
			ballots: []ballot{
				{vote: markus.Ballot{0: 1, 1: 1}},
				{vote: markus.Ballot{1: 1, 2: 1, 0: 2}},
				{vote: markus.Ballot{0: 1, 1: 2, 2: 2}},
				{vote: markus.Ballot{0: 1, 1: 200, 2: 10}},
				{unvote: [][]uint64{{0}, {1, 2}, {3, 4}}},
			},
			result: []markus.Result{
				{Index: 0, Wins: 3, Strength: 8, Advantage: 8},
				{Index: 1, Wins: 2, Strength: 6, Advantage: 6},
				{Index: 2, Wins: 2, Strength: 4, Advantage: 4},
				{Index: 3, Wins: 0, Strength: 0, Advantage: 0},
				{Index: 4, Wins: 0, Strength: 0, Advantage: 0},
			},
		},
		{
			name:         "multiple unvote complex",
			choicesCount: 5,
			ballots: []ballot{
				{vote: markus.Ballot{0: 1, 1: 1}},
				{vote: markus.Ballot{1: 1, 2: 1, 0: 2}},
				{vote: markus.Ballot{0: 1, 1: 2, 2: 2}},
				{unvote: [][]uint64{{0, 1}, {2, 3, 4}}},
				{vote: markus.Ballot{0: 1, 1: 200, 2: 10}},
				{unvote: [][]uint64{{0}, {1, 2}, {3, 4}}},
				{unvote: [][]uint64{{1, 2}, {0}, {3, 4}}},
			},
			result: []markus.Result{
				{Index: 0, Wins: 4, Strength: 4, Advantage: 4},
				{Index: 2, Wins: 3, Strength: 3, Advantage: 3},
				{Index: 1, Wins: 2, Strength: 2, Advantage: 2},
				{Index: 3, Wins: 0, Strength: 0, Advantage: 0},
				{Index: 4, Wins: 0, Strength: 0, Advantage: 0},
			},
		},
		{
			name:         "multiple unvote cancel compete vote",
			choicesCount: 2,
			ballots: []ballot{
				{vote: markus.Ballot{0: 1}},
				{unvote: [][]uint64{{0}, {1}}},
				{vote: markus.Ballot{1: 1, 0: 2}},
				{unvote: [][]uint64{{1}, {0}, {}}},
				{vote: markus.Ballot{1: 1}},
			},
			result: []markus.Result{
				{Index: 1, Wins: 1, Strength: 1, Advantage: 1},
				{Index: 0, Wins: 0, Strength: 0, Advantage: 0},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Run("Compute", func(t *testing.T) {
				v := newMarkusVoting(t)

				if _, _, err := v.AddChoices(tc.choicesCount); err != nil {
					t.Fatal(err)
				}

				for _, b := range tc.ballots {
					if len(b.unvote) > 0 {
						if err := v.Unvote(b.unvote); err != nil {
							t.Fatal(err)
						}
					} else {
						if _, err := v.Vote(b.vote); err != nil {
							t.Fatal(err)
						}
					}
				}

				results := make([]markus.Result, 0)
				stale, err := v.Compute(context.Background(), func(r markus.Result) (bool, error) {
					results = append(results, r)
					return true, nil
				})

				sort.Slice(results, func(i, j int) bool {
					if results[i].Wins != results[j].Wins {
						return results[i].Wins > results[j].Wins
					}
					if results[i].Strength != results[j].Strength {
						return results[i].Strength > results[j].Strength
					}
					return results[i].Index < results[j].Index
				})

				if err != nil {
					t.Fatal(err)
				}
				assertEqual(t, "stale", stale, false)
				assertEqual(t, "result", results, tc.result)
			})
			t.Run("ComputeSorted", func(t *testing.T) {
				v := newMarkusVoting(t)

				if _, _, err := v.AddChoices(tc.choicesCount); err != nil {
					t.Fatal(err)
				}

				for _, b := range tc.ballots {
					if len(b.unvote) > 0 {
						if err := v.Unvote(b.unvote); err != nil {
							t.Fatal(err)
						}
					} else {
						if _, err := v.Vote(b.vote); err != nil {
							t.Fatal(err)
						}
					}
				}

				results, tie, stale, err := v.ComputeSorted(context.Background())
				if err != nil {
					t.Fatal(err)
				}
				assertEqual(t, "stale", stale, false)
				assertEqual(t, "tie", tie, tc.tie)
				assertEqual(t, "result", results, tc.result)
			})
		})
	}
}

func TestVoting_Unvote_afterAddChoices(t *testing.T) {
	t.Run("with unvoted", func(t *testing.T) {
		v := newMarkusVoting(t)
		if _, _, err := v.AddChoices(3); err != nil {
			t.Fatal(err)
		}

		validationVoting := newMarkusVoting(t)
		if _, _, err := validationVoting.AddChoices(3); err != nil {
			t.Fatal(err)
		}

		ballot := markus.Ballot{0: 1, 1: 2}
		record, err := v.Vote(ballot)
		if err != nil {
			t.Fatal(err)
		}

		if _, _, err := v.AddChoices(1); err != nil {
			t.Fatal(err)
		}
		if _, _, err := validationVoting.AddChoices(1); err != nil {
			t.Fatal(err)
		}

		if err := v.Unvote(record); err != nil {
			t.Fatal(err)
		}

		assertPreferences(t, v, validationVoting)
	})

	t.Run("without unvoted", func(t *testing.T) {
		v := newMarkusVoting(t)
		if _, _, err := v.AddChoices(3); err != nil {
			t.Fatal(err)
		}

		validationVoting := newMarkusVoting(t)
		if _, _, err := validationVoting.AddChoices(3); err != nil {
			t.Fatal(err)
		}

		ballot := markus.Ballot{0: 1, 1: 2, 2: 1}
		record, err := v.Vote(ballot)
		if err != nil {
			t.Fatal(err)
		}

		if _, _, err := v.AddChoices(1); err != nil {
			t.Fatal(err)
		}
		if _, _, err := validationVoting.AddChoices(1); err != nil {
			t.Fatal(err)
		}

		if err := v.Unvote(record); err != nil {
			t.Fatal(err)
		}

		assertPreferences(t, v, validationVoting)
	})
}

func TestVoting_Unvote_afterRemoveChoices(t *testing.T) {
	t.Run("with unvoted", func(t *testing.T) {
		v := newMarkusVoting(t)
		if _, _, err := v.AddChoices(8); err != nil {
			t.Fatal(err)
		}

		validationVoting := newMarkusVoting(t)
		if _, _, err := validationVoting.AddChoices(8); err != nil {
			t.Fatal(err)
		}

		ballot := markus.Ballot{0: 1, 1: 2, 4: 3, 7: 4}
		record, err := v.Vote(ballot)
		if err != nil {
			t.Fatal(err)
		}

		if err := validationVoting.RemoveChoices(4, 5); err != nil {
			t.Fatal(err)
		}

		if err := v.RemoveChoices(4, 5); err != nil {
			t.Fatal(err)
		}

		if err := v.Unvote(record); err != nil {
			t.Fatal(err)
		}

		assertPreferences(t, v, validationVoting)
	})

	t.Run("without unvoted", func(t *testing.T) {
		v := newMarkusVoting(t)
		if _, _, err := v.AddChoices(8); err != nil {
			t.Fatal(err)
		}

		validationVoting := newMarkusVoting(t)
		if _, _, err := validationVoting.AddChoices(8); err != nil {
			t.Fatal(err)
		}

		ballot := markus.Ballot{0: 1, 1: 2, 3: 1, 4: 3, 5: 2, 6: 2, 7: 4}
		record, err := v.Vote(ballot)
		if err != nil {
			t.Fatal(err)
		}

		if err := validationVoting.RemoveChoices(4, 5); err != nil {
			t.Fatal(err)
		}

		if err := v.RemoveChoices(4, 5); err != nil {
			t.Fatal(err)
		}

		if err := v.Unvote(record); err != nil {
			t.Fatal(err)
		}

		assertPreferences(t, v, validationVoting)
	})
}

func TestVoting_persistance(t *testing.T) {
	dir := t.TempDir()
	v, err := markus.NewVoting(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer v.Close()

	if _, _, err := v.AddChoices(5); err != nil {
		t.Fatal(err)
	}

	if _, err := v.Vote(markus.Ballot{
		3: 1,
		1: 5,
	}); err != nil {
		t.Fatal(err)
	}

	if _, err := v.Vote(markus.Ballot{
		2: 1,
	}); err != nil {
		t.Fatal(err)
	}

	if err := v.RemoveChoices(4); err != nil {
		t.Fatal(err)
	}

	results, tie, staled, err := v.ComputeSorted(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	wantResults := []markus.Result{
		{Index: 3, Wins: 2, Strength: 2, Advantage: 2},
		{Index: 1, Wins: 1, Strength: 1, Advantage: 1},
		{Index: 2, Wins: 1, Strength: 1, Advantage: 1},
		{Index: 0, Wins: 0, Strength: 0, Advantage: 0},
	}
	assertEqual(t, "results", results, wantResults)
	wantTie := false
	assertEqual(t, "tie", tie, wantTie)
	wantStaled := false
	assertEqual(t, "staled", staled, wantStaled)

	if err := v.Close(); err != nil {
		t.Fatal(err)
	}

	v2, err := markus.NewVoting(dir)
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

func TestVoting_addVoteAndUnvote(t *testing.T) {
	v, err := markus.NewVoting(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer v.Close()

	_, _, err = v.AddChoices(1)
	assertEqual(t, "error", err, nil)
	_, err = v.Vote(markus.Ballot{
		0: 1,
	})
	assertEqual(t, "error", err, nil)

	_, _, err = v.AddChoices(1)
	assertEqual(t, "error", err, nil)
	r2, err := v.Vote(markus.Ballot{
		1: 1,
	})
	assertEqual(t, "error", err, nil)

	_, _, err = v.AddChoices(1)
	assertEqual(t, "error", err, nil)
	_, err = v.Vote(markus.Ballot{
		2: 1,
	})
	assertEqual(t, "error", err, nil)

	err = v.Unvote(r2)
	assertEqual(t, "error", err, nil)
	_, err = v.Vote(markus.Ballot{
		1: 1,
	})
	assertEqual(t, "error", err, nil)

	_, _, err = v.AddChoices(1)
	assertEqual(t, "error", err, nil)
	_, err = v.Vote(markus.Ballot{
		3: 1,
	})
	assertEqual(t, "error", err, nil)

	results, tie, staled, err := v.ComputeSorted(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	wantResults := []markus.Result{
		{Index: 0, Wins: 0, Strength: 0, Advantage: 0},
		{Index: 1, Wins: 0, Strength: 0, Advantage: 0},
		{Index: 2, Wins: 0, Strength: 0, Advantage: 0},
		{Index: 3, Wins: 0, Strength: 0, Advantage: 0},
	}
	assertEqual(t, "results", results, wantResults)
	wantTie := true
	assertEqual(t, "tie", tie, wantTie)
	wantStaled := false
	assertEqual(t, "staled", staled, wantStaled)
}

func TestVoting_concurrency(t *testing.T) {
	v := newMarkusVoting(t)

	seed := time.Now().UnixNano()
	t.Log("randomness seed", seed)
	random := rand.New(rand.NewSource(seed))

	mu := new(sync.Mutex)
	votingLog := make([]any, 0)
	recordsToRemove := make([]markus.Record, 0)

	var (
		concurrency         = runtime.NumCPU()*2 + 1
		iterations          = 100
		choicesCount uint64 = 10
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

			mu.Lock()
			defer mu.Unlock()

			size := v.Size()

			count := random.Uint64() % choicesCount
			if count <= 0 {
				return
			}

			start := time.Now()
			if _, _, err := v.AddChoices(count); err != nil {
				t.Error(err)
			}
			if size > 0 {
				addTimesMu.Lock()
				addTimes = append(addTimes, time.Since(start)/time.Duration(size))
				addTimesMu.Unlock()
			}

			votingLog = append(votingLog, count)
		}()

		wg.Add(1)
		sem <- struct{}{}
		go func() {
			defer wg.Done()
			defer func() { <-sem }()

			mu.Lock()
			defer mu.Unlock()

			choices := v.Choices()

			if len(choices) == 0 {
				return
			}

			toRemove := make([]uint64, random.Intn(3))
			for i := range toRemove {
				toRemove[i] = choices[random.Intn(len(choices))]
			}

			if err := v.RemoveChoices(toRemove...); err != nil {
				t.Error(err)
			}

			votingLog = append(votingLog, toRemove)
		}()

		wg.Add(1)
		sem <- struct{}{}
		go func() {
			defer wg.Done()
			defer func() { <-sem }()

			mu.Lock()
			defer mu.Unlock()

			ballot := make(markus.Ballot)
			for _, c := range v.Choices() {
				if random.Uint32()%2 == 0 {
					continue
				}
				ballot[c] = random.Uint32() % 10
			}

			size := v.Size()
			start := time.Now()
			record, err := v.Vote(ballot)
			if err != nil {
				var e *markus.UnknownChoiceError
				if errors.As(err, &e) {
					// do not care about racing remove choice and vote
					return
				}
				t.Error("vote", err)
			}
			if size > 0 {
				voteTimesMu.Lock()
				voteTimes = append(voteTimes, time.Since(start)/time.Duration(size))
				voteTimesMu.Unlock()
			}

			recordsToRemove = append(recordsToRemove, record)
			votingLog = append(votingLog, ballot)
		}()

		wg.Add(1)
		sem <- struct{}{}
		go func() {
			defer wg.Done()
			defer func() { <-sem }()

			mu.Lock()
			defer mu.Unlock()

			if random.Uint32()%2 == 0 || len(recordsToRemove) == 0 {
				return
			}

			index := random.Intn(len(recordsToRemove))
			record := recordsToRemove[index]

			if err := v.Unvote(record); err != nil {
				t.Error("unvote", err)
			}
			recordsToRemove = append(recordsToRemove[:index], recordsToRemove[index+1:]...)

			votingLog = append(votingLog, record)
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

	validation := newMarkusVoting(t)

	for _, m := range votingLog {
		switch m := m.(type) {
		case uint64:
			if _, _, err := validation.AddChoices(m); err != nil {
				t.Fatal(err)
			}
		case []uint64:
			if err := validation.RemoveChoices(m...); err != nil {
				t.Fatal(err)
			}
		case markus.Ballot:
			if _, err := validation.Vote(m); err != nil {
				var e *markus.UnknownChoiceError
				if errors.As(err, &e) {
					// do not care about racing remove choice and vote
					continue
				}
				t.Fatal(err)
			}
		case markus.Record:
			if err := validation.Unvote(m); err != nil {
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

func TestVoting_strengthsMatrixPreparation(t *testing.T) {
	dir := t.TempDir()

	v, err := markus.NewVoting(dir)
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() {
		if err := v.Close(); err != nil {
			t.Error(err)
		}
	})

	const choicesCount = 34

	if _, _, err := v.AddChoices(choicesCount); err != nil {
		t.Fatal(err)
	}

	for i := 0; i < 20; i++ {
		ballot := make(markus.Ballot)
		ballot[rand.Uint64()%choicesCount] = 1
		ballot[rand.Uint64()%choicesCount] = 1
		ballot[rand.Uint64()%choicesCount] = 2
		ballot[rand.Uint64()%choicesCount] = 3
		ballot[rand.Uint64()%choicesCount] = 20
		ballot[rand.Uint64()%choicesCount] = 20
		if _, err := v.Vote(ballot); err != nil {
			t.Fatal(err)
		}
	}

	wantResults, wantTie, stale, err := v.ComputeSorted(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	assertEqual(t, "staled", stale, false)

	if runtime.GOOS != "windows" { // windows does not allow file to be removed
		// check results on file removal
		if err := os.Remove(v.StrengthsMatrixFilename()); err != nil {
			t.Fatal(err)
		}

		gotResults, gotTie, stale, err := v.ComputeSorted(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		assertEqual(t, "staled", stale, false)

		assertEqual(t, "results", gotResults, wantResults)
		assertEqual(t, "tie", gotTie, wantTie)
	}

	if err := v.Close(); err != nil {
		t.Fatal(err)
	}

	v2, err := markus.NewVoting(dir)
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() {
		if err := v2.Close(); err != nil {
			t.Error(err)
		}
	})

	gotResults, gotTie, stale, err := v2.ComputeSorted(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	assertEqual(t, "staled", stale, false)

	assertEqual(t, "results", gotResults, wantResults)
	assertEqual(t, "tie", gotTie, wantTie)
}

func TestVoting_changeChoices(t *testing.T) {
	seed := time.Now().UnixNano()
	t.Log("randomness seed", seed)
	random := rand.New(rand.NewSource(seed))

	type action struct {
		add    uint64
		remove []uint64
	}

	for _, tc := range []struct {
		name    string
		ballots []markus.Ballot
		initial uint64
		steps   []action
	}{
		{
			name:    "no votes, no choices",
			initial: 0,
			steps:   nil,
		},
		{
			name:    "initial votes, no change",
			initial: 5,
			steps:   nil,
		},
		{
			name: "single vote, single choice",
			ballots: []markus.Ballot{
				{0: 1},
			},
			initial: 1,
		},
		{
			name: "single vote, no change",
			ballots: []markus.Ballot{
				{1: 1},
			},
			initial: 5,
		},
		{
			name: "multiple votes, no change",
			ballots: []markus.Ballot{
				{1: 1},
				{0: 1, 2: 2, 3: 2},
				{1: 1, 3: 2, 4: 3},
			},
			initial: 5,
		},
		{
			name: "multiple votes, remove first choice",
			ballots: []markus.Ballot{
				{0: 1},
				{0: 1},
				{1: 1},
				{1: 1},
				{1: 1, 0: 2},
				{2: 1},
				{2: 1},
				{2: 1, 1: 2},
				{2: 2, 1: 2, 0: 3},
				{3: 1},
				{3: 1},
				{3: 1, 2: 2},
				{3: 2, 2: 2, 1: 3},
				{3: 1, 2: 3, 1: 3, 0: 4},
				{4: 1},
				{4: 1},
				{4: 2, 3: 2},
				{4: 1, 3: 2},
				{4: 2, 3: 2, 2: 3},
				{4: 1, 3: 2, 2: 3, 1: 3},
				{4: 2, 3: 2, 2: 3, 1: 4, 0: 5},
			},
			initial: 5,
			steps: []action{
				{remove: []uint64{0}},
			},
		},
		{
			name: "multiple votes, remove last choice",
			ballots: []markus.Ballot{
				{0: 1},
				{0: 1},
				{1: 1},
				{1: 1},
				{1: 1, 0: 2},
				{2: 1},
				{2: 1},
				{2: 1, 1: 2},
				{2: 2, 1: 2, 0: 3},
				{3: 1},
				{3: 1},
				{3: 1, 2: 2},
				{3: 2, 2: 2, 1: 3},
				{3: 1, 2: 3, 1: 3, 0: 4},
				{4: 1},
				{4: 1},
				{4: 2, 3: 2},
				{4: 1, 3: 2},
				{4: 2, 3: 2, 2: 3},
				{4: 1, 3: 2, 2: 3, 1: 3},
				{4: 2, 3: 2, 2: 3, 1: 4, 0: 5},
			},
			initial: 5,
			steps: []action{
				{remove: []uint64{4}},
			},
		},
		{
			name: "multiple votes, remove middle choice",
			ballots: []markus.Ballot{
				{0: 1},
				{0: 1},
				{1: 1},
				{1: 1},
				{1: 1, 0: 2},
				{2: 1},
				{2: 1},
				{2: 1, 1: 2},
				{2: 2, 1: 2, 0: 3},
				{3: 1},
				{3: 1},
				{3: 1, 2: 2},
				{3: 2, 2: 2, 1: 3},
				{3: 1, 2: 3, 1: 3, 0: 4},
				{4: 1},
				{4: 1},
				{4: 2, 3: 2},
				{4: 1, 3: 2},
				{4: 2, 3: 2, 2: 3},
				{4: 1, 3: 2, 2: 3, 1: 3},
				{4: 2, 3: 2, 2: 3, 1: 4, 0: 5},
			},
			initial: 5,
			steps: []action{
				{remove: []uint64{2}},
			},
		},
		{
			name: "multiple votes, remove multiple choices",
			ballots: []markus.Ballot{
				{0: 1},
				{0: 1},
				{1: 1},
				{1: 1},
				{1: 1, 0: 2},
				{2: 1},
				{2: 1},
				{2: 1, 1: 2},
				{2: 2, 1: 2, 0: 3},
				{3: 1},
				{3: 1},
				{3: 1, 2: 2},
				{3: 2, 2: 2, 1: 3},
				{3: 1, 2: 3, 1: 3, 0: 4},
				{4: 1},
				{4: 1},
				{4: 2, 3: 2},
				{4: 1, 3: 2},
				{4: 2, 3: 2, 2: 3},
				{4: 1, 3: 2, 2: 3, 1: 3},
				{4: 2, 3: 2, 2: 3, 1: 4, 0: 5},
			},
			initial: 5,
			steps: []action{
				{remove: []uint64{1, 2}},
			},
		},
		{
			name: "single vote add choice",
			ballots: []markus.Ballot{
				{0: 1},
			},
			initial: 2,
			steps: []action{
				{add: 1},
			},
		},
		{
			name: "multiple votes, add choice",
			ballots: []markus.Ballot{
				{0: 1},
				{0: 1},
				{0: 1},
				{0: 1},
				{0: 1},
				{0: 1},
				{0: 1},
				{0: 1},
				{0: 1},
				{0: 1},
				{1: 1, 0: 2},
				{1: 1},
				{1: 1, 0: 2},
				{1: 1, 0: 2},
				{1: 1, 0: 2},
				{1: 1, 0: 2},
				{1: 1, 0: 2},
				{2: 1},
				{2: 1},
				{2: 1},
				{2: 1, 1: 2},
				{2: 2, 1: 2, 0: 3},
				{3: 1},
				{3: 1},
				{3: 1, 2: 2},
				{3: 2, 2: 2, 1: 3},
				{3: 1, 2: 3, 1: 3, 0: 4},
				{4: 1},
				{4: 1},
				{4: 2, 3: 2},
				{4: 1, 3: 2},
				{4: 2, 3: 2, 2: 3},
				{4: 1, 3: 2, 2: 3, 1: 3},
				{4: 2, 3: 2, 2: 3, 1: 4, 0: 5},
				{0: 1, 1: 2, 2: 3, 3: 4, 4: 5},
				{0: 1, 1: 2, 2: 3, 3: 4},
				{5: 1},
			},
			initial: 6,
			steps: []action{
				{add: 1},
			},
		},
		{
			name: "multiple votes, new choices",
			ballots: []markus.Ballot{
				{0: 1},
				{0: 1},
				{0: 1},
				{0: 1},
				{0: 1},
				{0: 1},
				{0: 1},
				{0: 1},
				{0: 1},
				{0: 1},
				{1: 1, 0: 2},
				{1: 1},
				{1: 1, 0: 2},
				{1: 1, 0: 2},
				{1: 1, 0: 2},
				{1: 1, 0: 2},
				{1: 1, 0: 2},
				{2: 1},
				{2: 1},
				{2: 1},
				{2: 1, 1: 2},
				{2: 2, 1: 2, 0: 3},
				{3: 1},
				{3: 1},
				{3: 1, 2: 2},
				{3: 2, 2: 2, 1: 3},
				{3: 1, 2: 3, 1: 3, 0: 4},
				{4: 1},
				{4: 1},
				{4: 2, 3: 2},
				{4: 1, 3: 2},
				{4: 2, 3: 2, 2: 3},
				{4: 1, 3: 2, 2: 3, 1: 3},
				{4: 2, 3: 2, 2: 3, 1: 4, 0: 5},
				{0: 1, 1: 2, 2: 3, 3: 4, 4: 5},
				{0: 1, 1: 2, 2: 3, 3: 4},
				{5: 1},
			},
			initial: 6,
			steps: []action{
				{add: 3},
			},
		},
		{
			name: "multiple votes, add and remove choices",
			ballots: []markus.Ballot{
				{0: 1},
				{0: 1},
				{0: 1},
				{0: 1},
				{0: 1},
				{0: 1},
				{0: 1},
				{0: 1},
				{0: 1},
				{0: 1},
				{1: 1, 0: 2},
				{1: 1},
				{1: 1, 0: 2},
				{1: 1, 0: 2},
				{1: 1, 0: 2},
				{1: 1, 0: 2},
				{1: 1, 0: 2},
				{2: 1},
				{2: 1},
				{2: 1},
				{2: 1, 1: 2},
				{2: 2, 1: 2, 0: 3},
				{3: 1},
				{3: 1},
				{3: 1, 2: 2},
				{3: 2, 2: 2, 1: 3},
				{3: 1, 2: 3, 1: 3, 0: 4},
				{4: 1},
				{4: 1},
				{4: 2, 3: 2},
				{4: 1, 3: 2},
				{4: 2, 3: 2, 2: 3},
				{4: 1, 3: 2, 2: 3, 1: 3},
				{4: 2, 3: 2, 2: 3, 1: 4, 0: 5},
				{0: 1, 1: 2, 2: 3, 3: 4, 4: 5},
				{0: 1, 1: 2, 2: 3, 3: 4},
				{5: 1},
			},
			initial: 6,
			steps: []action{
				{add: 1},
				{remove: []uint64{0, 5}},
				{add: 2},
				{add: 1},
				{remove: []uint64{1, 7, 8}},
				{remove: []uint64{2}},
				{add: 3},
			},
		},
		{
			name:    "hundred random votes, new, remove and swap choices",
			ballots: randomBallots(t, random, 6, 100),
			initial: 6,
			steps: []action{
				{add: 1},
				{remove: []uint64{0, 5}},
				{add: 2},
				{add: 1},
				{remove: []uint64{1, 7, 8}},
				{remove: []uint64{2}},
				{add: 3},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			currentVoting := newMarkusVoting(t)
			_, _, err := currentVoting.AddChoices(tc.initial)
			assertEqual(t, "error", err, nil)

			for _, b := range tc.ballots {
				_, err := currentVoting.Vote(b)
				assertEqual(t, "error", err, nil)
			}

			for _, a := range tc.steps {
				if a.add > 0 {
					_, _, err := currentVoting.AddChoices(a.add)
					assertEqual(t, "error", err, nil)
				}
				if len(a.remove) > 0 {
					err := currentVoting.RemoveChoices(a.remove...)
					assertEqual(t, "error", err, nil)
				}
			}

			validationVoting := newMarkusVoting(t)
			_, _, err = validationVoting.AddChoices(tc.initial)
			assertEqual(t, "error", err, nil)

			for _, a := range tc.steps {
				if a.add > 0 {
					_, _, err := validationVoting.AddChoices(a.add)
					assertEqual(t, "error", err, nil)
				}
				if len(a.remove) > 0 {
					err := validationVoting.RemoveChoices(a.remove...)
					assertEqual(t, "error", err, nil)
				}
			}

			for _, b := range tc.ballots {
				b := removeMissingChoices(b, validationVoting.Choices())
				if _, err := validationVoting.Vote(b); err != nil {
					t.Fatal(err)
				}
			}

			assertPreferences(t, currentVoting, validationVoting)
		})
	}
}

func TestVoting_Unvote_correctness(t *testing.T) {
	v1 := newMarkusVoting(t)
	v2 := newMarkusVoting(t) // validation (does not receive unvote calls)

	_, _, err := v1.AddChoices(10)
	assertEqual(t, "error", err, nil)
	_, _, err = v2.AddChoices(10)
	assertEqual(t, "error", err, nil)

	var recordsToRemove []debugRecord

	seed := time.Now().UnixNano()
	t.Log("randomness seed", seed)
	random := rand.New(rand.NewSource(seed))

	for i := 0; i < 1000; i++ {
		t.Log("iteration", i)

		b := randomBallot(t, random, excludeChoices(v1.Choices(), 5))

		t.Log("vote ballot", i, b)

		r, err := v1.Vote(b)
		assertEqual(t, "error", err, nil)

		t.Log("vote record", i, r)

		if random.Intn(100) == 0 {
			recordsToRemove = append(recordsToRemove, debugRecord{
				Record: r,
				index:  i,
			})
		} else {
			_, err := v2.Vote(b)
			assertEqual(t, "error", err, nil)
		}

		if len(recordsToRemove) == 0 {
			assertPreferences(t, v1, v2)
		}

		if choices := randomRemoveChoices(t, random, excludeChoices(v1.Choices(), 5)); len(choices) != 0 {
			err := v1.RemoveChoices(choices...)
			assertEqual(t, "error", err, nil)
			err = v2.RemoveChoices(choices...)
			assertEqual(t, "error", err, nil)
			t.Log("remove choices", choices)
		}

		recordsToRemove = randomUnvote(t, random, v1, recordsToRemove)

		if len(recordsToRemove) == 0 {
			assertPreferences(t, v1, v2)
		}

		if count := randomAddChoices(t, random); count != nil {
			from, to, err := v1.AddChoices(*count)
			assertEqual(t, "error", err, nil)
			validationFrom, validationTo, err := v2.AddChoices(*count)
			assertEqual(t, "error", err, nil)
			t.Log("add choices", "count", *count, "from", from, "to", to, "validation from", validationFrom, "to", validationTo)
		}

		if len(recordsToRemove) == 0 {
			assertPreferences(t, v1, v2)
		}
	}
}

func BenchmarkVoting_ComputeSorted(b *testing.B) {
	b.Log("creating voting...")

	const choicesCount = 1000

	v := newMarkusVoting(b)

	b.Log("adding choices...")
	if _, _, err := v.AddChoices(choicesCount); err != nil {
		b.Fatal(err)
	}

	b.Log("voting...")
	for i := 0; i < 20; i++ {
		ballot := make(markus.Ballot)
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
		err := v.InvalidateStrengthMatrix()
		assertEqual(b, "invalidate strength matrix", err, nil)
	}
}

func BenchmarkVoting_Vote(b *testing.B) {
	b.Log("creating voting...")

	const choicesCount = 100

	v := newMarkusVoting(b)

	b.Log("adding choices...")
	if _, _, err := v.AddChoices(choicesCount); err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()

	for n := 0; n < b.N; n++ {
		if _, err := v.Vote(markus.Ballot{
			choicesCount / 2: 1,
		}); err != nil {
			b.Fatal(err)
		}
	}
}

func assertEqual[T any](t testing.TB, name string, got, want T) {
	t.Helper()

	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %s %#+v, want %+#v", name, got, want)
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

func newMarkusVoting(t testing.TB) *markus.Voting {
	t.Helper()

	dir := t.TempDir()

	v, err := markus.NewVoting(dir)
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

func fprintPreferences(w io.Writer, choices []string, preferences []uint32) (int, error) {
	var width int
	for _, c := range choices {
		l := len(fmt.Sprint(c))
		if l > width {
			width = l
		}
	}
	for _, p := range preferences {
		l := len(fmt.Sprint(p))
		if l > width {
			width = l
		}
	}
	format := fmt.Sprintf("%%%vv ", width)
	var count int
	write := func(v string) error {
		n, err := fmt.Fprint(w, v)
		if err != nil {
			return err
		}
		count += n
		return nil
	}

	if err := write(fmt.Sprintf(format, "")); err != nil {
		return count, err
	}
	for _, c := range choices {
		if err := write(fmt.Sprintf(format, c)); err != nil {
			return count, err
		}
	}
	if err := write("\n"); err != nil {
		return count, err
	}

	m := matrix(preferences)

	for i, col := range m {
		if err := write(fmt.Sprintf(format, choices[i])); err != nil {
			return count, err
		}
		for _, p := range col {
			if err := write(fmt.Sprintf(format, p)); err != nil {
				return count, err
			}
		}
		if err := write("\n"); err != nil {
			return count, err
		}
	}

	return count, nil
}

func sprintPreferences(choices []string, preferences []uint32) string {
	var buf bytes.Buffer
	_, _ = fprintPreferences(&buf, choices, preferences)
	return buf.String()
}

func matrix(preferences []uint32) [][]uint32 {
	l := len(preferences)
	choicesCount := floorSqrt(l)
	if choicesCount*choicesCount != l {
		return nil
	}

	matrix := make([][]uint32, 0, choicesCount)

	for i := 0; i < choicesCount; i++ {
		matrix = append(matrix, preferences[i*choicesCount:(i+1)*choicesCount])
	}
	return matrix
}

func floorSqrt(x int) int {
	if x == 0 || x == 1 {
		return x
	}
	start := 1
	end := x / 2
	ans := 0
	for start <= end {
		mid := (start + end) / 2
		if mid*mid == x {
			return mid
		}
		if mid*mid < x {
			start = mid + 1
			ans = mid
		} else {
			end = mid - 1
		}
	}
	return ans
}

func assertPreferences(t *testing.T, gotVoting, wantVoting *markus.Voting) {
	t.Helper()

	gotPreferences := sprintPreferences(gotVoting.PreferencesMatrixLabels(), gotVoting.PreferencesMatrix())
	wantPreferences := sprintPreferences(wantVoting.PreferencesMatrixLabels(), wantVoting.PreferencesMatrix())

	if gotPreferences != wantPreferences {
		t.Fatalf("\ngot preferences\n%v\nwant\n%v\n", gotPreferences, wantPreferences)
	} else {
		if verbose {
			t.Logf("\npreferences\n%v\nvalidation preferences\n%v\n", gotPreferences, wantPreferences)
		}
	}
}

func randomBallot(t testing.TB, random *rand.Rand, choices []uint64) markus.Ballot {
	t.Helper()

	choicesCount := uint64(len(choices))

	votedChoices := choicesCount
	if random.Intn(3) != 0 { // 30% ballots with unvoted choices
		votedChoices = random.Uint64() % choicesCount
	}

	b := make(markus.Ballot)
	for i := uint64(0); i < votedChoices; i++ {
		b[choices[random.Uint64()%choicesCount]] = random.Uint32() % uint32(choicesCount)
	}

	return b
}

func randomAddChoices(t testing.TB, random *rand.Rand) *uint64 {
	t.Helper()

	if random.Intn(100) != 0 {
		return nil
	}

	count := random.Uint64()%3 + 1
	return &count
}

func randomRemoveChoices(t testing.TB, random *rand.Rand, choices []uint64) []uint64 {
	t.Helper()

	if random.Intn(100) != 0 || len(choices) <= 2 {
		return nil
	}

	count := random.Int()%3 + 1
	toRemove := make([]uint64, 0, count)
	for i := 0; i < count; i++ {
		if len(choices) <= 2 {
			break
		}
		index := random.Intn(len(choices))
		var c uint64
		c, choices = choices[index], choices[index:index+1]
		toRemove = append(toRemove, c)
	}

	return toRemove
}

type debugRecord struct {
	markus.Record
	index int
}

func randomUnvote(t testing.TB, random *rand.Rand, v *markus.Voting, candidates []debugRecord) []debugRecord {
	if random.Intn(2) != 0 || len(candidates) == 0 {
		return candidates
	}

	count := len(candidates)
	if random.Intn(3) != 0 {
		count = random.Intn(len(candidates)) + 1
	}

	for i := 0; i < count; i++ {
		index := random.Intn(len(candidates))
		r := candidates[index]
		candidates = append(candidates[:index], candidates[index+1:]...)
		if verbose {
			t.Logf("\nbefore unvote\n%v\n", sprintPreferences(v.PreferencesMatrixLabels(), v.PreferencesMatrix()))
		}
		if err := v.Unvote(r.Record); err != nil {
			t.Fatal(err)
		}
		t.Log("unvote", r.index, r.Record)
	}

	return candidates
}

func excludeChoices(choices []uint64, exclude ...uint64) []uint64 {
	contains := func(c uint64) bool {
		for _, e := range exclude {
			if e == c {
				return true
			}
		}
		return false
	}
	r := make([]uint64, 0, len(choices)-len(exclude))
	for _, c := range choices {
		if contains(c) {
			continue
		}
		r = append(r, c)
	}
	return r
}

func removeMissingChoices(b markus.Ballot, choices []uint64) markus.Ballot {
	r := make(map[uint64]uint32)
	for c, v := range b {
		if !contains(choices, c) {
			continue
		}
		r[c] = v
	}
	return r
}

func contains(s []uint64, e uint64) bool {
	for _, x := range s {
		if x == e {
			return true
		}
	}
	return false
}

func randomBallots(t *testing.T, random *rand.Rand, choices uint64, count int) []markus.Ballot {
	t.Helper()

	ballots := make([]markus.Ballot, 0, count)

	for i := 0; i < count; i++ {
		b := make(markus.Ballot)
		for i := uint64(0); i < choices; i++ {
			b[random.Uint64()%choices] = random.Uint32() % uint32(choices)
		}
		ballots = append(ballots, b)
	}

	return ballots
}
