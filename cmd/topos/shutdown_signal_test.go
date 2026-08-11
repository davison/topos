// shutdown_signal_test.go is the regression gate for the orphaned-plugin
// bug: `topos serve` used to end in a bare `http.ListenAndServe`, so a
// SIGINT or SIGTERM killed the kernel with the Go runtime's default
// disposition and the deferred `sup.Shutdown()` / `linkStore.Shutdown()`
// teardown never ran. hashicorp/go-plugin children do not die with their
// parent — the plugin side explicitly swallows SIGINT ("Eat the
// interrupts", go-plugin server.go) and has no parent-death watchdog — so
// every plugin subprocess was left ALIVE, reparented to init.
//
// The bug hid because the most-travelled exit path masked it: `make dev`'s
// `trap 'kill 0' INT TERM` turns Ctrl-C into a PROCESS-GROUP SIGTERM,
// which go-plugin does not swallow, so the children died on their own
// signal rather than by any cleanup in the kernel. That is why this test
// is a 2x2 matrix over {SIGINT, SIGTERM} x {process-group, kernel-pid}
// rather than a single case: three of those four rows orphaned before the
// fix, and the one that passed (group SIGTERM) passed for the WRONG
// REASON. A single-row test written against the masked path would have
// gone green against the unfixed kernel.
//
// The assertions deliberately distinguish a live orphan from a zombie:
// the reported symptom was live processes, not un-reaped exit statuses.
//
// Linux-only: child discovery reads /proc. This project targets a Linux
// desktop (topos/CLAUDE.md).
package main

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"
)

// signalTestBins builds the kernel and the mock plugin ONCE for every case
// in this file — each `go build` costs seconds, and the 2x2 matrix would
// otherwise pay for it four times.
var (
	signalBinsOnce sync.Once
	signalKernel   string // path to the built topos binary
	signalPlugins  string // dir containing topos-plugin-mock
	signalBinsErr  error
)

func buildSignalTestBins(t *testing.T) (kernel, pluginsDir string) {
	t.Helper()
	signalBinsOnce.Do(func() {
		out, err := exec.Command("go", "list", "-m", "-f", "{{.Dir}}", "github.com/davison/topos").Output()
		if err != nil {
			signalBinsErr = fmt.Errorf("resolve module root: %w", err)
			return
		}
		root := strings.TrimSpace(string(out))

		dir, err := os.MkdirTemp("", "topos-shutdown-signal-test-*")
		if err != nil {
			signalBinsErr = err
			return
		}
		plugins := filepath.Join(dir, "plugins")
		if err := os.MkdirAll(plugins, 0o755); err != nil {
			signalBinsErr = err
			return
		}

		build := func(out, pkg string) error {
			cmd := exec.Command("go", "build", "-o", out, pkg)
			cmd.Dir = root
			if b, err := cmd.CombinedOutput(); err != nil {
				return fmt.Errorf("build %s: %w\n%s", pkg, err, b)
			}
			return nil
		}
		kbin := filepath.Join(dir, "topos")
		if err := build(kbin, "./cmd/topos"); err != nil {
			signalBinsErr = err
			return
		}
		if err := build(filepath.Join(plugins, "topos-plugin-mock"), "./plugins/mock"); err != nil {
			signalBinsErr = err
			return
		}
		signalKernel, signalPlugins = kbin, plugins
	})
	if signalBinsErr != nil {
		t.Fatalf("build signal-test fixtures: %v", signalBinsErr)
	}
	return signalKernel, signalPlugins
}

// freePort returns a port nothing is listening on. There is an inherent
// race between closing this listener and the kernel binding it, which is
// why the caller treats a failure to reach "serving" as a fatal test
// error rather than a bug report.
func freePort(t *testing.T) int {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("reserve port: %v", err)
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port
}

// procChildren returns the PIDs of pid's direct children, as the union
// over EVERY thread's /proc/<pid>/task/<tid>/children.
//
// The union is load-bearing, not defensive. That file lists the children
// of the individual THREAD, not of the process, and the Go runtime
// fork/execs from whichever OS thread happens to be running the
// goroutine that calls exec. Reading only task/<pid>/children (the main
// thread) therefore finds the plugin subprocess or misses it depending
// purely on the scheduler — which made an earlier version of this test
// fail on a different, randomly-chosen row of the matrix each run.
func procChildren(t *testing.T, pid int) []int {
	t.Helper()
	tasks, err := filepath.Glob(fmt.Sprintf("/proc/%d/task/*/children", pid))
	if err != nil || len(tasks) == 0 {
		return nil
	}
	seen := map[int]bool{}
	var out []int
	for _, task := range tasks {
		raw, err := os.ReadFile(task)
		if err != nil {
			continue // thread exited between Glob and read
		}
		for _, f := range strings.Fields(string(raw)) {
			var c int
			if _, err := fmt.Sscanf(f, "%d", &c); err == nil && !seen[c] {
				seen[c] = true
				out = append(out, c)
			}
		}
	}
	return out
}

// procState returns the single-letter process state from /proc/<pid>/stat
// ("R", "S", "Z", ...), the process's current parent pid, and whether the
// process exists at all. The comm field can itself contain spaces and
// parentheses, so both fields are parsed relative to the LAST ')' rather
// than by splitting the whole line on whitespace.
func procState(pid int) (state string, ppid int, exists bool) {
	raw, err := os.ReadFile(fmt.Sprintf("/proc/%d/stat", pid))
	if err != nil {
		return "", 0, false
	}
	i := strings.LastIndex(string(raw), ")")
	if i < 0 {
		return "", 0, false
	}
	fields := strings.Fields(string(raw)[i+1:])
	if len(fields) < 2 {
		return "", 0, false
	}
	if _, err := fmt.Sscanf(fields[1], "%d", &ppid); err != nil {
		return fields[0], 0, true
	}
	return fields[0], ppid, true
}

func writeSignalTestConfig(t *testing.T, dir, pluginsDir string, port int) {
	t.Helper()
	cfgDir := filepath.Join(dir, "topos")
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		t.Fatalf("mkdir config dir: %v", err)
	}
	// base_url/token are non-empty placeholders only because
	// config.Validate requires a connection field; topos-plugin-mock
	// ignores both.
	cfg := fmt.Sprintf(`
[server]
listen = "127.0.0.1:%d"

[index]
path = %q

[plugins]
dir = %q

[sources.mock1]
plugin = "topos-plugin-mock"
base_url = "placeholder"
token = "placeholder"

[webspaces.regression]
keywords = ["alpha"]
`, port, filepath.Join(dir, "index.db"), pluginsDir)
	if err := os.WriteFile(filepath.Join(cfgDir, "config.toml"), []byte(cfg), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
}

// TestServeReapsPluginSubprocessesOnShutdownSignal is the regression gate.
// Each row starts a real kernel with a real go-plugin subprocess, signals
// it the way an operator or a service manager would, and asserts no plugin
// subprocess is left alive.
func TestServeReapsPluginSubprocessesOnShutdownSignal(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("child discovery reads /proc — this project targets a Linux desktop")
	}

	cases := []struct {
		name string
		sig  syscall.Signal
		// toGroup sends the signal to the whole process group (what a tty
		// does on Ctrl-C, and what `make dev`'s `trap 'kill 0'` does)
		// rather than to the kernel pid alone (`kill <pid>`, pkill, a
		// service-manager stop).
		toGroup bool
	}{
		// The row an operator hits with a bare `topos serve` + Ctrl-C, and
		// the single most important row: the plugin child receives this
		// SIGINT too and DELIBERATELY IGNORES IT, so only the kernel's own
		// teardown can reap it.
		{name: "SIGINT to process group (terminal Ctrl-C)", sig: syscall.SIGINT, toGroup: true},
		// `kill -INT <pid>` — child never sees the signal at all.
		{name: "SIGINT to kernel pid", sig: syscall.SIGINT, toGroup: false},
		// The masked row: passed even before the fix, because the child
		// dies of its own group SIGTERM. Kept so a future change that
		// re-breaks teardown is still caught here if it also stops the
		// child dying on its own.
		{name: "SIGTERM to process group (make dev trap)", sig: syscall.SIGTERM, toGroup: true},
		// `kill <pid>` / pkill / a service-manager stop.
		{name: "SIGTERM to kernel pid", sig: syscall.SIGTERM, toGroup: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			kernelBin, pluginsDir := buildSignalTestBins(t)
			home := t.TempDir()
			port := freePort(t)
			writeSignalTestConfig(t, home, pluginsDir, port)

			cmd := exec.Command(kernelBin, "serve")
			// Setpgid puts the kernel in its OWN process group so the
			// group-directed rows below cannot signal the test runner.
			cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
			// Explicit env allowlist — never the developer's own
			// XDG_CONFIG_HOME, which would load the operator's real config.
			cmd.Env = []string{
				"PATH=" + os.Getenv("PATH"),
				"HOME=" + home,
				"XDG_CONFIG_HOME=" + home,
				"XDG_DATA_HOME=" + home,
				"XDG_CACHE_HOME=" + home,
			}
			// Capture to a real file, never a strings.Builder: os/exec
			// hands an *os.File straight to the child as its fd, so there
			// is no copier goroutine racing this test's reads of the
			// captured output (a strings.Builder here is a genuine data
			// race, and loses output when read during a timeout).
			logPath := filepath.Join(home, "kernel.log")
			logFile, err := os.Create(logPath)
			if err != nil {
				t.Fatalf("create kernel log: %v", err)
			}
			defer logFile.Close()
			kernelLog := func() string {
				b, err := os.ReadFile(logPath)
				if err != nil {
					return fmt.Sprintf("<could not read kernel log: %v>", err)
				}
				if len(b) == 0 {
					return "<kernel produced no output>"
				}
				return string(b)
			}
			cmd.Stdout, cmd.Stderr = logFile, logFile
			if err := cmd.Start(); err != nil {
				t.Fatalf("start kernel: %v", err)
			}
			kpid := cmd.Process.Pid

			waited := make(chan error, 1)
			go func() { waited <- cmd.Wait() }()

			// Belt-and-braces cleanup: if any assertion below fails early,
			// nothing is left running for the next test (or the developer).
			defer func() {
				_ = syscall.Kill(-kpid, syscall.SIGKILL)
				_ = syscall.Kill(kpid, syscall.SIGKILL)
			}()

			// Precondition: wait for a real plugin subprocess to exist.
			// Without this the test could pass vacuously by signalling a
			// kernel that had not launched anything yet.
			var children []int
			deadline := time.Now().Add(60 * time.Second)
			for time.Now().Before(deadline) {
				if children = procChildren(t, kpid); len(children) > 0 {
					break
				}
				time.Sleep(50 * time.Millisecond)
			}
			if len(children) == 0 {
				t.Fatalf("kernel launched no plugin subprocess within the timeout; kernel output:\n%s", kernelLog())
			}

			if tc.toGroup {
				if err := syscall.Kill(-kpid, tc.sig); err != nil {
					t.Fatalf("signal process group: %v", err)
				}
			} else {
				if err := syscall.Kill(kpid, tc.sig); err != nil {
					t.Fatalf("signal kernel: %v", err)
				}
			}

			select {
			case <-waited:
			case <-time.After(serverShutdownTimeout + 20*time.Second):
				t.Fatalf("kernel did not exit after %v; kernel output:\n%s", tc.sig, kernelLog())
			}

			// The kernel is gone. Every child it launched must be gone too.
			// A zombie ("Z") is acceptable — it has terminated and is merely
			// awaiting reap by init. A live state (R/S/D) is the bug.
			for _, child := range children {
				state, ppid, exists := procState(child)
				if !exists || state == "Z" {
					continue
				}
				// A reparented ppid (1, or a subreaper) is the signature of
				// the original report: a LIVE process outliving the kernel.
				t.Errorf("plugin subprocess %d survived kernel exit in state %q, reparented to ppid %d — orphaned; kernel output:\n%s",
					child, state, ppid, kernelLog())
			}
		})
	}
}
