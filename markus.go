// Copyright (c) 2023, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package markus

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"

	"resenje.org/markus/internal/bitset"
	"resenje.org/markus/internal/list"
	"resenje.org/markus/internal/matrix"
)

type VotingCounter interface {
	~uint8 | ~uint16 | ~uint32 | ~uint64
}

type Voting[C comparable, VC VotingCounter] struct {
	preferences *matrix.Matrix[VC]
	strengths   *matrix.Matrix[VC]
	strengthsMu sync.RWMutex
	choices     *list.List[C]
	mu          sync.RWMutex
	closed      bool
}

func New[C comparable, VC VotingCounter](path string) (*Voting[C, VC], error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		if err := os.MkdirAll(path, 0777); err != nil {
			return nil, fmt.Errorf("create directory: %w", err)
		}
	} else if err != nil {
		return nil, fmt.Errorf("stat directory: %w", err)
	}
	preferences, err := matrix.New[VC](filepath.Join(path, "preferences.matrix"))
	if err != nil {
		return nil, fmt.Errorf("open preferences matrix: %w", err)
	}
	strengths, err := matrix.New[VC](filepath.Join(path, "strengths.matrix"))
	if err != nil {
		return nil, fmt.Errorf("open strengths matrix: %w", err)
	}
	if ps, ss := preferences.Size(), strengths.Size(); ps != ss {
		return nil, fmt.Errorf("preferences matrix dimension %v and strengths matrix dimension %v do not match", ps, ss)
	}
	choices, err := list.New[C](filepath.Join(path, "choices.list"))
	if err != nil {
		return nil, fmt.Errorf("open choices list: %w", err)
	}
	if cs, ps := int64(choices.Size()), preferences.Size(); cs != ps {
		return nil, fmt.Errorf("choices dimension %v and preferences matrix dimension %v do not match", cs, ps)
	}
	return &Voting[C, VC]{
		preferences: preferences,
		strengths:   strengths,
		choices:     choices,
	}, nil
}

func (v *Voting[C, VC]) Add(choices ...C) error {
	v.mu.Lock()
	v.strengthsMu.Lock()
	defer v.mu.Unlock()
	defer v.strengthsMu.Unlock()

	diff, err := v.choices.Append(choices...)
	if err != nil {
		return fmt.Errorf("append choices: %w", err)
	}
	if diff == 0 {
		return nil
	}

	if _, _, err = v.preferences.Resize(int64(diff)); err != nil {
		return fmt.Errorf("resize matrix for %d: %w", diff, err)
	}

	if _, _, err = v.strengths.Resize(int64(diff)); err != nil {
		return fmt.Errorf("resize matrix for %d: %w", diff, err)
	}

	return nil
}

// Ballot represents a single vote with ranked choices. Lowest number represents
// the highest rank. Not all choices have to be ranked and multiple choices can
// have the same rank. Ranks do not have to be in consecutive order.
type Ballot[C comparable, VC VotingCounter] map[C]VC

// Vote updates the preferences passed as the first argument with the Ballot
// values.
func (v *Voting[C, VC]) Vote(b Ballot[C, VC]) error {
	return v.vote(b, func(e VC) VC {
		return e + 1
	})
}

// Unvote removes the Ballot values from the preferences.
func (v *Voting[C, VC]) Unvote(b Ballot[C, VC]) error {
	return v.vote(b, func(e VC) VC {
		return e - 1
	})
}

func (v *Voting[C, VC]) vote(b Ballot[C, VC], change func(VC) VC) error {
	v.mu.Lock()
	defer v.mu.Unlock()

	ranks, err := v.ballotRanks(b)
	if err != nil {
		return fmt.Errorf("ballot ranks: %w", err)
	}

	var sum int64
	for rank, choices1 := range ranks {
		rest := ranks[rank+1:]
		for _, i := range choices1 {
			sum += i
			for _, choices1 := range rest {
				for _, j := range choices1 {
					v.preferences.Set(i, j, change(v.preferences.Get(i, j)))
				}
			}
		}
	}

	if err := v.preferences.SetVersion(v.preferences.Version() + sum); err != nil {
		return fmt.Errorf("set preferences version: %w", err)
	}

	if err := v.preferences.Sync(); err != nil {
		return fmt.Errorf("sync preferences matrix: %w", err)
	}

	return nil
}

func (v *Voting[C, VC]) ballotRanks(b Ballot[C, VC]) (ranks [][]int64, err error) {
	choicesLen := int64(v.choices.Size())
	ballotLen := int64(len(b))
	hasUnrankedChoices := ballotLen != choicesLen

	ballotRanks := make(map[VC][]int64, ballotLen)
	var rankedChoices bitset.BitSet[int64]
	if hasUnrankedChoices {
		rankedChoices = bitset.New(choicesLen)
	}

	for choice, rank := range b {
		index, ok := v.choices.Index(choice)
		if !ok {
			return nil, &UnknownChoiceError[C]{Choice: choice}
		}

		ballotRanks[rank] = append(ballotRanks[rank], int64(index))

		if hasUnrankedChoices {
			rankedChoices.Set(index)
		}
	}

	rankNumbers := make([]VC, 0, len(ballotRanks))
	for rank := range ballotRanks {
		rankNumbers = append(rankNumbers, rank)
	}

	sort.Slice(rankNumbers, func(i, j int) bool {
		return rankNumbers[i] < rankNumbers[j]
	})

	ranks = make([][]int64, 0, len(rankNumbers))
	for _, rankNumber := range rankNumbers {
		ranks = append(ranks, ballotRanks[rankNumber])
	}

	if hasUnrankedChoices {
		unranked := make([]int64, 0, choicesLen-ballotLen)
		for i := int64(0); i < choicesLen; i++ {
			if !rankedChoices.IsSet(i) {
				unranked = append(unranked, i)
			}
		}
		if len(unranked) > 0 {
			ranks = append(ranks, unranked)
		}
	}

	return ranks, nil
}

// Result represents a total number of wins for a single choice.
type Result[C comparable] struct {
	// The choice value.
	Choice C
	// 0-based ordinal number of the choice in the choice slice.
	Index int64
	// Number of wins in pairwise comparisons to other choices votings.
	Wins int
}

// Compute calculates a sorted list of choices with the total number of wins for
// each of them. If there are multiple winners, tie boolean parameter is true.
func (v *Voting[C, VC]) Compute() (results []Result[C], tie, stale bool, err error) {
	stale, err = v.calculatePairwiseStrengths()
	if err != nil {
		return nil, false, false, fmt.Errorf("calculate pairwise strengths: %w", err)
	}
	results, tie, err = v.calculateResults()
	if err != nil {
		return nil, false, false, fmt.Errorf("calculate results: %w", err)
	}
	return results, tie, stale, nil
}

func (v *Voting[C, VC]) calculatePairwiseStrengths() (bool, error) {
	v.mu.RLock()
	defer v.mu.RUnlock()

	choicesCount := int64(v.choices.Size())
	if choicesCount == 0 {
		return false, nil
	}

	if !v.strengthsMu.TryLock() {
		// If the lock is already taken, it means that the strengths matrix is
		// being calculated by another goroutine. In this case, we can just
		// return nil and wait for the other goroutine to finish.
		return true, nil
	}
	defer v.strengthsMu.Unlock()

	preferencesVersion := v.preferences.Version()
	if v.strengths.Version() == preferencesVersion {
		return false, nil
	}

	for i := int64(0); i < choicesCount; i++ {
		for j := int64(0); j < choicesCount; j++ {
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

	for i := int64(0); i < choicesCount; i++ {
		for j := int64(0); j < choicesCount; j++ {
			if i == j {
				continue
			}

			for k := int64(0); k < choicesCount; k++ {
				if i == k || j == k {
					continue
				}
				m := max(
					v.strengths.Get(j, k),
					min(
						v.strengths.Get(j, i),
						v.strengths.Get(i, k),
					),
				)
				v.strengths.Set(j, k, m)
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

func (v *Voting[C, VC]) calculateResults() (results []Result[C], tie bool, err error) {
	v.strengthsMu.RLock()
	defer v.strengthsMu.RUnlock()

	choicesCount := int64(v.choices.Size())
	results = make([]Result[C], 0, choicesCount)

	for i := int64(0); i < choicesCount; i++ {
		var count int

		for j := int64(0); j < choicesCount; j++ {
			if i != j {
				if v.strengths.Get(i, j) > v.strengths.Get(j, i) {
					count++
				}
			}
		}
		results = append(results, Result[C]{Choice: v.choices.Elements()[i], Index: i, Wins: count})
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

	return results, tie, nil
}

func (v *Voting[C, VC]) Close() error {
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

func min[VC VotingCounter](a, b VC) VC {
	if a < b {
		return a
	}
	return b
}

func max[VC VotingCounter](a, b VC) VC {
	if a > b {
		return a
	}
	return b
}
