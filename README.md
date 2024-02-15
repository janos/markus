# Schulze method Go library for large number of choices

[![Go](https://github.com/janos/markus/workflows/Go/badge.svg)](https://github.com/janos/markus/actions)
[![PkgGoDev](https://pkg.go.dev/badge/resenje.org/markus)](https://pkg.go.dev/resenje.org/markus)
[![NewReleases](https://newreleases.io/badge.svg)](https://newreleases.io/github/janos/markus)

Schulze is a Go implementation of the [Schulze method](https://en.wikipedia.org/wiki/Schulze_method) voting system. The system is developed in 1997 by Markus Schulze. It is a single winner preferential voting. The Schulze method is also known as Schwartz Sequential dropping (SSD), cloneproof Schwartz sequential dropping (CSSD), the beatpath method, beatpath winner, path voting, and path winner.

The Schulze method is a [Condorcet method](https://en.wikipedia.org/wiki/Condorcet_method), which means that if there is a candidate who is preferred by a majority over every other candidate in pairwise comparisons, then this candidate will be the winner when the Schulze method is applied.

White paper [Markus Schulze, "The Schulze Method of Voting"](https://arxiv.org/pdf/1804.02973.pdf).

This library is intended to be used for votings with large number of choices, usually more than 1000. For votings that need more control on the voting choices and have less than 1000 choices, [resenje.org/schulze](https://pkg.go.dev/resenje.org/schulze) can be used.

## Usage

A new voting can be created with `NewVoting` constructor. Voting is persisted on the disk to minimize memory usage with large number of choices.

Choices can only be added with `AddChoices` method, which is expanding choice indexes. Choices cannot be removed or reordered. It is up to the user to ignore or reorder choices in the upper layers of application.

Voting is done by calling a `Vote` method with a `Ballot` containing preferences for a single vote. `Record` is returned that can be used to undo the vote, if needed.

`Unvote` removes the vote represented with a `Record` from the voting.

`ComputeSorted` returns an ordered list of all choices based on submitted votes. If only partial list is needed, method `Compute` can be used to minimize memory usage and to avoid allocating a list of all choices.

## Example

```go
package main

import (
 "context"
 "fmt"
 "log"
 "os"

 "resenje.org/markus"
)

func main() {
 dir, err := os.MkdirTemp("", "")
 if err != nil {
  log.Fatal(err)
 }
 defer os.RemoveAll(dir)

 // Create a new voting.
 v, err := markus.NewVoting(dir)
 if err != nil {
  log.Fatal(err)
 }
 defer v.Close()

 // Add choices.
 if _, _, err := v.AddChoices(3); err != nil {
  log.Fatal(err)
 }

 // First vote.
 record1, err := v.Vote(markus.Ballot{
  0: 1,
  2: 2,
 })
 if err != nil {
  log.Fatal(err)
 }

 // Second vote.
 if _, err := v.Vote(markus.Ballot{
  1: 1,
  2: 2,
 }); err != nil {
  log.Fatal(err)
 }

 // A vote can be changed.
 if err := v.Unvote(record1); err != nil {
  log.Fatal(err)
 }
 if _, err := v.Vote(markus.Ballot{
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
}
```

## License

This application is distributed under the BSD-style license found in the [LICENSE](LICENSE) file.
