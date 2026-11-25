// Command startup-time sets or shows the time before which the machine
// refuses to finish booting (see startup-time-check and its systemd unit).
//
// It is installed setuid root so that an unprivileged user can lock the
// machine. Because of that it trusts nothing from the caller's environment:
// the time is parsed by /usr/bin/date (absolute path, minimal environment).
package main

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// Overridable at build time (-ldflags "-X main.stateFile=...") for tests;
// deliberately not configurable at run time because the binary is setuid.
var (
	stateFile = "/var/lib/startup-time.state"
	dateCmd   = "/usr/bin/date"
)

func usage() {
	fmt.Fprintln(os.Stderr, `Usage:
  startup-time set <time>   lock booting until <time> (any format date(1) accepts,
                            e.g. "tomorrow 07:00" or "2026-12-24 18:00")
  startup-time get          show the current lock
  startup-time clear        remove the lock`)
}

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}

	var err error
	switch os.Args[1] {
	case "set":
		if len(os.Args) < 3 {
			usage()
			os.Exit(2)
		}
		err = set(strings.Join(os.Args[2:], " "))
	case "get":
		err = get()
	case "clear":
		err = clear()
	case "-h", "--help", "help":
		usage()
	default:
		fmt.Fprintln(os.Stderr, "Unknown command:", os.Args[1])
		usage()
		os.Exit(2)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}

// parseTime lets date(1) interpret the free-form input. The child gets an
// empty environment apart from a fixed locale, so PATH, TZ and similar
// variables from an unprivileged caller cannot influence a root process.
func parseTime(input string) (time.Time, error) {
	cmd := exec.Command(dateCmd, "--date="+input, "+%s")
	cmd.Env = []string{"LC_ALL=C"}
	out, err := cmd.Output()
	if err != nil {
		var ee *exec.ExitError
		if errors.As(err, &ee) && len(ee.Stderr) > 0 {
			return time.Time{}, fmt.Errorf("cannot parse %q: %s", input, strings.TrimSpace(string(ee.Stderr)))
		}
		return time.Time{}, fmt.Errorf("cannot parse %q: %v", input, err)
	}
	var secs int64
	if _, err := fmt.Sscan(strings.TrimSpace(string(out)), &secs); err != nil {
		return time.Time{}, fmt.Errorf("unexpected output from date: %q", out)
	}
	return time.Unix(secs, 0), nil
}

// readLock returns the stored lock time, or the zero time if there is none.
func readLock() (time.Time, error) {
	data, err := os.ReadFile(stateFile)
	if errors.Is(err, fs.ErrNotExist) {
		return time.Time{}, nil
	}
	if err != nil {
		return time.Time{}, err
	}
	t, err := time.Parse(time.RFC3339, strings.TrimSpace(string(data)))
	if err != nil {
		return time.Time{}, fmt.Errorf("corrupt state file %s: %v", stateFile, err)
	}
	return t, nil
}

func set(input string) error {
	t, err := parseTime(input)
	if err != nil {
		return err
	}
	if err := writeAtomic(t.Local().Format(time.RFC3339) + "\n"); err != nil {
		return err
	}
	fmt.Println("Set startup time to", t.Local().Format(time.RFC3339))
	if !t.After(time.Now()) {
		fmt.Println("Note: that time has already passed, so the next boot is not locked.")
	}
	return nil
}

func get() error {
	t, err := readLock()
	if err != nil {
		return err
	}
	switch {
	case t.IsZero():
		fmt.Println("No lock set.")
	case t.After(time.Now()):
		fmt.Printf("Boot locked until %s (%s from now).\n",
			t.Local().Format(time.RFC3339), time.Until(t).Round(time.Second))
	default:
		fmt.Printf("Lock expired at %s.\n", t.Local().Format(time.RFC3339))
	}
	return nil
}

func clear() error {
	if err := os.Remove(stateFile); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	fmt.Println("Lock cleared.")
	return nil
}

// writeAtomic replaces the state file via a temp file and rename, so a crash
// or power loss never leaves a half-written lock behind.
func writeAtomic(content string) error {
	dir := filepath.Dir(stateFile)
	tmp, err := os.CreateTemp(dir, ".startup-time.*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.WriteString(content); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Chmod(0o644); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), stateFile)
}
