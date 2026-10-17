package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sort"
	"strings"
	"syscall"
	"time"

	"github.com/Eyepatch5263/gopulse/internal/system"
	"github.com/Eyepatch5263/gopulse/internal/tui"
)

type Metrics struct {
	CPUUsage float64
	Memory   system.Memory
}

type MonitorResult struct {
	Metrics Metrics
	Err     error
}

func showWatch() {
	ctx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)

	defer stop()

	results := make(chan MonitorResult)

	go runMonitor(ctx, results, time.Second)

	fmt.Println("GoPulse - Live Monitor")
	fmt.Println("Press Ctrl + C to stop")

	fmt.Println("----------------------------------------------------")

	for res := range results {
		if res.Err != nil {
			log.Printf("Monitor Stopped : %v", res.Err)
			break
		}

		m := res.Metrics
		usedKB := m.Memory.TotalKB - m.Memory.AvailableKB
		ramPercentage := float64(usedKB) / float64(m.Memory.TotalKB) * 100

		fmt.Printf("CPU: %6.2f%% | RAM: %6.2f%% | Used: %.2f Gib\n", m.CPUUsage, ramPercentage, float64(usedKB)/(1024*1024))

	}
}

// ctx context.Context: Used to signal cancellation, timeouts, or deadlines across goroutine boundaries.
// results chan<- MonitorResult: A send-only channel that accepts values of type MonitorResult
// result MonitorResult: The payload being sent into the channel.
// bool return value: Indicates whether the send operation succeeded (true) or was aborted due to context cancellation (false).

func sendResult(
	ctx context.Context,
	results chan<- MonitorResult,
	result MonitorResult,
) bool {
	select {
	// If a receiver is ready (or buffer space is available), result is delivered to the results channel.
	case results <- result:
		return true
	// If the context is canceled (e.g., via cancel(), a timeout, or a deadline) before or while waiting to send, <-ctx.Done() unblocks.
	case <-ctx.Done():
		return false
	}
}

func runMonitor(
	ctx context.Context,
	results chan<- MonitorResult,
	interval time.Duration,
) {
	defer close(results)

	prev, err := system.ReadCPUStats()

	if err != nil {
		sendResult(ctx, results, MonitorResult{Err: err})
		return
	}

	// Creates a new ticker that delivers periodic events, unlike time.Sleep, a ticker lets the monitoring goroutine repeatedly wait for the next
	// sampling event without manually creating a new timer each iteration
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		// receive only channels
		case <-ctx.Done():
			return
		case <-ticker.C:
			curr, err := system.ReadCPUStats()
			if err != nil {
				sendResult(ctx, results, MonitorResult{Err: err})
				return
			}
			usage, err := system.CPUUsage(prev, curr)
			if err != nil {
				sendResult(ctx, results, MonitorResult{Err: err})
				return
			}

			memory, err := system.ReadMemory()
			if err != nil {
				sendResult(ctx, results, MonitorResult{Err: err})
				return
			}

			result := MonitorResult{
				Metrics: Metrics{
					CPUUsage: usage,
					Memory:   memory,
				},
			}

			if !sendResult(ctx, results, result) {
				return
			}
			prev = curr

		}
	}

}

func showMemory() {
	memory, err := system.ReadMemory()
	if err != nil {
		log.Fatalf("Error reading memory: %v", err)
	}

	usedKB := memory.TotalKB - memory.AvailableKB
	usedPercentage := float64(usedKB) / float64(memory.TotalKB) * 100

	fmt.Println("GoPulse - System Monitor")

	fmt.Println("----------------------")

	fmt.Printf("Total RAM : %d MB\n", memory.TotalKB/1024)
	fmt.Printf("Available RAM : %d MB\n", memory.AvailableKB/1024)
	fmt.Printf("Used RAM : %d MB (%.2f%%)\n", usedKB/1024, usedPercentage)
}

func showProcesses() {
	processes, err := system.ReadProcesses()
	if err != nil {
		log.Fatalf("Error reading processes: %v", err)
	}

	sort.Slice(processes, func(i, j int) bool {
		return int(processes[i].RSSKB) > int(processes[j].RSSKB)
	})

	fmt.Println("---------------------------------------------------------------")
	// Header
	fmt.Printf("%-7s %-7s %-25s %-8s %12s\n", "PID", "PPID", "NAME", "STATE", "RSS (MB)")
	fmt.Println(strings.Repeat("-", 63))

	// Rows
	for _, proc := range processes {
		// Truncate overly long process names so they don't break table alignment
		name := proc.Name
		if len(name) > 24 {
			name = name[:21] + "..."
		}

		ram := "-"

		if proc.RSSAvailable {
			ram = fmt.Sprintf("%.2f", float64(proc.RSSKB)/1024)
		}

		fmt.Printf("%-7d %-7d %-25s %-8s %12s\n",
			proc.PID,
			proc.PPid,
			name,
			proc.State,
			ram,
		)
	}
}

func showCPU() {
	prev, err := system.ReadCPUStats()

	if err != nil {
		log.Fatal(err)
	}

	time.Sleep(time.Second)

	curr, err := system.ReadCPUStats()
	if err != nil {
		log.Fatal(err)
	}

	usage, err := system.CPUUsage(prev, curr)

	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("CPU Utilization: %.2f%%\n", usage)
}

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "ps":
			showProcesses()
			return
		case "cpu":
			showCPU()
			return

		case "watch":
			showWatch()
			return

		case "tui":
			if err := tui.Run(); err != nil {
				log.Fatal(err)
			}
			return
		}

	}

	showMemory()

}
