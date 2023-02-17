// Copyright (c) 2023, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package markus

import "fmt"

type UnknownChoiceError struct {
	Index uint64
}

func (e *UnknownChoiceError) Error() string {
	return fmt.Sprintf("markus: unknown choice %v", e.Index)
}
