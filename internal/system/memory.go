package system

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

type Memory struct {
	TotalKB     uint64
	AvailableKB uint64
}

func ReadMemory() (Memory, error) {
	file, err := os.Open("/proc/meminfo")
	if err != nil {
		return Memory{}, fmt.Errorf("failed to open /proc/meminfo: %v", err)
	}
	defer file.Close()

	var memory Memory
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		// separate the string into slices using whitespace as the delimiter
		fields := strings.Fields(scanner.Text())
		if len(fields) < 2 {
			continue
		}

		switch strings.TrimSuffix(fields[0], ":") {
		case "MemTotal", "MemAvailable":
			// matches the pattern "MemTotal: 16384256 kB" or "MemAvailable: 1234567 kB"
			value, err := strconv.ParseUint(fields[1], 10, 64)
			if err != nil {
				return Memory{}, fmt.Errorf("failed to parse memory value: %v", err)
			}
			switch strings.TrimSuffix(fields[0], ":") {
			case "MemTotal":
				memory.TotalKB = value
			case "MemAvailable":
				memory.AvailableKB = value
			}
		}

		// check for errors during scanning
		if err := scanner.Err(); err != nil {
			return Memory{}, fmt.Errorf("error reading /proc/meminfo: %v", err)
		}

		// break when you've both the values greater than 0
		if memory.TotalKB > 0 && memory.AvailableKB > 0 {
			break
		}

	}
	if memory.TotalKB == 0 || memory.AvailableKB == 0 {
		return Memory{}, fmt.Errorf("failed to find memory values in /proc/meminfo")
	}

	return memory, nil
}
