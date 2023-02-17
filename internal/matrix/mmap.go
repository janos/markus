// Copyright (c) 2023, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package matrix

import (
	"os"

	"golang.org/x/sys/unix"
)

// todo: add support for windows

type mmap []byte

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

func (m mmap) Lock() error {
	return unix.Mlock([]byte(m))
}

func (m mmap) Unlock() error {
	return unix.Munlock([]byte(m))
}

func (m mmap) Sync() error {
	if len(m) == 0 {
		return nil
	}
	return unix.Msync([]byte(m), unix.MS_SYNC)
}

func (m mmap) Unmap() error {
	data := []byte(m)
	if len(data) == 0 {
		return nil
	}
	return unix.Munmap(data)
}
