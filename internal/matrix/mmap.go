// Copyright (c) 2023, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build !windows

package matrix

import (
	"os"
	"syscall"

	"golang.org/x/sys/unix"
)

type mmap []byte

var _ = maxMapSize // avoid unused warning, used in mmap_windows.go

func newMmap(f *os.File) (*mmap, error) {
	fi, err := f.Stat()
	if err != nil {
		return nil, err
	}

	size := fi.Size()
	if size == 0 {
		m := mmap(make([]byte, 0))
		return &m, nil
	}

	data, err := unix.Mmap(int(f.Fd()), 0, int(size), unix.PROT_READ|unix.PROT_WRITE, unix.MAP_SHARED)
	if err != nil {
		return nil, err
	}

	m := mmap(data)
	return &m, nil
}

func (m *mmap) sync() error {
	if len(*m) == 0 {
		return nil
	}
	return unix.Msync(*m, unix.MS_SYNC)
}

func (m *mmap) unmap() error {
	if len(*m) == 0 {
		return nil
	}
	err := unix.Munmap(*m)
	*m = nil
	return err
}

func flock(f *os.File) error {
	return syscall.Flock(int(f.Fd()), syscall.LOCK_NB|syscall.LOCK_EX)
}

func funlock(f *os.File) error {
	return syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
}
