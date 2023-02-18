// Copyright (c) 2023, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build windows

package matrix

import (
	"os"
	"reflect"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

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

	high := uint32(size >> 32)
	low := uint32(size) & 0xffffffff
	h, errno := syscall.CreateFileMapping(syscall.Handle(f.Fd()), nil, windows.PAGE_READWRITE, high, low, nil)
	if h == 0 {
		return nil, os.NewSyscallError("CreateFileMapping", errno)
	}

	addr, errno := syscall.MapViewOfFile(h, windows.FILE_MAP_WRITE, 0, 0, uintptr(size))
	if addr == 0 {
		_ = syscall.CloseHandle(h)
		return nil, os.NewSyscallError("MapViewOfFile", errno)
	}

	if err := syscall.CloseHandle(syscall.Handle(h)); err != nil {
		return nil, os.NewSyscallError("CloseHandle", err)
	}

	data := ((*[maxMapSize]byte)(unsafe.Pointer(addr)))[:size:size]

	m := mmap(data)

	return &m, nil
}

func (m *mmap) sync() error {
	if len(*m) == 0 {
		return nil
	}
	header := (*reflect.SliceHeader)(unsafe.Pointer(m))
	errno := windows.FlushViewOfFile(header.Data, uintptr(header.Len))
	if errno != nil {
		return os.NewSyscallError("FlushViewOfFile", errno)
	}

	return nil
}

func (m *mmap) unmap() error {
	if len(*m) == 0 {
		return nil
	}
	err := syscall.UnmapViewOfFile((uintptr)(unsafe.Pointer(&(*m)[0])))
	*m = nil
	if err != nil {
		return os.NewSyscallError("UnmapViewOfFile", err)
	}
	return nil
}

func flock(f *os.File) error {
	var m1 uint32 = (1 << 32) - 1 // -1 in a uint32
	return windows.LockFileEx(windows.Handle(f.Fd()), windows.LOCKFILE_FAIL_IMMEDIATELY|windows.LOCKFILE_EXCLUSIVE_LOCK, 0, 1, 0, &windows.Overlapped{
		Offset:     m1,
		OffsetHigh: m1,
	})
}

func funlock(f *os.File) error {
	var m1 uint32 = (1 << 32) - 1 // -1 in a uint32
	return windows.UnlockFileEx(windows.Handle(f.Fd()), 0, 1, 0, &windows.Overlapped{
		Offset:     m1,
		OffsetHigh: m1,
	})
}
