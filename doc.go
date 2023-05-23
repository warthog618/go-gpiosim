// SPDX-FileCopyrightText: 2023 Kent Gibson <warthog618@gmail.com>
//
// SPDX-License-Identifier: Apache-2.0 OR MIT

/*
Package gpiosim is a library for creating and controlling GPIO simulators for testing
users of the Linux GPIO uAPI (both v1 and v2).

The simulators are provided by the Linux [gpio-sim] kernel module and require a
recent kernel (5.19 or later) built with CONFIG_GPIO_SIM.

Simulators ([Sim]) contain one or more [Chip]s, each with a collection of lines being
simulated. Configuring a simulator involves adding [Bank]s, eash representing a
chip, to [NewSim], which will construct the corresponding simulator using
gpio-sim and take it live.

Once live, the [Chip] exposes lines which may be manipulated to drive the
GPIO uAPI from the kernel side.
For input lines, applying a pull using [Chip.SetPull], or related methods, controls
the level of the simulated line.  For output lines, the [Chip.Level] method returns
the level the simulated line is being driven to by userspace.

For tests that only require vanilla lines on a single chip, the [Simpleton]
provides a slightly simpler interface.

Closing the [Sim] deconstructs the simulator, removing the gpio-sim
configuration and the corresponding gpiochips.

Configuring a simulator involves configfs, and manipulating the chips once live
involves sysfs, so root permissions are typically required to run a simulator.

# Example Usage

Create a [Simpleton] with 12 lines:

	s, err := gpiosim.NewSimpleton(12)
	s.SetPull(5, 1)
	level, err := s.Level(3)

Creating a simulator with two chips, with 8 and 42 lines respectively, each with
several named lines and some hogged lines:

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
	c := &sim.Chips[0]
	c.Pullup(5);
	level, err := c.Level(3);

[gpio-sim]: https://www.kernel.org/doc/html/latest/admin-guide/gpio/gpio-sim.html
*/
package gpiosim
