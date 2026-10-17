package system

import (
	"math"
	"testing"
)

func TestCPUUsage(t *testing.T) {
	previous := CPUStats{
		Total: 8300,
		Idle:  7000,
	}

	current := CPUStats{
		Total: 9000,
		Idle:  7450,
	}

	got, err := CPUUsage(previous, current)

	if err != nil {
		t.Errorf("CPUUsage() error =%v", err)
	}

	want := (250.0 / 700.0) * 100

	if math.Abs(got-want) > 0.001 {
		t.Errorf("CPUUsage() = %.3f, want %.3f", got, want)
	}
}

func TestCPUUsageNoIncrease(t *testing.T) {
	previous := CPUStats{Total: 100, Idle: 50}
	current := CPUStats{Total: 100, Idle: 50}

	if _, err := CPUUsage(previous, current); err == nil {
		t.Fatalf("expected error when total counter does not increase")
	}
}

func TestCPUUsageIdleExceedsTotal(t *testing.T) {
	previous := CPUStats{Total: 100, Idle: 50}
	current := CPUStats{Total: 200, Idle: 160}

	if _, err := CPUUsage(previous, current); err == nil {
		t.Fatalf("expected error when idle delta exceeds total delta")
	}
}
