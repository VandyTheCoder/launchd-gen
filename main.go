// launchd-gen converts a cron expression into a macOS launchd plist.
//
// Usage:
//
//	launchd-gen [flags] <cron-expression> <command> [args...]
//
// The generated plist is written to stdout by default, or to
// ~/Library/LaunchAgents/<label>.plist when --install is set.
package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/VandyTheCoder/launchd-gen/internal/cron"
	"github.com/VandyTheCoder/launchd-gen/internal/plist"
)

const usage = `launchd-gen — convert cron expressions into macOS launchd plists

Usage:
  launchd-gen [flags] <cron-expression> <command> [args...]

Flags:
  --label        string   Required. Reverse-DNS label for the job (e.g. com.lucifer.news)
  --workdir      string   WorkingDirectory for the job
  --stdout       string   Path to capture stdout
  --stderr       string   Path to capture stderr
  --env          string   Environment var in KEY=VALUE form (repeatable)
  --install              Write plist to ~/Library/LaunchAgents/<label>.plist
  --load                 After --install, run 'launchctl load' on the plist
  -h, --help             Show this help

Examples:
  # Print a plist to stdout
  launchd-gen --label com.lucifer.daily "0 9 * * 1-5" /usr/bin/python3 /path/to/script.py

  # Install and load in one shot
  launchd-gen --label com.lucifer.news --install --load \
      --stdout /tmp/news.log --stderr /tmp/news.err \
      "57 9 * * *" /usr/local/bin/claude -p "fetch news"
`

type envFlag []string

func (e *envFlag) String() string     { return strings.Join(*e, ",") }
func (e *envFlag) Set(v string) error { *e = append(*e, v); return nil }

func main() {
	fs := flag.NewFlagSet("launchd-gen", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	fs.Usage = func() { fmt.Fprint(os.Stderr, usage) }

	var (
		label   = fs.String("label", "", "reverse-DNS label for the job")
		workdir = fs.String("workdir", "", "WorkingDirectory for the job")
		stdout  = fs.String("stdout", "", "path to capture stdout")
		stderr  = fs.String("stderr", "", "path to capture stderr")
		install = fs.Bool("install", false, "write plist to ~/Library/LaunchAgents")
		load    = fs.Bool("load", false, "after --install, also launchctl load the plist")
	)
	var envs envFlag
	fs.Var(&envs, "env", "environment var KEY=VALUE (repeatable)")

	if err := fs.Parse(os.Args[1:]); err != nil {
		os.Exit(2)
	}

	args := fs.Args()
	if len(args) < 2 {
		fmt.Fprint(os.Stderr, usage)
		os.Exit(2)
	}
	if *label == "" {
		fatal("--label is required")
	}

	expr, program := args[0], args[1:]

	schedule, err := cron.Parse(expr)
	if err != nil {
		fatal("invalid cron expression: %v", err)
	}

	envMap, err := parseEnv(envs)
	if err != nil {
		fatal("%v", err)
	}

	job := plist.Job{
		Label:            *label,
		ProgramArguments: program,
		WorkingDirectory: *workdir,
		StandardOutPath:  *stdout,
		StandardErrPath:  *stderr,
		EnvironmentVars:  envMap,
		Schedule:         schedule,
	}

	if *install {
		path, err := installPath(*label)
		if err != nil {
			fatal("%v", err)
		}
		f, err := os.Create(path)
		if err != nil {
			fatal("cannot create %s: %v", path, err)
		}
		if err := plist.Write(f, job); err != nil {
			f.Close()
			fatal("write plist: %v", err)
		}
		if err := f.Close(); err != nil {
			fatal("close plist: %v", err)
		}
		fmt.Fprintf(os.Stderr, "wrote %s\n", path)
		if *load {
			if err := launchctlLoad(path); err != nil {
				fatal("launchctl load: %v", err)
			}
			fmt.Fprintf(os.Stderr, "loaded %s\n", *label)
		}
		return
	}

	if err := plist.Write(os.Stdout, job); err != nil {
		fatal("write plist: %v", err)
	}
}

func parseEnv(raw []string) (map[string]string, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	m := make(map[string]string, len(raw))
	for _, pair := range raw {
		idx := strings.Index(pair, "=")
		if idx <= 0 {
			return nil, fmt.Errorf("invalid --env %q, expected KEY=VALUE", pair)
		}
		m[pair[:idx]] = pair[idx+1:]
	}
	return m, nil
}

func installPath(label string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("home dir: %w", err)
	}
	dir := filepath.Join(home, "Library", "LaunchAgents")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("mkdir %s: %w", dir, err)
	}
	return filepath.Join(dir, label+".plist"), nil
}

func launchctlLoad(path string) error {
	cmd := exec.Command("launchctl", "load", path)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func fatal(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "launchd-gen: "+format+"\n", args...)
	os.Exit(1)
}
