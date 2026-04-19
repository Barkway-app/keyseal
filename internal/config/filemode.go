package config

import (
	"fmt"
	"strconv"
)

func parseFileMode(value string) (uint32, error) {
	if len(value) != 4 || value[0] != '0' {
		return 0, fmt.Errorf("must be a 4-digit octal string like 0600")
	}
	mode, err := strconv.ParseUint(value, 8, 32)
	if err != nil {
		return 0, fmt.Errorf("invalid octal mode %q", value)
	}
	return uint32(mode), nil
}
