<!--
SPDX-FileCopyrightText: 2023 Kent Gibson <warthog618@gmail.com>

SPDX-License-Identifier: CC0-1.0
-->
# gpiosim

[![Build Status](https://img.shields.io/github/actions/workflow/status/warthog618/go-gpiosim/go.yml?logo=github&branch=master)](https://github.com/warthog618/go-gpiosim/actions/workflows/go.yml)
[![PkgGoDev](https://pkg.go.dev/badge/github.com/warthog618/go-gpiosim)](https://pkg.go.dev/github.com/warthog618/go-gpiosim)
[![Go Report Card](https://goreportcard.com/badge/github.com/warthog618/go-gpiosim)](https://goreportcard.com/report/github.com/warthog618/go-gpiosim)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://github.com/warthog618/go-gpiosim/blob/master/LICENSE)

A Go library for creating and controlling GPIO simulators for testing users of
the Linux GPIO uAPI (both v1 and v2).

The simulators are provided by the Linux [**gpio-sim**](https://www.kernel.org/doc/html/latest/admin-guide/gpio/gpio-sim.html) kernel module and so require a
recent kernel (5.19 or later) built with **CONFIG_GPIO_SIM**.

Simulators contain one or more **Chip**s, each with a collection of lines being
simulated. Configuring a simulator involves adding **Bank**s, eash representing a
chip, to **NewSim**, which will construct the corresponding simulator using
**gpio-sim** and take it live.

Once live, the **Chip** exposes lines which may be manipulated to drive the
GPIO uAPI from the kernel side.
For input lines, applying a pull using **Chip** pull methods controls the level
of the simulated line.  For output lines, the **Chip.Level** method returns
the level the simulated line is being driven to by userspace.

For tests that only require vanilla lines on a single chip, the **Simpleton**
provides a slightly simpler interface.

Closing the **Sim** deconstructs the simulator, removing the **gpio-sim**
configuration and the corresponding gpiochips.

Configuring a simulator involves *configfs*, and manipulating the chips once live
involves *sysfs*, so root permissions are typically required to run a simulator.

## Example Usage

Creating a simulator with two chips, with 8 and 42 lines respectively, each with
several named lines and some hogged lines:

```go
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
defer s.Close()
c := &sim.Chips[0]
c.Pullup(5);
level, err := c.Level(3);
```

Use a **Simpleton** to create a single chip simulator with 12 lines, for where multiple chips or
named lines are not required:

```go
s, err := gpiosim.NewSimpleton(12)
s.SetPull(5, 1)
level, err := s.Level(3)
```

## License

Licensed under either of

- Apache License, Version 2.0 ([LICENSE-APACHE](https://github.com/warthog618/go-gpiosim/blob/master/LICENSES/Apache-2.0.txt) or
  <http://www.apache.org/licenses/LICENSE-2.0>)
- MIT license ([LICENSE-MIT](https://github.com/warthog618/go-gpiosim/blob/master/LICENSES/MIT.txt) or <http://opensource.org/licenses/MIT>)

at your option.

## Contribution

Unless you explicitly state otherwise, any contribution intentionally submitted
for inclusion in the work by you, as defined in the Apache-2.0 license, shall be
dual licensed as above, without any additional terms or conditions.
