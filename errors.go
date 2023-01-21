// Copyright (c) 2023, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package markus

import "fmt"

type UnknownChoiceError[C comparable] struct {
	Choice C
}

func (e *UnknownChoiceError[C]) Error() string {
	return fmt.Sprintf("schulze: unknown choice %v", e.Choice)
}
