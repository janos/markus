// Copyright (c) 2023, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package markus

import (
	"context"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"sync"

	"resenje.org/markus/internal/bitset"
	"resenje.org/markus/internal/indexmap"
	"resenje.org/markus/internal/matrix"
)

// Voting holds number of votes for every pair of choices. Methods on the Voting
// type are safe for concurrent calls.
type Voting struct {
	path            string
	preferences     *matrix.Matrix
	strengths       *matrix.Matrix
	choicesIndex    *indexmap.Map
	preferencesSize uint64
	closed          bool

	mu          sync.RWMutex
	strengthsMu sync.Mutex
}

// NewVoting initializes a new voting with state stored in the provided
// directory.
func NewVoting(path string) (*Voting, error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		if err := os.MkdirAll(path, 0777); err != nil {
			return nil, fmt.Errorf("create directory: %w", err)
		}
	} else if err != nil {
		return nil, fmt.Errorf("stat directory: %w", err)
	}
	preferences, err := matrix.New(filepath.Join(path, "preferences.matrix"))
	if err != nil {
		return nil, fmt.Errorf("open preferences matrix: %w", err)
	}
	im, err := indexmap.New(filepath.Join(path, "choices.index"))
	if err != nil {
		return nil, fmt.Errorf("open choices index: %w", err)
	}
	return &Voting{
		path:            path,
		preferences:     preferences,
		preferencesSize: preferences.Size(),
		choicesIndex:    im,
	}, nil
}

// AddChoices adds new choices to the voting. It returns the range of the new
// indexes, with to value as inclusive.
func (v *Voting) AddChoices(count uint64) (from, to uint64, err error) {
	v.mu.Lock()
	defer v.mu.Unlock()

	preferencesResize := int64(0)
	from = math.MaxUint64
	for i := uint64(0); i < count; i++ {
		physical, virtual := v.choicesIndex.Add()
		if physical >= v.preferencesSize {
			preferencesResize++
		}
		if virtual < from {
			from = virtual
		}
		to = virtual
	}

	if preferencesResize > 0 {
		if _, _, err = v.preferences.Resize(preferencesResize); err != nil {
			return 0, 0, fmt.Errorf("resize preferences matrix for %v: %w", preferencesResize, err)
		}
	}

	// set the column of the new choice to the values of the
	// preferences' diagonal values, just as nobody voted for the
	// new choice to ensure consistency
	for j := from; j <= to; j++ {
		j, has := v.choicesIndex.GetPhysical(j)
		if !has {
			continue
		}
		for i := uint64(0); i <= to; i++ {
			i, has := v.choicesIndex.GetPhysical(i)
			if !has {
				continue
			}
			value := v.preferences.Get(i, i)
			v.preferences.Set(i, j, value)
		}
	}

	if err := v.choicesIndex.Write(); err != nil {
		return 0, 0, fmt.Errorf("write choices index: %w", err)
	}

	v.preferencesSize = v.preferences.Size()

	if err := v.preferences.Sync(); err != nil {
		return 0, 0, fmt.Errorf("sync preferences matrix: %w", err)
	}

	return from, to, nil
}

// RemoveChoices marks indexes as no longer available to vote for.
func (v *Voting) RemoveChoices(indexes ...uint64) error {
	v.mu.Lock()
	defer v.mu.Unlock()

	for _, i := range indexes {
		physical, has := v.choicesIndex.GetPhysical(i)
		if !has {
			continue
		}
		v.choicesIndex.Remove(i)
		for j := uint64(0); j < v.preferencesSize; j++ {
			v.preferences.Set(physical, j, 0)
			v.preferences.Set(j, physical, 0)
		}
	}

	if err := v.choicesIndex.Write(); err != nil {
		return fmt.Errorf("write choices index: %w", err)
	}

	return nil
}

// Ballot represents a single vote with ranked choices. Lowest number represents
// the highest rank. Not all choices have to be ranked and multiple choices can
// have the same rank. Ranks do not have to be in consecutive order.
type Ballot map[uint64]uint32

// Record represents a single vote with ranked choices. It is a list of Ballot
// values. The first ballot is the list with the first choices, the second
// ballot is the list with the second choices, and so on. The last ballot is the
// list of choices that are not ranked, which can be an empty list.
type Record [][]uint64

// Vote adds a voting preferences by a single voting ballot. A record of a
// complete and normalized preferences is returned that can be used to unvote.
func (v *Voting) Vote(b Ballot) (Record, error) {
	v.mu.Lock()
	defer v.mu.Unlock()

	choicesCursor := v.choicesIndex.Cursor()

	ballotLen := uint64(len(b))

	ballotRanks := make(map[uint32][]uint64, ballotLen)

	for index, rank := range b {
		if index >= choicesCursor {
			return Record{}, &UnknownChoiceError{Index: index}
		}
		if _, has := v.choicesIndex.GetPhysical(index); !has {
			return Record{}, &UnknownChoiceError{Index: index}
		}

		ballotRanks[rank] = append(ballotRanks[rank], index)
	}

	rankNumbers := make([]uint32, 0, len(ballotRanks))
	for rank := range ballotRanks {
		rankNumbers = append(rankNumbers, rank)
	}

	sort.Slice(rankNumbers, func(i, j int) bool {
		return rankNumbers[i] < rankNumbers[j]
	})

	hasUnrankedChoices := uint64(len(b)) != choicesCursor

	ranksLen := len(rankNumbers)
	if hasUnrankedChoices {
		ranksLen++
	}

	// all choices are ranked, tread diagonal values as a single not ranked
	// choice, deprioritizing them for all existing choices
	for i := range b {
		i, has := v.choicesIndex.GetPhysical(i)
		if !has {
			continue
		}
		v.preferences.Inc(i, i)
	}

	ranks := make([][]uint64, 0, ranksLen)
	for _, rankNumber := range rankNumbers {
		ranks = append(ranks, ballotRanks[rankNumber])
	}

	if hasUnrankedChoices {
		ranks = append(ranks, unrankedChoices(choicesCursor, ranks))
	} else {
		ranks = append(ranks, make([]uint64, 0))
	}

	for rank, choices1 := range ranks {
		rest := ranks[rank+1:]
		for _, index1 := range choices1 {
			index1, has := v.choicesIndex.GetPhysical(index1)
			if !has {
				continue
			}
			for _, choices2 := range rest {
				for _, index2 := range choices2 {
					index2, has := v.choicesIndex.GetPhysical(index2)
					if !has {
						continue
					}
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

	return ranks, nil
}

// Unvote removes the vote from the preferences.
func (v *Voting) Unvote(r Record) error {
	v.mu.Lock()
	defer v.mu.Unlock()

	recordLength := len(r)
	if recordLength == 0 {
		return nil
	}

	var max uint64
	for rank, choices1 := range r {
		rest := r[rank+1:]
		for _, index1 := range choices1 {
			if index1 > max {
				max = index1
			}
			index1, has := v.choicesIndex.GetPhysical(index1)
			if !has {
				continue
			}
			for _, choices2 := range rest {
				for _, index2 := range choices2 {
					index2, has := v.choicesIndex.GetPhysical(index2)
					if !has {
						continue
					}
					v.preferences.Dec(index1, index2)
				}
			}
		}
	}

	rankedChoices := make(map[uint64]struct{}) // physical

	// remove voting from the ranked choices of the Record
	for _, choices := range r[:recordLength-1] {
		for _, index := range choices {
			index, has := v.choicesIndex.GetPhysical(index)
			if !has {
				continue
			}
			v.preferences.Dec(index, index)
			rankedChoices[index] = struct{}{} // physical
		}
	}

	// remove votes of the choices that were added after the Record
	for i := uint64(0); i < v.preferencesSize; i++ {
		if _, ok := rankedChoices[i]; ok {
			for j := uint64(0); j < v.preferencesSize; j++ {
				virtual, has := v.choicesIndex.GetVirtual(j)
				if !has {
					continue
				}
				if virtual > max {
					v.preferences.Dec(i, j)
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
	var count uint64
	for _, rank := range ranks {
		for _, index := range rank {
			ranked.Set(index)
			count++
		}
	}

	unranked = make([]uint64, 0, choicesLen-count)

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
	Wins uint64
	// Total number of votes in the weakest link of the strongest path in wins
	// in pairwise comparisons to other choices votings. Strength does not
	// effect the winner, and may be less then the Strength of the choice with
	// more wins.
	Strength uint64
	// Total number of preferred votes (difference between votes of the winner
	// choice and the opponent choice) in the weakest link of the strongest path
	// in wins in pairwise comparisons to other choices votings. Advantage does
	// not effect the winner, and may be less then the Advantage of the choice
	// with more wins. The code with less wins and greater Advantage had
	// stronger but fewer wins and that information can be taken into the
	// analysis of the results.
	Advantage uint64
}

// Size returns the number of choices in the voting.
func (v *Voting) Size() uint64 {
	v.mu.RLock()
	defer v.mu.RUnlock()

	return v.preferencesSize - uint64(v.choicesIndex.FreeCount())
}

// Compute calculates the results of the voting. The function passed as the
// second argument is called for each choice with the Result value. If the function
// returns false, the iteration is stopped. The order of the results is not
// sorted by the number of wins.
func (v *Voting) Compute(ctx context.Context, f func(Result) (bool, error)) (stale bool, err error) {
	v.mu.RLock()
	defer v.mu.RUnlock()

	return v.compute(ctx, f)
}

// ComputeSorted calculates a sorted list of choices with the total number of wins for
// each of them. If there are multiple winners, tie boolean parameter is true.
func (v *Voting) ComputeSorted(ctx context.Context) (results []Result, tie, stale bool, err error) {
	v.mu.RLock()
	defer v.mu.RUnlock()

	results = make([]Result, 0, v.preferencesSize)

	stale, err = v.compute(ctx, func(r Result) (bool, error) {
		results = append(results, r)
		return true, nil
	})
	if err != nil {
		return nil, false, false, fmt.Errorf("compute: %w", err)
	}

	sort.Slice(results, func(i, j int) bool {
		if results[i].Wins != results[j].Wins {
			return results[i].Wins > results[j].Wins
		}
		if results[i].Strength != results[j].Strength {
			return results[i].Strength > results[j].Strength
		}
		return results[i].Index < results[j].Index
	})

	if len(results) >= 2 {
		tie = results[0].Wins == results[1].Wins
	}

	return results, tie, stale, nil
}

func (v *Voting) compute(ctx context.Context, f func(Result) (bool, error)) (stale bool, err error) {
	preferencesSize := v.preferencesSize
	if preferencesSize == 0 {
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

	if err := v.prepareStrengthsMatrix(); err != nil {
		return false, fmt.Errorf("prepare strengths matrix: %w", err)
	}

	if v.strengths.Version() != preferencesVersion {
		// Invalidate the strengths matrix version to ensure that it will be
		// recalculated even if the iteration is interrupted by a context.
		if err := v.strengths.SetVersion(0); err != nil {
			return false, fmt.Errorf("invalidate strengths matrix version: %w", err)
		}

		for i := uint64(0); i < preferencesSize; i++ {
			select {
			case <-ctx.Done():
				return false, ctx.Err()
			default:
			}

			for j := uint64(0); j < preferencesSize; j++ {

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

		for i := uint64(0); i < preferencesSize; i++ {
			_, has := v.choicesIndex.GetPhysical(i)
			if !has {
				continue
			}

			for j := uint64(0); j < preferencesSize; j++ {
				// check is not needed, avoid the condition call for performance
				// has := v.choicesIndex.Has(j)
				// if !has {
				// 	continue
				// }

				if i == j {
					continue
				}
				select {
				case <-ctx.Done():
					return false, ctx.Err()
				default:
				}
				ji := v.strengths.Get(j, i)
				for k := uint64(0); k < preferencesSize; k++ {
					// check is not needed, avoid the condition call for performance
					// has := v.choicesIndex.Has(k)
					// if !has {
					// 	continue
					// }

					// check is not needed, avoid the condition call for performance
					// if i == k || j == k {
					// 	continue
					// }
					jk := v.strengths.Get(j, k)
					m := min(
						ji,
						v.strengths.Get(i, k),
					)
					if m > jk {
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
	}

	choicesCursor := v.choicesIndex.Cursor()
	for i := uint64(0); i < choicesCursor; i++ {
		select {
		case <-ctx.Done():
			return false, ctx.Err()
		default:
		}

		i, has := v.choicesIndex.GetPhysical(i)
		if !has {
			continue
		}

		var wins uint64
		var strength uint64
		var advantage uint64

		for j := uint64(0); j < choicesCursor; j++ {
			j, has := v.choicesIndex.GetPhysical(j)
			if !has {
				continue
			}

			if i != j {
				ijVotes := v.strengths.Get(i, j)
				jiVotes := v.strengths.Get(j, i)
				if ijVotes > jiVotes {
					wins++
					strength += uint64(ijVotes)
					advantage += uint64(ijVotes) - uint64(jiVotes)
				}
			}
		}
		cont, err := f(Result{Index: i, Wins: wins, Strength: strength, Advantage: advantage})
		if err != nil {
			return false, fmt.Errorf("calculate results: %w", err)
		}
		if !cont {
			return false, nil
		}
	}

	return false, nil
}

// Close closes the voting. It is not safe to use the voting after it is closed.
func (v *Voting) Close() error {
	v.mu.Lock()
	defer v.mu.Unlock()

	if v.closed {
		return nil
	}
	v.closed = true

	if err := v.preferences.Close(); err != nil {
		return fmt.Errorf("close preferences matrix: %w", err)
	}

	if v.strengths != nil {
		if err := v.strengths.Close(); err != nil {
			return fmt.Errorf("close strengths matrix: %w", err)
		}
	}

	return nil
}

func (v *Voting) prepareStrengthsMatrix() error {
	if v.strengths == nil || v.strengths.Sync() != nil {
		strengths, err := matrix.New(v.strengthsMatrixFilename())
		if err != nil {
			return fmt.Errorf("open: %w", err)
		}
		v.strengths = strengths
	}

	if diff := int64(v.preferences.Size()) - int64(v.strengths.Size()); diff > 0 {
		if _, _, err := v.strengths.Resize(diff); err != nil {
			return fmt.Errorf("resize for %d: %w", diff, err)
		}
	}

	return nil
}

func (v *Voting) strengthsMatrixFilename() string {
	return filepath.Join(v.path, "strengths.matrix")
}

func min(a, b uint32) uint32 {
	if a < b {
		return a
	}
	return b
}
