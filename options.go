// SPDX-FileCopyrightText: 2023 Kent Gibson <warthog618@gmail.com>
//
// SPDX-License-Identifier: Apache-2.0 OR MIT

package gpiosim

// NewSimOption defines the interface required to provide an option to NewSim.
type NewSimOption interface {
	applySimOption(*builder)
}

// WithBank returns an option that adds the given bank to the Sim.
func WithBank(b *Bank) Bank {
	return *b
}

func (o Bank) applySimOption(b *builder) {
	b.banks = append(b.banks, Bank(o))
}

// NewBankOption defines the interface required to provide an option to NewBank.
type NewBankOption interface {
	applyBankOption(*Bank)
}

// HoggedLine is an option that hogs a line.
type HoggedLine struct {
	offset int
	Hog
}

// WithHoggedLine returns an option to hog a simulated line.
//
// Hogging the line makes it appear in use by another consumer.
func WithHoggedLine(offset int, consumer string, direction HogDirection) HoggedLine {
	return HoggedLine{offset, Hog{consumer, direction}}
}

func (o HoggedLine) applyBankOption(b *Bank) {
	if b.Hogs == nil {
		b.Hogs = make(map[int]Hog)
	}
	b.Hogs[o.offset] = o.Hog
}

// NameOption defines the name for a Sim.
type NameOption string

// WithName returns an option that defines the name of a Sim.
func WithName(name string) NameOption {
	return NameOption(name)
}

func (o NameOption) applySimOption(b *builder) {
	b.name = string(o)
}

// NamedLine is an option that names a line.
type NamedLine struct {
	Offset int
	Name   string
}

// WithNamedLine returns an option that defines the name of a simulated line.
func WithNamedLine(offset int, name string) NamedLine {
	return NamedLine{offset, name}
}

func (o NamedLine) applyBankOption(b *Bank) {
	if b.Names == nil {
		b.Names = make(map[int]string)
	}
	b.Names[o.Offset] = o.Name
}
