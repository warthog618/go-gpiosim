// SPDX-FileCopyrightText: 2023 Kent Gibson <warthog618@gmail.com>
//
// SPDX-License-Identifier: Apache-2.0 OR MIT

package gpiosim

import (
	"fmt"
	"path"

	"github.com/pkg/errors"
)

// Chip provides the interface to a simulated gpiochip.
//
// Lines are identified by offset into the chip, with offsets
// being in the range 0..Config().NumLines-1.
type Chip struct {
	// The path to the bank in configfs
	configfsPath string

	// The path to the chip in /dev
	devPath string

	// The name of the gpiochip in /dev and sysfs.
	chipName string

	// The name of the device in sysfs.
	devName string

	// The path to the chip in /sys/device/platform.
	sysfsPath string

	// The configuration for this chip
	cfg Bank
}

// ChipName returns the name of the gpiochip.
//
// e.g. gpiochip0
func (c *Chip) ChipName() string {
	return c.chipName
}

// Config returns the configuration used for the Chip.
func (c *Chip) Config() Bank {
	return c.cfg
}

// DevPath returns the path to the gpiochip device.
//
// e.g. "/dev/gpiochip0"
//
// This the path that should be opened to access the gpiochip via the uAPI.
func (c *Chip) DevPath() string {
	return c.devPath
}

// Level returns the level the line is being pulled to.
//
// If the line is requested as an output then this is the level userspace is
// driving it to, and otherwise there is little point calling this method -
// you probably should be calling Pull instead.
func (c *Chip) Level(offset int) (int, error) {
	v, err := c.attr(offset, "value")
	if err == nil {
		if v == "0" {
			return LevelInactive, nil
		}
		if v == "1" {
			return LevelActive, nil
		}
		err = errors.Errorf("unexpected level value: %s", v)
	}
	return LevelInactive, err
}

const (
	// Line is inactive.
	LevelInactive int = iota

	// Line is active.
	LevelActive
)

// Pull returns the current the pull of the given line.
func (c *Chip) Pull(offset int) (int, error) {
	v, err := c.attr(offset, "pull")
	if err == nil {
		if v == "pull-down" {
			return LevelInactive, nil
		}
		if v == "pull-up" {
			return LevelActive, nil
		}
		err = errors.Errorf("unexpected pull value: %s", v)
	}
	return LevelInactive, err
}

// Pulldown sets the pull of the given line to pull-down.
func (c *Chip) Pulldown(offset int) error {
	return c.SetPull(offset, LevelInactive)
}

// Pullup sets the pull of the given line to pull-up.
func (c *Chip) Pullup(offset int) error {
	return c.SetPull(offset, LevelActive)
}

// SetPull sets the pull of the given line.
func (c *Chip) SetPull(offset int, level int) error {
	l := "pull-down"
	if level == LevelActive {
		l = "pull-up"
	}
	return c.setAttr(offset, "pull", l)
}

// Toggle flips the pull of the given line.
//
// If it was pull-up it becomes pull-down, and vice versa.
func (c *Chip) Toggle(offset int) error {
	p, err := c.Pull(offset)
	if err != nil {
		return err
	}
	if p == 0 {
		p = 1
	} else {
		p = 0
	}
	return c.SetPull(offset, p)
}

// attr reads the given line attribute from sysfs
func (c *Chip) attr(offset int, name string) (string, error) {
	return readAttr(path.Join(c.sysfsPath, fmt.Sprintf("sim_gpio%d", offset)), name)
}

// setAttr writes the given line attribute to sysfs
func (c *Chip) setAttr(offset int, name, value string) error {
	return writeAttr(path.Join(c.sysfsPath, fmt.Sprintf("sim_gpio%d", offset)), name, value)
}
