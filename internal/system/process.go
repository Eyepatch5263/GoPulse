package system

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type Process struct {
	PID          int
	PPid         int
	Name         string
	State        string
	RSSKB        uint64
	RSSAvailable bool
	Cmdline      string
}

// cat /proc/8501/status
// --------------------------------- structure of /proc/[pid]/status ---------------------------------

// Name:	dbus-daemon
// Umask:	0022
// State:	S (sleeping)
// Tgid:	8501
// Ngid:	0
// Pid:	8501
// PPid:	1
// TracerPid:	0
// Uid:	995	995	995	995
// Gid:	995	995	995	995
// FDSize:	512
// Groups:	995
// NStgid:	8501
// NSpid:	8501
// NSpgid:	8501
// NSsid:	8501
// Kthread:	0
// VmPeak:	   14480 kB
// VmSize:	   14480 kB
// VmLck:	       0 kB
// VmPin:	       0 kB
// VmHWM:	    9748 kB
// VmRSS:	    9748 kB
// RssAnon:	    5056 kB
// RssFile:	    4692 kB
// RssShmem:	       0 kB
// VmData:	    7320 kB
// VmStk:	     132 kB
// VmExe:	     148 kB
// VmLib:	    4572 kB
// VmPTE:	      72 kB
// VmSwap:	       0 kB
// HugetlbPages:	       0 kB
// CoreDumping:	0
// THP_enabled:	1
// untag_mask:	0xffffffffffffffff
// Threads:	1
// SigQ:	0/47684
// SigPnd:	0000000000000000
// ShdPnd:	0000000000000000
// SigBlk:	0000000000000000
// SigIgn:	0000000000001000
// SigCgt:	0000000000004001
// CapInh:	0000000020000000
// CapPrm:	0000000020000000
// CapEff:	0000000020000000
// CapBnd:	000001ffffffffff
// CapAmb:	0000000020000000
// NoNewPrivs:	0
// Seccomp:	0
// Seccomp_filters:	0
// Speculation_Store_Bypass:	thread vulnerable
// SpeculationIndirectBranch:	conditional enabled
// Cpus_allowed:	fff
// Cpus_allowed_list:	0-11
// Mems_allowed:	00000000,00000000,00000000,00000000,00000000,00000000,00000000,00000000,00000000,00000000,00000000,00000000,00000000,00000000,00000000,00000000,00000000,00000000,00000000,00000000,00000000,00000000,00000000,00000000,00000000,00000000,00000000,00000000,00000000,00000000,00000000,00000001
// Mems_allowed_list:	0
// voluntary_ctxt_switches:	315709
// nonvoluntary_ctxt_switches:	10196
// x86_Thread_features:
// x86_Thread_features_locked:

func readCmdlineAt(root string, pid int) string {
	path := filepath.Join(root, strconv.Itoa(pid), "cmdline")

	data, err := os.ReadFile(path)
	if err != nil || len(data) == 0 {
		return ""
	}

	return strings.TrimSpace(
		strings.ReplaceAll(string(data), "\x00", ""),
	)
}
func parseStatus(r io.Reader, pid int) (Process, error) {
	proc := Process{PID: pid}

	var hasName, hasState, hasPPid bool
	scanner := bufio.NewScanner(r)

	for scanner.Scan() {
		// separate the string into slices using ":" as the delimiter and gives key and value
		key, value, ok := strings.Cut(scanner.Text(), ":")

		if !ok {
			continue
		}

		fields := strings.Fields(value)
		key = strings.TrimSpace(key)

		switch key {
		case "PPid":
			if len(fields) > 0 {
				ppid, err := strconv.Atoi(fields[0])
				if err != nil {
					return Process{}, fmt.Errorf("failed to parse PPid value: %v", err)
				}
				proc.PPid = ppid
				hasPPid = true

			}
		case "Name":
			if len(fields) > 0 {
				proc.Name = strings.Join(fields, " ")
				hasName = true
			}
		case "State":
			if len(fields) > 0 {
				proc.State = strings.TrimSpace(value)
				hasState = true
			}

		case "VmRSS":
			if len(fields) < 2 || fields[1] != "kB" {
				return Process{}, fmt.Errorf("unexpected VmRSS format: %v", fields)
			}

			rss, err := strconv.ParseUint(fields[0], 10, 64)
			if err != nil {
				return Process{}, fmt.Errorf("failed to parse VmRSS value: %v", err)
			}
			proc.RSSKB = rss
			proc.RSSAvailable = true

		}
		// don't read the entire file if we have all the values we need
		if hasName && hasState && hasPPid && proc.RSSAvailable {
			break
		}
	}

	if !hasName || !hasState || !hasPPid {
		return Process{}, fmt.Errorf("missing required fields in process %d: Name=%v, State=%v PPid=%v", pid, hasName, hasState, hasPPid)
	}
	if err := scanner.Err(); err != nil {
		return Process{}, fmt.Errorf("error reading process %d: %v", pid, err)
	}

	return proc, nil
}

func ReadProcess(pid int) (Process, error) {
	return ReadProcessAt("/proc", pid)
}

func ReadProcessAt(root string, pid int) (Process, error) {
	path := filepath.Join(
		root,
		strconv.Itoa(pid),
		"status",
	)

	file, err := os.Open(path)

	if err != nil {
		return Process{}, fmt.Errorf("failed to open %s: %w", path, err)
	}
	defer file.Close()

	process, err := parseStatus(file, pid)
	if err != nil {
		return Process{}, nil
	}
	return process, nil
}

func ReadProcesses() ([]Process, error) {
	return ReadProcessesFrom("/proc")
}

func ReadProcessesFrom(root string) ([]Process, error) {
	// root = /proc
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, fmt.Errorf("failed to read %s directory: %v", root, err)
	}
	// 0 -> length and len(entries) -> capacity of the slice
	processes := make([]Process, 0, len(entries))

	for _, entry := range entries {
		pid, err := strconv.Atoi(entry.Name())
		if err != nil {
			continue // skip non-numeric entries
		}

		proc, err := ReadProcess(pid)
		if err != nil {

			if errors.Is(err, os.ErrNotExist) || errors.Is(err, os.ErrPermission) {
				continue // skip processes that have exited or are inaccessible
			}
			return nil, fmt.Errorf("failed to read process %d: %v", pid, err)
		}
		// it is used to increase the length of the slice by one and add the new element to the end of the slice.
		// It is used to build a list of processes by appending each process to the slice as it is read from the /proc directory.
		proc.Cmdline = readCmdlineAt(root, pid)
		processes = append(processes, proc)
	}
	return processes, nil
}
