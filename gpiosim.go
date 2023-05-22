// SPDX-FileCopyrightText: 2023 Kent Gibson <warthog618@gmail.com>
//
// SPDX-License-Identifier: MIT

// Package gpiosim is a library for creating and controlling GPIO simulators for testing users of
// the Linux GPIO uAPI (both v1 and v2).
//
// The simulators are provided by the Linux **gpio-sim** kernel module and require a
// recent kernel (v5.19 or later) built with **CONFIG_GPIO_SIM**.
//
// Simulators ([`Sim`]) contain one or more chips, each with a collection of lines being
// simulated. The [`Builder`] is responsible for constructing the [`Sim`] and taking it live.
// Configuring a simulator involves adding [`Bank`]s, representing the
// chip, to the builder, then taking the simulator live.
//
// Once live, the [`Chip`] exposes lines which may be manipulated to drive the
// GPIO uAPI from the kernel side.
// For input lines, applying a pull using [`Chip.Pull`] and related
// methods controls the level of the simulated line.  For output lines,
// [`Chip.Level`] returns the level the simulated line is being driven to.
//
// For simple tests that only require lines on a single chip, the [`Simpleton`]
// provides a simplified interface.
//
// Configuring a simulator involves *configfs*, and manipulating the chips once live
// involves *sysfs*, so root permissions are typically required to run a simulator.
//
// ## Example Usage
//
// Creating a simulator with two chips, with 8 and 42 lines respectively, each with
// several named lines and a hogged line:
//
// ```no_run
// s, err := gpiosim.NewSim(
//	gpiosim.WithName("gpiosim_test"),
//	gpiosim.WithBank(gpiosim.NewBank("left", 8,
//		gpiosim.WithNamedLine(3, "LED0"),
//		gpiosim.WithNamedLine(5, "BUTTON1"),
//		gpiosim.WithHoggedLine(2, "piggy", gpiosim.HogDirectionOutputLow),
//	)),
//	gpiosim.WithBank(gpiosim.NewBank("right", 42,
//		gpiosim.WithNamedLine(3, "BUTTON2"),
//		gpiosim.WithNamedLine(4, "LED2"),
//		gpiosim.WithHoggedLine(7, "hogster", gpiosim.HogDirectionOutputHigh),
//		gpiosim.WithHoggedLine(9, "piggy", gpiosim.HogDirectionInput),
//	)),
// )
// c := &sim.Chips[0]
// c.Pullup(5);
// level, err := c.Level(3);
// # }
// ```
//
// Use a simpleton to create a single chip simulator with 12 lines, for situations
// where multiple chips or named lines are not required:
//
// ```
// s, err := gpiosim.NewSimpleton(12)
// s.SetPull(5, 1)
// level, err := s.Level(3)
// ```

package gpiosim

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path"
	"strings"
	"sync/atomic"

	"github.com/pkg/errors"
)

type Sim struct {
	// The name of the simulator in configfs and sysfs space.
	Name string

	// The details of the chips being simulated.
	Chips []Chip

	/// Path to the gpio-sim in configfs.
	configfsPath string
}

type Simpleton struct {
	Sim
}

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

type builder struct {
	// The name for the simulator in the configfs space.
	//
	// If empty when [`Live`] is called then a unique name is generated.
	//
	// [`Live`]: Builder::live
	name string // optional

	// The details of the banks to be simulated.
	//
	// Each bank becomes a chip when the simulator goes live.
	banks []Bank
}

// SimOption defines the interface required to provide a Sim option.
type SimOption interface {
	applySimOption(*builder)
}

type WithNameOption string

func WithName(name string) WithNameOption {
	return WithNameOption(name)
}

func (o WithNameOption) applySimOption(b *builder) {
	b.name = string(o)
}

func WithBank(b *Bank) Bank {
	return *b
}

func (o Bank) applySimOption(b *builder) {
	b.banks = append(b.banks, Bank(o))
}

// BankOption defines the interface required to provide a Bank option.
type BankOption interface {
	applyBankOption(*Bank)
}

func WithNamedLine(offset int, name string) NamedLine {
	return NamedLine{offset, name}
}

func (o NamedLine) applyBankOption(b *Bank) {
	if b.Names == nil {
		b.Names = make(map[int]string)
	}
	b.Names[o.Offset] = o.Name
}

func WithHoggedLine(offset int, consumer string, direction HogDirection) HoggedLine {
	return HoggedLine{offset, Hog{consumer, direction}}
}

func (o HoggedLine) applyBankOption(b *Bank) {
	if b.Hogs == nil {
		b.Hogs = make(map[int]Hog)
	}
	b.Hogs[o.offset] = o.Hog
}

func NewBank(label string, numLines int, opts ...BankOption) *Bank {
	b := &Bank{Label: label, NumLines: numLines}
	for _, o := range opts {
		o.applyBankOption(b)
	}
	return b
}

func NewSim(opts ...SimOption) (*Sim, error) {
	b := builder{}
	for _, o := range opts {
		o.applySimOption(&b)
	}
	return b.live()
}

type Bank struct {
	// The number of lines simulated by this bank.
	NumLines int

	// The label of the chip.
	Label string

	// Lines assigned a name.
	Names map[int]string

	// Lines that appear to be already in use by some other entity.
	Hogs map[int]Hog
}

type Hog struct {
	// The name of the consumer that appears to be using the line.
	Consumer string

	// The requested direction for the hogged line, and if an
	// output then the pull.
	Direction HogDirection
}

type HoggedLine struct {
	offset int
	Hog
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

const (
	// Line is inactive.
	LevelInactive int = iota

	// Line is active.
	LevelActive
)

type NamedLine struct {
	Offset int
	Name   string
}

func NewSimpleton(numLines int) (*Simpleton, error) {
	s, err := NewSim(WithBank(NewBank("simpleton", numLines)))
	if s == nil {
		return nil, err
	}
	return &Simpleton{*s}, err
}

func appName() string {
	str, err := os.Executable()
	if err != nil {
		return "gpiosim"
	}
	return path.Base(str)
}

var simCounter uint32 = 0

func uniqueName() string {
	return fmt.Sprintf("%s-p%d-%d", appName(), os.Getpid(), atomic.AddUint32(&simCounter, 1))
}

func configfsMountPoint() (string, error) {
	file, err := os.Open("/proc/mounts")
	if err != nil {
		return "", err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		words := strings.Fields(scanner.Text())
		if len(words) >= 6 && words[2] == "configfs" {
			return words[1], nil
		}
	}
	// not mounted, so try to mount
	configfs := "/sys/kernel/config"
	cmd := exec.Command("mount", "-t", "configfs", "configfs", configfs)
	if err = cmd.Run(); err == nil {
		return configfs, nil
	}
	return "", errors.New("can't find configfs mountpoint")
}

func findConfigfsPath() (string, error) {
	configfs := "/sys/kernel/config/gpio-sim"
	if _, err := os.Stat(configfs); err == nil {
		return configfs, nil
	}
	// try loading gpio-sim module
	cmd := exec.Command("modprobe", "gpio-sim")
	if err := cmd.Run(); err == nil {
		if _, err := os.Stat(configfs); err == nil {
			return configfs, nil
		}
	}
	// check mountpoints in case configfs is mounted somewhere unusual
	if configfs, err := configfsMountPoint(); err == nil {
		configfs = path.Join(configfs, "gpio-sim")
		if _, err := os.Stat(configfs); err == nil {
			return configfs, nil
		}
	}
	return "", errors.New("gpio-sim module not loaded")
}

func (b *builder) live() (*Sim, error) {
	if len(b.banks) == 0 {
		return nil, errors.New("no banks defined")
	}
	if len(b.name) == 0 {
		b.name = uniqueName()
	}
	configfsPath, err := findConfigfsPath()
	if err != nil {
		return nil, err
	}
	configfsPath = path.Join(configfsPath, b.name)
	if _, err := os.Stat(configfsPath); err == nil {
		return nil, errors.Errorf("sim with name '%s' already exists", b.name)
	}

	s := Sim{Name: b.name, configfsPath: configfsPath}
	for _, k := range b.banks {
		s.Chips = append(s.Chips, Chip{cfg: k})
	}
	err = s.setupConfigfs()
	if err == nil {
		err = writeAttr(s.configfsPath, "live", "1")
	}
	if err != nil {
		s.Close()
		return nil, err
	}
	devName, err := readAttr(s.configfsPath, "dev_name")
	if err != nil {
		s.Close()
		return nil, err
	}
	for i := range s.Chips {
		bankPath := path.Join(s.configfsPath, fmt.Sprintf("bank%d", i))
		chipName, err := readAttr(bankPath, "chip_name")
		if err != nil {
			s.Close()
			return nil, err
		}
		s.Chips[i].devName = devName
		s.Chips[i].chipName = chipName
		s.Chips[i].devPath = path.Join("/dev", chipName)
		s.Chips[i].sysfsPath = path.Join("/sys/devices/platform", devName, chipName)
	}
	return &s, nil
}

func (s *Sim) setupConfigfs() error {
	for i, c := range s.Chips {
		bankPath := path.Join(s.configfsPath, fmt.Sprintf("bank%d", i))
		if err := os.MkdirAll(bankPath, 0755); err != nil {
			return err
		}
		if err := writeAttr(bankPath, "label", c.cfg.Label); err != nil {
			return err
		}
		if err := writeAttr(bankPath, "num_lines", fmt.Sprintf("%d", c.cfg.NumLines)); err != nil {
			return err
		}
		for o, n := range c.cfg.Names {
			linePath := path.Join(bankPath, fmt.Sprintf("line%d", o))
			if err := os.Mkdir(linePath, 0755); err != nil {
				return err
			}
			if err := writeAttr(linePath, "name", n); err != nil {
				return err
			}
		}
		for o, h := range c.cfg.Hogs {
			hogPath := path.Join(bankPath, fmt.Sprintf("line%d", o), "hog")
			if err := os.MkdirAll(hogPath, 0755); err != nil {
				return err
			}
			if err := writeAttr(hogPath, "name", h.Consumer); err != nil {
				return err
			}
			if err := writeAttr(hogPath, "direction", hogDirectionToString(h.Direction)); err != nil {
				return err
			}
		}
	}
	return nil
}

func hogDirectionToString(d HogDirection) string {
	switch d {
	case HogDirectionOutputLow:
		return "output-low"
	case HogDirectionOutputHigh:
		return "output-high"
	default:
		return "input"
	}
}

func (s *Sim) cleanupConfigfs() error {
	err := writeAttr(s.configfsPath, "live", "0")
	if err != nil {
		return err
	}
	for i, c := range s.Chips {
		bankPath := path.Join(s.configfsPath, fmt.Sprintf("bank%d", i))
		if _, err := os.Stat(bankPath); err != nil {
			continue
		}
		for o := range c.cfg.Hogs {
			linePath := path.Join(bankPath, fmt.Sprintf("line%d", o))
			os.Remove(path.Join(linePath, "hog"))
			os.Remove(linePath)
		}
		for o := range c.cfg.Names {
			linePath := path.Join(bankPath, fmt.Sprintf("line%d", o))
			os.Remove(linePath)
		}
		os.Remove(bankPath)
	}
	os.Remove(s.configfsPath)
	return nil
}

func (s *Sim) Close() {
	s.cleanupConfigfs()
	s.Chips = nil
}

func (s *Simpleton) Config() Bank {
	return s.Chips[0].cfg
}

func (s *Simpleton) Pull(offset int) (int, error) {
	return s.Chips[0].Pull(offset)
}

func (s *Simpleton) SetPull(offset int, level int) error {
	return s.Chips[0].SetPull(offset, level)
}

func (s *Simpleton) Pullup(offset int) error {
	return s.Chips[0].Pullup(offset)
}

func (s *Simpleton) Pulldown(offset int) error {
	return s.Chips[0].Pulldown(offset)
}

func (s *Simpleton) Toggle(offset int) error {
	return s.Chips[0].Toggle(offset)
}

func (s *Simpleton) Level(offset int) (int, error) {
	return s.Chips[0].Level(offset)
}

func (s *Simpleton) ChipName() string {
	return s.Chips[0].chipName
}

func (s *Simpleton) DevPath() string {
	return s.Chips[0].devPath
}

func (c *Chip) Config() Bank {
	return c.cfg
}

func (c *Chip) ChipName() string {
	return c.chipName
}

func (c *Chip) DevPath() string {
	return c.devPath
}

func (c *Chip) attr(offset int, name string) (string, error) {
	return readAttr(path.Join(c.sysfsPath, fmt.Sprintf("sim_gpio%d", offset)), name)
}

func (c *Chip) setAttr(offset int, name, value string) error {
	return writeAttr(path.Join(c.sysfsPath, fmt.Sprintf("sim_gpio%d", offset)), name, value)
}

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

func readAttr(p, attr string) (string, error) {
	data, err := os.ReadFile(path.Join(p, attr))
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}

func writeAttr(p, attr, value string) error {
	return os.WriteFile(path.Join(p, attr), []byte(value), 0666)
}

func (c *Chip) SetPull(offset int, level int) error {
	l := "pull-down"
	if level == LevelActive {
		l = "pull-up"
	}
	return c.setAttr(offset, "pull", l)
}

func (c *Chip) Pulldown(offset int) error {
	return c.SetPull(offset, LevelInactive)
}

func (c *Chip) Pullup(offset int) error {
	return c.SetPull(offset, LevelActive)
}

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
