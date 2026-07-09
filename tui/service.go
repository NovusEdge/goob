package main

import (
	"bufio"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/shirou/gopsutil/v4/process"
)

// ring is a fixed-size, newest-wins line buffer for log tailing.
type ring struct {
	mu    sync.Mutex
	lines []string
	max   int
}

func newRing(max int) *ring { return &ring{max: max} }

func (r *ring) add(s string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.lines = append(r.lines, s)
	if len(r.lines) > r.max {
		r.lines = r.lines[len(r.lines)-r.max:]
	}
}

func (r *ring) text() string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return strings.Join(r.lines, "\n")
}

// projectRoot walks up from cwd to the dir containing justfile so the TUI works
// whether launched from repo root (`just tui`) or from tui/ (`go run .`).
func projectRoot() string {
	dir, err := os.Getwd()
	if err != nil {
		return "."
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "justfile")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "."
		}
		dir = parent
	}
}

// Service supervises one `just <target>` process (and its whole tree).
type Service struct {
	name   string
	target string
	dir    string
	logs   *ring
	cpu    float64
	dbg    string // latest "goob-dbg:" line the process emitted (pet debug readout)

	mu    sync.Mutex
	cmd   *exec.Cmd
	alive bool
	procs map[int32]*process.Process // reused across samples for CPU deltas
}

func newService(name, target, dir string) *Service {
	return &Service{name: name, target: target, dir: dir,
		logs: newRing(500), procs: map[int32]*process.Process{}}
}

func (s *Service) running() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.alive
}

func (s *Service) pid() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cmd != nil && s.cmd.Process != nil {
		return s.cmd.Process.Pid
	}
	return 0
}

func (s *Service) logText() string { return s.logs.text() }

func (s *Service) debugLine() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.dbg
}

func (s *Service) start() error {
	s.mu.Lock()
	if s.alive {
		s.mu.Unlock()
		return nil
	}
	s.alive = true
	s.mu.Unlock()

	cmd := exec.Command("just", s.target)
	cmd.Dir = s.dir
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	pr, pw, err := os.Pipe()
	if err != nil {
		s.mu.Lock()
		s.alive = false
		s.mu.Unlock()
		return err
	}
	cmd.Stdout, cmd.Stderr = pw, pw
	if err := cmd.Start(); err != nil {
		pr.Close()
		pw.Close()
		s.mu.Lock()
		s.alive = false
		s.mu.Unlock()
		return err
	}
	pw.Close() // parent's copy; the child holds its own fds

	s.mu.Lock()
	s.cmd = cmd
	s.procs = map[int32]*process.Process{}
	s.mu.Unlock()

	go func() {
		sc := bufio.NewScanner(pr)
		sc.Buffer(make([]byte, 64*1024), 1<<20)
		for sc.Scan() {
			line := sc.Text()
			s.logs.add(line)
			if strings.HasPrefix(line, "goob-dbg:") {
				s.mu.Lock()
				s.dbg = line
				s.mu.Unlock()
			}
		}
		pr.Close()
	}()
	go func() {
		cmd.Wait() // reap; then mark stopped
		s.mu.Lock()
		s.alive = false
		s.cpu = 0
		s.mu.Unlock()
	}()
	return nil
}

func (s *Service) stop() {
	s.mu.Lock()
	cmd, alive := s.cmd, s.alive
	s.mu.Unlock()
	if !alive || cmd == nil || cmd.Process == nil {
		return
	}
	pgid := cmd.Process.Pid // Setpgid leader: pgid == pid
	syscall.Kill(-pgid, syscall.SIGTERM)
	go func() {
		time.Sleep(3 * time.Second)
		s.mu.Lock()
		still := s.alive && s.cmd == cmd
		s.mu.Unlock()
		if still {
			syscall.Kill(-pgid, syscall.SIGKILL)
		}
	}()
}

// sampleCPU sums CPU% across the process tree. gopsutil Percent(0) returns the
// delta since the previous call on the SAME Process object, so we cache objects
// in s.procs and reuse them. First sample after start reads ~0.
func (s *Service) sampleCPU() float64 {
	root := s.pid()
	if !s.running() || root == 0 {
		s.mu.Lock()
		s.cpu = 0
		s.mu.Unlock()
		return 0
	}
	pids := treePIDs(int32(root))
	sum := 0.0
	seen := map[int32]bool{}

	s.mu.Lock()
	cache := s.procs
	for _, pid := range pids {
		seen[pid] = true
		p := cache[pid]
		if p == nil {
			np, err := process.NewProcess(pid)
			if err != nil {
				continue
			}
			p = np
			cache[pid] = p
		}
		if v, err := p.Percent(0); err == nil {
			sum += v
		}
	}
	for pid := range cache {
		if !seen[pid] {
			delete(cache, pid)
		}
	}
	s.cpu = sum
	s.mu.Unlock()
	return sum
}

// treePIDs returns pid and all descendants (ppid tree).
func treePIDs(pid int32) []int32 {
	out := []int32{pid}
	p, err := process.NewProcess(pid)
	if err != nil {
		return out
	}
	kids, err := p.Children()
	if err != nil {
		return out
	}
	for _, k := range kids {
		out = append(out, treePIDs(k.Pid)...)
	}
	return out
}
