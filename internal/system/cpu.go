package system

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

type CPUStats struct {
	Total uint64
	Idle  uint64
}

// ReadCPUStats reads aggregate cpu counters from /proc/stat

// cpu aggregate data -------------->

// user nice system idle iowait irq softirq steal guest guest_nice

// cpu  1736813 8057 1969779 55050690 27008 0 28972 0 0 0
// cpu0 302682 1127 380675 4177354 2518 0 13752 0 0 0
// cpu1 194112 827 124154 4569400 1826 0 5286 0 0 0
// cpu2 361679 1347 455207 4052152 3184 0 2679 0 0 0
// cpu3 171639 883 107975 4619964 2319 0 1156 0 0 0
// cpu4 144274 556 149392 4602761 2590 0 1272 0 0 0

func ReadCPUStats() (CPUStats, error) {
	file, err := os.Open("/proc/stat")
	if err != nil {
		return CPUStats{}, fmt.Errorf("Open /proc/stat:%w", err)
	}

	defer file.Close()

	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())

		if len(fields) == 0 || fields[0] != "cpu" {
			continue
		}

		if len(fields) < 5 {
			return CPUStats{}, fmt.Errorf("Incomplete CPU Stats")
		}

		var stats CPUStats

		for i, field := range fields[1:] {
			val, err := strconv.ParseUint(field, 10, 64)

			if err != nil {
				return CPUStats{}, fmt.Errorf("parse CPU counter %q: %w", field, err)
			}

			// we are considering idle and iowait into Idle
			// total = user + nice + system + idle + iowait + irq + softirq + steal
			// Guest and guest_nice (i >= 8) are already counted inside user and nice
			if i < 8 {
				stats.Total += val
			}
			if i == 3 || i == 4 {
				stats.Idle += val
			}
		}
		return stats, nil
	}

	if err := scanner.Err(); err != nil {
		return CPUStats{}, fmt.Errorf("read /proc/stat: %w", err)
	}

	return CPUStats{}, fmt.Errorf("Aggregate CPU stats not found")
}

func CPUUsage(previous, current CPUStats) (float64, error) {
	if current.Total <= previous.Total {
		return 0, fmt.Errorf("CPU total counter didn't increase")
	}

	if current.Idle < previous.Idle {
		return 0, fmt.Errorf("CPU Idle counter decreased")
	}

	totalDelta := current.Total - previous.Total
	idleDelta := current.Idle - previous.Idle

	if idleDelta > totalDelta {
		return 0, fmt.Errorf("idle delta exceeds total delta")
	}

	busyDelta := totalDelta - idleDelta
	return float64(busyDelta) / float64(totalDelta) * 100, nil
}
