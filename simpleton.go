// SPDX-FileCopyrightText: 2023 Kent Gibson <warthog618@gmail.com>
//
// SPDX-License-Identifier: Apache-2.0 OR MIT

package gpiosim

type Simpleton struct {
	Sim
}

func NewSimpleton(numLines int) (*Simpleton, error) {
	s, err := NewSim(WithBank(NewBank("simpleton", numLines)))
	if s == nil {
		return nil, err
	}
	return &Simpleton{*s}, err
}

// ChipName returns the name of the gpiochip.
//
// e.g. "gpiochip0"
func (s *Simpleton) ChipName() string {
	return s.Chips[0].chipName
}

// Config returns the configuration used for the Chip.
func (s *Simpleton) Config() Bank {
	return s.Chips[0].cfg
}

// DevPath returns the path to the gpiochip device.
//
// e.g. "/dev/gpiochip0"
func (s *Simpleton) DevPath() string {
	return s.Chips[0].devPath
}

// Level returns the level the line is being pulled to.
//
// If the line is requested as an output then this is the level userspace is
// driving it to, and otherwise there is little point calling this method -
// you probably should be calling Pull instead.
func (s *Simpleton) Level(offset int) (int, error) {
	return s.Chips[0].Level(offset)
}

// Pull returns the current the pull of the given line.
func (s *Simpleton) Pull(offset int) (int, error) {
	return s.Chips[0].Pull(offset)
}

// Pulldown sets the pull of the given line to pull-down.
func (s *Simpleton) Pulldown(offset int) error {
	return s.Chips[0].Pulldown(offset)
}

// Pullup sets the pull of the given line to pull-up.
func (s *Simpleton) Pullup(offset int) error {
	return s.Chips[0].Pullup(offset)
}

// SetPull sets the pull of the given line.
func (s *Simpleton) SetPull(offset int, level int) error {
	return s.Chips[0].SetPull(offset, level)
}

// Toggle flips the pull of the given line.
//
// If it was pull-up it becomes pull-down, and vice versa.
func (s *Simpleton) Toggle(offset int) error {
	return s.Chips[0].Toggle(offset)
}
