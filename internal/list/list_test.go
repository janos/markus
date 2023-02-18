// Copyright (c) 2023, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package list_test

import (
	"path/filepath"
	"testing"

	"resenje.org/markus/internal/list"
)

func TestList(t *testing.T) {
	dir := t.TempDir()
	l, err := list.New[string](filepath.Join(dir, "list"))
	if err != nil {
		t.Fatal(err)
	}
	n, err := l.Append("A", "B", "C")
	if err != nil {
		t.Fatal(err)
	}
	if n != 3 {
		t.Fatalf("got %d, want %d", n, 3)
	}
	if size := l.Size(); size != 3 {
		t.Fatalf("got %d, want %d", size, 3)
	}

	indexB, has := l.Index("B")
	if !has {
		t.Fatal("B not found")
	}
	if indexB != 1 {
		t.Fatalf("got %d, want %d", indexB, 1)
	}

	n, err = l.Append("D", "E")
	if err != nil {
		t.Fatal(err)
	}
	if n != 2 {
		t.Fatalf("got %d, want %d", n, 2)
	}

	indexE, has := l.Index("E")
	if !has {
		t.Fatal("E not found")
	}
	if indexE != 4 {
		t.Fatalf("got %d, want %d", indexE, 4)
	}
	if size := l.Size(); size != 5 {
		t.Fatalf("got %d, want %d", size, 5)
	}

	n, err = l.Append("C", "E", "F")
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("got %d, want %d", n, 1)
	}

	indexF, has := l.Index("F")
	if !has {
		t.Fatal("F not found")
	}
	if indexF != 5 {
		t.Fatalf("got %d, want %d", indexF, 5)
	}
	if size := l.Size(); size != 6 {
		t.Fatalf("got %d, want %d", size, 6)
	}
}
