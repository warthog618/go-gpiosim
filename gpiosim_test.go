// SPDX-FileCopyrightText: 2023 Kent Gibson <warthog618@gmail.com>
//
// SPDX-License-Identifier: Apache-2.0 OR MIT

package gpiosim_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/warthog618/go-gpiosim"
	"github.com/warthog618/gpiod"
)

func TestNewSim(t *testing.T) {
	s, err := gpiosim.NewSim(
		gpiosim.WithName("gpiosim_test"),
		gpiosim.WithBank(gpiosim.NewBank("left", 8,
			gpiosim.WithNamedLine(3, "LED0"),
			gpiosim.WithNamedLine(5, "BUTTON1"),
			gpiosim.WithHoggedLine(2, "piggy", gpiosim.HogDirectionOutputLow),
		)),
		gpiosim.WithBank(gpiosim.NewBank("right", 42,
			gpiosim.WithNamedLine(3, "BUTTON2"),
			gpiosim.WithNamedLine(4, "LED2"),
			gpiosim.WithHoggedLine(7, "hogster", gpiosim.HogDirectionOutputHigh),
			gpiosim.WithHoggedLine(9, "piggy", gpiosim.HogDirectionInput),
		)),
	)
	require.Nil(t, err)
	defer s.Close()

	// Chip0
	// check config
	assert.Equal(t, "gpiosim_test", s.Name)
	require.Equal(t, 2, len(s.Chips))
	k := s.Chips[0].Config()
	assert.Equal(t, 8, k.NumLines)
	assert.Equal(t, 2, len(k.Names))
	assert.Equal(t, 1, len(k.Hogs))
	// check device created
	p := s.Chips[0].DevPath()
	assert.FileExists(t, p)
	// check uAPI info
	c, err := gpiod.NewChip(p)
	require.Nil(t, err)
	assert.Equal(t, k.NumLines, c.Lines())
	assert.Equal(t, k.Label, c.Label)
	checkLineInfo(t, c, k)

	// Chip1
	// check config
	k = s.Chips[1].Config()
	assert.Equal(t, 42, k.NumLines)
	assert.Equal(t, 2, len(k.Names))
	assert.Equal(t, 2, len(k.Hogs))
	// check device created
	p = s.Chips[1].DevPath()
	assert.FileExists(t, p)
	// check uAPI info
	c, err = gpiod.NewChip(p)
	require.Nil(t, err)
	assert.Equal(t, k.NumLines, c.Lines())
	assert.Equal(t, k.Label, c.Label)
	checkLineInfo(t, c, k)

	// non-unique name
	bs, err := gpiosim.NewSim(
		gpiosim.WithName("gpiosim_test"),
		gpiosim.WithBank(gpiosim.NewBank("left", 8)),
	)
	assert.NotNil(t, err)
	assert.Nil(t, bs)

	// no banks
	bs, err = gpiosim.NewSim()
	assert.NotNil(t, err)
	assert.Nil(t, bs)

	p0 := s.Chips[0].DevPath()
	s.Close()
	assert.NoFileExists(t, p0)
	assert.NoFileExists(t, p)
}

func checkLineInfo(t *testing.T, c *gpiod.Chip, k gpiosim.Bank) {
	for o := 0; o < k.NumLines; o++ {
		xli := gpiod.LineInfo{
			Offset: o,
			Config: gpiod.LineConfig{Direction: gpiod.LineDirectionInput},
		}
		if name, ok := k.Names[o]; ok {
			xli.Name = name
		}
		if hog, ok := k.Hogs[o]; ok {
			if hog.Direction != gpiosim.HogDirectionInput {
				xli.Config.Direction = gpiod.LineDirectionOutput
			}
			xli.Used = true
			xli.Consumer = hog.Consumer
		}
		li, err := c.LineInfo(o)
		assert.Nil(t, err)
		assert.Equal(t, xli, li)
	}
}

func checkLineLevel(t *testing.T, l *gpiod.Line, xv int) {
	v, err := l.Value()
	assert.Nil(t, err)
	assert.Equal(t, xv, v)
}

func checkChipPull(t *testing.T, c *gpiosim.Chip, offset, xv int) {
	v, err := c.Pull(offset)
	assert.Nil(t, err)
	assert.Equal(t, xv, v)
}

func TestChipPull(t *testing.T) {
	s, err := gpiosim.NewSim(
		gpiosim.WithBank(gpiosim.NewBank("left", 8)),
		gpiosim.WithBank(gpiosim.NewBank("right", 42)),
	)
	require.Nil(t, err)
	defer s.Close()

	offset := 3
	c := &s.Chips[0]
	l, err := gpiod.RequestLine(c.DevPath(), offset, gpiod.AsInput)
	require.Nil(t, err)
	defer l.Close()

	checkLineLevel(t, l, 0)

	// pull-up
	err = c.SetPull(offset, 1)
	assert.Nil(t, err)
	checkLineLevel(t, l, 1)
	checkChipPull(t, c, offset, 1)

	// pull-down
	err = c.SetPull(offset, 0)
	assert.Nil(t, err)
	checkLineLevel(t, l, 0)
	checkChipPull(t, c, offset, 0)

	// functional variants
	err = c.Pullup(offset)
	assert.Nil(t, err)
	checkLineLevel(t, l, 1)
	checkChipPull(t, c, offset, 1)

	err = c.Pulldown(offset)
	assert.Nil(t, err)
	checkLineLevel(t, l, 0)
	checkChipPull(t, c, offset, 0)

	// Toggle
	err = c.Toggle(offset)
	assert.Nil(t, err)
	checkLineLevel(t, l, 1)
	checkChipPull(t, c, offset, 1)
	err = c.Toggle(offset)
	assert.Nil(t, err)
	checkLineLevel(t, l, 0)
	checkChipPull(t, c, offset, 0)
}

func TestChipCloseWithRequestedLines(t *testing.T) {
	s, err := gpiosim.NewSim(
		gpiosim.WithBank(gpiosim.NewBank("left", 8)),
		gpiosim.WithBank(gpiosim.NewBank("right", 42)),
	)
	require.Nil(t, err)
	defer s.Close()

	offset := 3
	c := &s.Chips[0]
	l, err := gpiod.RequestLine(c.DevPath(), offset, gpiod.AsInput)
	require.Nil(t, err)
	checkLineLevel(t, l, 0)
	s.Close()
}

func checkChipLevel(t *testing.T, c *gpiosim.Chip, offset, v int) {
	lv, err := c.Level(offset)
	assert.Nil(t, err)
	assert.Equal(t, v, lv)
}

func TestChipLevel(t *testing.T) {
	s, err := gpiosim.NewSim(
		gpiosim.WithBank(gpiosim.NewBank("left", 8)),
		gpiosim.WithBank(gpiosim.NewBank("right", 42)),
	)
	require.Nil(t, err)
	defer s.Close()

	offset := 3
	c := &s.Chips[0]
	l, err := gpiod.RequestLine(c.DevPath(), offset, gpiod.AsOutput(0))
	require.Nil(t, err)
	defer l.Close()
	checkChipLevel(t, c, offset, 0)
	checkChipPull(t, c, offset, 0)

	// pull-up
	err = l.SetValue(1)
	assert.Nil(t, err)
	checkChipLevel(t, c, offset, 1)
	checkChipPull(t, c, offset, 0) // driven level does not effect pull

	// pull-down
	err = l.SetValue(0)
	assert.Nil(t, err)
	checkChipLevel(t, c, offset, 0)
	checkChipPull(t, c, offset, 0)
}

func TestNewSimpleton(t *testing.T) {
	s, err := gpiosim.NewSimpleton(8)
	require.Nil(t, err)
	defer s.Close()

	// check config
	k := s.Config()
	assert.Equal(t, 8, k.NumLines)
	assert.Zero(t, len(k.Names))
	assert.Zero(t, len(k.Hogs))
	// check device created
	p := s.DevPath()
	assert.FileExists(t, p)
	// check uAPI info
	c, err := gpiod.NewChip(p)
	require.Nil(t, err)
	assert.Equal(t, k.NumLines, c.Lines())
	assert.Equal(t, k.Label, c.Label)
	checkLineInfo(t, c, k)
	s.Close()
	assert.NoFileExists(t, p)
}

func checkSimpletonPull(t *testing.T, s *gpiosim.Simpleton, offset, v int) {
	lv, err := s.Pull(offset)
	assert.Nil(t, err)
	assert.Equal(t, v, lv)
}

func TestSimpletonPull(t *testing.T) {
	s, err := gpiosim.NewSimpleton(8)
	require.Nil(t, err)
	defer s.Close()

	offset := 3
	l, err := gpiod.RequestLine(s.DevPath(), offset, gpiod.AsInput)
	require.Nil(t, err)
	defer l.Close()

	checkLineLevel(t, l, 0)

	// pull-up
	err = s.SetPull(offset, 1)
	assert.Nil(t, err)
	checkLineLevel(t, l, 1)
	checkSimpletonPull(t, s, offset, 1)

	// pull-down
	err = s.SetPull(3, 0)
	assert.Nil(t, err)
	checkLineLevel(t, l, 0)
	checkSimpletonPull(t, s, offset, 0)

	// functional variants
	err = s.Pullup(3)
	assert.Nil(t, err)
	checkLineLevel(t, l, 1)
	checkSimpletonPull(t, s, offset, 1)

	err = s.Pulldown(3)
	assert.Nil(t, err)
	checkLineLevel(t, l, 0)
	checkSimpletonPull(t, s, offset, 0)

	// Toggle
	err = s.Toggle(3)
	assert.Nil(t, err)
	checkSimpletonPull(t, s, offset, 1)
	checkLineLevel(t, l, 1)
	err = s.Toggle(3)
	assert.Nil(t, err)
	checkLineLevel(t, l, 0)
	checkSimpletonPull(t, s, offset, 0)
}

func checkSimpletonLevel(t *testing.T, s *gpiosim.Simpleton, offset, v int) {
	lv, err := s.Level(offset)
	assert.Nil(t, err)
	assert.Equal(t, v, lv)
}

func TestSimpletonLevel(t *testing.T) {
	s, err := gpiosim.NewSimpleton(8)
	require.Nil(t, err)
	defer s.Close()

	offset := 3
	l, err := gpiod.RequestLine(s.DevPath(), offset, gpiod.AsOutput(0))
	require.Nil(t, err)
	defer l.Close()
	checkSimpletonLevel(t, s, offset, 0)
	checkSimpletonPull(t, s, offset, 0)

	// pull-up
	err = l.SetValue(1)
	assert.Nil(t, err)
	checkSimpletonLevel(t, s, offset, 1)
	checkSimpletonPull(t, s, offset, 0) // driven level does not effect pull

	// pull-down
	err = l.SetValue(0)
	assert.Nil(t, err)
	checkSimpletonLevel(t, s, offset, 0)
	checkSimpletonPull(t, s, offset, 0)
}
