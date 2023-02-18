// Copyright (c) 2023, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package markus_test

import (
	"context"
	"fmt"
	"log"
	"os"

	"resenje.org/markus"
)

func ExampleVoting() {
	dir, err := os.MkdirTemp("", "")
	if err != nil {
		log.Fatal(err)
	}
	defer os.RemoveAll(dir)

	// Create a new voting.
	v, err := markus.NewVoting[uint64](dir)
	if err != nil {
		log.Fatal(err)
	}
	defer v.Close()

	// Add choices.
	if _, _, err := v.AddChoices(3); err != nil {
		log.Fatal(err)
	}

	// First vote.
	record1, err := v.Vote(markus.Ballot[uint64]{
		0: 1,
		2: 2,
	})
	if err != nil {
		log.Fatal(err)
	}

	// Second vote.
	if _, err := v.Vote(markus.Ballot[uint64]{
		1: 1,
		2: 2,
	}); err != nil {
		log.Fatal(err)
	}

	// A vote can be changed.
	if err := v.Unvote(record1); err != nil {
		log.Fatal(err)
	}
	if _, err := v.Vote(markus.Ballot[uint64]{
		0: 1,
		1: 2,
	}); err != nil {
		log.Fatal(err)
	}

	// Calculate the result.
	result, _, tie, err := v.ComputeSorted(context.Background())
	if err != nil {
		log.Fatal(err)
	}
	if tie {
		log.Fatal("tie")
	}
	fmt.Println("winner:", result[0].Index)

	// Output: winner: 1
}
