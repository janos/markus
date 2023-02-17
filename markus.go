// Copyright (c) 2023, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package markus

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"

	"resenje.org/markus/internal/bitset"
	"resenje.org/markus/internal/matrix"
)

type Type interface {
	~uint8 | ~uint16 | ~uint32 | ~uint64
}

type Voting[T Type] struct {
	preferences  *matrix.Matrix[T]
	strengths    *matrix.Matrix[T]
	choicesCount uint64
	closed       bool

	mu          sync.RWMutex
	strengthsMu sync.Mutex
}

func New[T Type](path string) (*Voting[T], error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		if err := os.MkdirAll(path, 0777); err != nil {
			return nil, fmt.Errorf("create directory: %w", err)
		}
	} else if err != nil {
		return nil, fmt.Errorf("stat directory: %w", err)
	}
	preferences, err := matrix.New[T](filepath.Join(path, "preferences.matrix"))
	if err != nil {
		return nil, fmt.Errorf("open preferences matrix: %w", err)
	}
	strengths, err := matrix.New[T](filepath.Join(path, "strengths.matrix"))
	if err != nil {
		return nil, fmt.Errorf("open strengths matrix: %w", err)
	}
	size := preferences.Size()
	if s := strengths.Size(); size != s {
		return nil, fmt.Errorf("preferences matrix dimension %v and strengths matrix dimension %v do not match", size, s)
	}
	return &Voting[T]{
		preferences:  preferences,
		strengths:    strengths,
		choicesCount: uint64(size),
	}, nil
}

func (v *Voting[T]) Add(count uint64) (from, to uint64, err error) {
	v.mu.Lock()
	defer v.mu.Unlock()

	from, to, err = v.preferences.Resize(int64(count))
	if err != nil {
		return 0, 0, fmt.Errorf("resize matrix for %d: %w", count, err)
	}

	strengthsFrom, strengthsTo, err := v.strengths.Resize(int64(count))
	if err != nil {
		return 0, 0, fmt.Errorf("resize matrix for %d: %w", count, err)
	}

	if from != strengthsFrom || to != strengthsTo {
		return 0, 0, fmt.Errorf("preferences matrix dimension %v and strengths matrix dimension %v do not match", to-from, strengthsTo-strengthsFrom)
	}

	v.choicesCount = v.preferences.Size()

	return from, to, nil
}

// Ballot represents a single vote with ranked choices. Lowest number represents
// the highest rank. Not all choices have to be ranked and multiple choices can
// have the same rank. Ranks do not have to be in consecutive order.
type Ballot[T Type] map[uint64]T

type Record struct {
	Ranks [][]uint64
	Size  uint64
}

// Vote updates the preferences passed as the first argument with the Ballot
// values.
func (v *Voting[T]) Vote(b Ballot[T]) (Record, error) {
	v.mu.Lock()
	defer v.mu.Unlock()

	choicesLen := v.choicesCount

	ballotLen := uint64(len(b))

	ballotRanks := make(map[T][]uint64, ballotLen)

	for index, rank := range b {
		if index >= choicesLen {
			return Record{}, &UnknownChoiceError{Index: index}
		}

		ballotRanks[rank] = append(ballotRanks[rank], index)
	}

	rankNumbers := make([]T, 0, len(ballotRanks))
	for rank := range ballotRanks {
		rankNumbers = append(rankNumbers, rank)
	}

	sort.Slice(rankNumbers, func(i, j int) bool {
		return rankNumbers[i] < rankNumbers[j]
	})

	hasUnrankedChoices := uint64(len(b)) != choicesLen

	ranksLen := len(rankNumbers)
	if hasUnrankedChoices {
		ranksLen++
	}

	ranks := make([][]uint64, 0, ranksLen)
	for _, rankNumber := range rankNumbers {
		ranks = append(ranks, ballotRanks[rankNumber])
	}

	if hasUnrankedChoices {
		ranks = append(ranks, unrankedChoices(choicesLen, ranks))
	}

	for rank, choices1 := range ranks {
		rest := ranks[rank+1:]
		for _, index1 := range choices1 {
			for _, choices2 := range rest {
				for _, index2 := range choices2 {
					v.preferences.Inc(index1, index2)
				}
			}
		}
	}

	if err := v.preferences.SetVersion(v.preferences.Version() + 1); err != nil {
		return Record{}, fmt.Errorf("set preferences version: %w", err)
	}

	if err := v.preferences.Sync(); err != nil {
		return Record{}, fmt.Errorf("sync preferences matrix: %w", err)
	}

	return Record{
		Ranks: ranks,
		Size:  v.choicesCount,
	}, nil
}

// Unvote removes the Ballot values from the preferences.
func (v *Voting[T]) Unvote(r Record) error {
	v.mu.Lock()
	defer v.mu.Unlock()

	ranks := r.Ranks

	uc := unrankedChoices(r.Size, ranks)
	if len(uc) > 0 {
		ranks = append(ranks, uc)
	}

	for rank, choices1 := range ranks {
		rest := ranks[rank+1:]
		for _, index1 := range choices1 {
			for _, choices2 := range rest {
				for _, index2 := range choices2 {
					v.preferences.Dec(index1, index2)
				}
			}
		}
	}

	if err := v.preferences.SetVersion(v.preferences.Version() + 1); err != nil {
		return fmt.Errorf("set preferences version: %w", err)
	}

	if err := v.preferences.Sync(); err != nil {
		return fmt.Errorf("sync preferences matrix: %w", err)
	}

	return nil
}

func unrankedChoices(choicesLen uint64, ranks [][]uint64) (unranked []uint64) {
	ranked := bitset.New(choicesLen)
	for _, rank := range ranks {
		for _, index := range rank {
			ranked.Set(index)
		}
	}

	ranked.IterateUnset(func(i uint64) {
		unranked = append(unranked, i)
	})

	return unranked
}

// Result represents a total number of wins for a single choice.
type Result struct {
	// 0-based ordinal number of the choice in the choice slice.
	Index uint64
	// Number of wins in pairwise comparisons to other choices votings.
	Wins int
}

func (v *Voting[T]) Size() uint64 {
	v.mu.RLock()
	defer v.mu.RUnlock()

	return v.choicesCount
}

// Compute calculates the results of the voting. The function passed as the
// second argument is called for each choice with the Result value. If the function
// returns true, the iteration is stopped. The order of the results is not
// sorted by the number of wins.
func (v *Voting[T]) Compute(ctx context.Context, f func(Result) (bool, error)) (stale bool, err error) {
	v.mu.RLock()
	defer v.mu.RUnlock()

	return v.compute(ctx, f)
}

// ComputeSorted calculates a sorted list of choices with the total number of wins for
// each of them. If there are multiple winners, tie boolean parameter is true.
func (v *Voting[T]) ComputeSorted(ctx context.Context) (results []Result, tie, stale bool, err error) {
	v.mu.RLock()
	defer v.mu.RUnlock()

	results = make([]Result, 0, v.choicesCount)

	stale, err = v.compute(ctx, func(r Result) (bool, error) {
		results = append(results, r)
		return false, nil
	})
	if err != nil {
		return nil, false, false, fmt.Errorf("calculate results: %w", err)
	}

	sort.Slice(results, func(i, j int) bool {
		if results[i].Wins == results[j].Wins {
			return results[i].Index < results[j].Index
		}
		return results[i].Wins > results[j].Wins
	})

	if len(results) >= 2 {
		tie = results[0].Wins == results[1].Wins
	}

	return results, tie, stale, nil
}

func (v *Voting[T]) compute(ctx context.Context, f func(Result) (bool, error)) (stale bool, err error) {
	stale, err = v.calculatePairwiseStrengths(ctx)
	if err != nil {
		return false, fmt.Errorf("calculate pairwise strengths: %w", err)
	}
	if err = v.calculateResults(ctx, f); err != nil {
		return false, fmt.Errorf("calculate results: %w", err)
	}
	return stale, nil
}

func (v *Voting[T]) calculatePairwiseStrengths(ctx context.Context) (stale bool, err error) {
	choicesCount := v.choicesCount
	if choicesCount == 0 {
		return false, nil
	}

	preferencesVersion := v.preferences.Version()
	if v.strengths.Version() == preferencesVersion {
		return false, nil
	}

	if !v.strengthsMu.TryLock() {
		// If the lock is already taken, it means that the strengths matrix is
		// being calculated by another goroutine. In this case, we can just
		// return nil and wait for the other goroutine to finish.
		return true, nil
	}
	defer v.strengthsMu.Unlock()

	// Invalidate the strengths matrix version to ensure that it will be
	// recalculated even if the iteration is interrupted by a context.
	if err := v.strengths.SetVersion(0); err != nil {
		return false, fmt.Errorf("invalidate strengths matrix version: %w", err)
	}

	for i := uint64(0); i < choicesCount; i++ {
		select {
		case <-ctx.Done():
			return false, ctx.Err()
		default:
		}
		for j := uint64(0); j < choicesCount; j++ {
			if i == j {
				continue
			}
			if c := v.preferences.Get(i, j); c > v.preferences.Get(j, i) {
				v.strengths.Set(i, j, c)
			} else {
				v.strengths.Set(i, j, 0)
			}
		}
	}

	for i := uint64(0); i < choicesCount; i++ {
		for j := uint64(0); j < choicesCount; j++ {
			if i == j {
				continue
			}
			select {
			case <-ctx.Done():
				return false, ctx.Err()
			default:
			}
			ji := v.strengths.Get(j, i)
			for k := uint64(0); k < choicesCount; k++ {
				if i == k || j == k {
					continue
				}
				jk := v.strengths.Get(j, k)
				m := max(
					jk,
					min(
						ji,
						v.strengths.Get(i, k),
					),
				)
				if m != jk {
					v.strengths.Set(j, k, m)
				}
			}
		}
	}

	if err := v.strengths.SetVersion(preferencesVersion); err != nil {
		return false, fmt.Errorf("set strengths matrix version: %w", err)
	}

	if err := v.strengths.Sync(); err != nil {
		return false, fmt.Errorf("sync strengths matrix: %w", err)
	}

	return false, nil
}

func (v *Voting[T]) calculateResults(ctx context.Context, f func(Result) (bool, error)) error {
	choicesCount := v.choicesCount

	for i := uint64(0); i < choicesCount; i++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		var count int

		for j := uint64(0); j < choicesCount; j++ {
			if i != j {
				if v.strengths.Get(i, j) > v.strengths.Get(j, i) {
					count++
				}
			}
		}
		stop, err := f(Result{Index: i, Wins: count})
		if err != nil {
			return fmt.Errorf("calculate results: %w", err)
		}
		if stop {
			return nil
		}
	}

	return nil
}

func (v *Voting[T]) Close() error {
	v.mu.Lock()
	defer v.mu.Unlock()

	if v.closed {
		return nil
	}
	v.closed = true

	if err := v.preferences.Close(); err != nil {
		return fmt.Errorf("close preferences matrix: %w", err)
	}

	if err := v.strengths.Close(); err != nil {
		return fmt.Errorf("close strengths matrix: %w", err)
	}

	return nil
}

func min[T Type](a, b T) T {
	if a < b {
		return a
	}
	return b
}

func max[T Type](a, b T) T {
	if a > b {
		return a
	}
	return b
}
