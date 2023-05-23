// SPDX-FileCopyrightText: 2023 Kent Gibson <warthog618@gmail.com>
//
// SPDX-License-Identifier: Apache-2.0 OR MIT

package gpiosim

import (
	"os"
	"path"
	"strings"
)

// Helper functions to read and write attributes to configfs and sysfs files.

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
