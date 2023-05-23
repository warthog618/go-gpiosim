// SPDX-FileCopyrightText: 2023 Kent Gibson <warthog618@gmail.com>
//
// SPDX-License-Identifier: Apache-2.0 OR MIT

package gpiosim

// Bank contains the information required to configure a chip in a gpio-sim.
type Bank struct {
	// The number of lines simulated by this bank/chip.
	NumLines int

	// The label of the chip.
	Label string

	// Lines assigned an identifying name.
	//
	// Line names do not need to be unique.
	Names map[int]string

	// Lines that appear to be already in use by some other entity.
	Hogs map[int]Hog
}

// NewBank constructs a Bank with the label, numLines and options provided.
//
// The numLines is the number of lines to be simulated.
//
// The label is informational.  It is returned in the uAPI ChipInfo and is
// intended to assist in identifying chips in the system.
// In a testing context the label can be used to identify the role of the chip
// in the test.
//
// The available options are [WithNamedLine] and [WithHoggedLine].
func NewBank(label string, numLines int, options ...NewBankOption) *Bank {
	b := &Bank{Label: label, NumLines: numLines}
	for _, o := range options {
		o.applyBankOption(b)
	}
	return b
}

// Hog contains the details of a line hog, i.e. some other user of a line.
type Hog struct {
	// The name of the consumer that appears to be using the line.
	Consumer string

	// The requested direction for the hogged line, and if an
	// output then the direction of pull.
	Direction HogDirection
}

// HogDirection indicates the direction of a hogged line.
type HogDirection int

const (
	// Hogged line is requested as an input.
	HogDirectionInput HogDirection = iota

	// Hogged line is requested as an output pulled low.
	HogDirectionOutputLow

	// Hogged line is requested as an output pulled high.
	HogDirectionOutputHigh
)
