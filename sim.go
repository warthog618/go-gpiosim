// SPDX-FileCopyrightText: 2023 Kent Gibson <warthog618@gmail.com>
//
// SPDX-License-Identifier: Apache-2.0 OR MIT

package gpiosim

import (
	"bufio"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path"
	"strings"
	"sync/atomic"

	"github.com/pkg/errors"
)

// Sim provides the interface to a simulator provided by gpio-sim.
//
// Each simulated chip is available through Chips, in the same order the banks
// were added to NewSim.
type Sim struct {
	// The name of the simulator in configfs and sysfs space.
	//
	// This is not something the user generally needs to be concerned with,
	// but is provided to assist with debugging.
	Name string

	// The details of the chips being simulated.
	Chips []Chip

	// Path to the gpio-sim in configfs.
	configfsPath string
}

// NewSim contstructs a Sim based on the provided options.
//
// The available options are [WithName] and [WithBank].
//
// Providing a WithName is optional, and is only necessary in rare cases.
// If you don't know if you need to provide a name then you don't.
// If a name is provided using WithName then that name must uniquely identify
// the sim on the system.
// If no name is provided then a unique name is automatically generated.
//
// At least one WithBank option must be provided.
func NewSim(options ...NewSimOption) (*Sim, error) {
	b := builder{}
	for _, o := range options {
		o.applySimOption(&b)
	}
	return b.live()
}

// Close deconstructs the sim, removing all gpio-sim configuration and the
// corresponding gpiochips.
func (s *Sim) Close() {
	s.cleanupConfigfs()
	s.Chips = nil
}

// cleanupConfigfs removes all the gpio-sim configurtation for the sim.
func (s *Sim) cleanupConfigfs() error {
	// not strictly necessary to set live=0, but it can't hurt.
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

// setupConfigfs constructs the gpio-sim configuration in configfs for the sim,
// including each of the simulated chips.
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

// builder contains all the information required to build a sim.
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

// live build creates the gpio-sim configuration for the sim and takes it live.
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
		devPath := path.Join("/dev", chipName)
		stat, err := os.Lstat(devPath)
		if err != nil {
			return nil, err
		}
		if stat.Mode()&fs.ModeSymlink != 0 {
			err = errors.New("A symlink (" + devPath + ") is masking GPIO device " + chipName)
			return nil, err
		}
		s.Chips[i].devPath = devPath
		s.Chips[i].sysfsPath = path.Join("/sys/devices/platform", devName, chipName)
	}
	return &s, nil
}

// configfsMountPoint finds the location where configfs is mounted in the file system.
//
// If no mountpoint is found, attempts to mount it in the usual "/sys/kernel/config".
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

// findConfigfsPath finds the location of gpio-sim in theconfigfs.
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

var simCounter uint32 = 0

// uniqueName returns a name for the sim that is very likely to be unique, using the
// appname, PID and a monotonic atomic counter.
//
// The only reason it may clash with an existing sim is if the user goes out of
// their way to explicitly create a sim with the same name.
func uniqueName() string {
	return fmt.Sprintf("%s-p%d-%d", appName(), os.Getpid(), atomic.AddUint32(&simCounter, 1))
}

// appName returns the name of the running execuable.
//
// Fallsback to "gpiosim" is htat can't be determined for some reason.
func appName() string {
	str, err := os.Executable()
	if err != nil {
		return "gpiosim"
	}
	return path.Base(str)
}

// hogDirectionToString maps the HogDirection to the corresponding string
// used when configuring the gpio-sim.
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
