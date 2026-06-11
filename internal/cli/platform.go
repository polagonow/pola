package cli

import (
	"fmt"
	"runtime"
	"strings"
)

type platformTarget struct {
	GOOS   string
	GOARCH string
}

var supportedTargets = []platformTarget{
	{GOOS: "darwin", GOARCH: "arm64"},
	{GOOS: "darwin", GOARCH: "amd64"},
	{GOOS: "linux", GOARCH: "arm64"},
	{GOOS: "linux", GOARCH: "amd64"},
	{GOOS: "windows", GOARCH: "arm64"},
	{GOOS: "windows", GOARCH: "amd64"},
}

var supportedOS = map[string]bool{"darwin": true, "linux": true, "windows": true}
var supportedArch = map[string]bool{"arm64": true, "amd64": true}

func (t platformTarget) label() string {
	return t.GOOS + "-" + t.GOARCH
}

func (t platformTarget) binName(base string) string {
	if t.GOOS == "windows" && !strings.HasSuffix(base, ".exe") {
		return base + ".exe"
	}
	return base
}

func (t platformTarget) isHost() bool {
	return t.GOOS == runtime.GOOS && t.GOARCH == runtime.GOARCH
}

func parsePlatforms(args []string) ([]platformTarget, error) {
	if len(args) == 0 {
		return nil, nil
	}

	seen := make(map[platformTarget]bool)
	var targets []platformTarget
	add := func(t platformTarget) {
		if !seen[t] {
			seen[t] = true
			targets = append(targets, t)
		}
	}

	for _, arg := range args {
		switch {
		case arg == "all":
			for _, t := range supportedTargets {
				add(t)
			}
		case supportedOS[arg]:
			for _, t := range supportedTargets {
				if t.GOOS == arg {
					add(t)
				}
			}
		case strings.Contains(arg, "/"):
			parts := strings.SplitN(arg, "/", 2)
			goos, goarch := parts[0], parts[1]
			if !supportedOS[goos] {
				return nil, fmt.Errorf("unsupported OS %q; supported: darwin, linux, windows", goos)
			}
			if !supportedArch[goarch] {
				return nil, fmt.Errorf("unsupported architecture %q; supported: arm64, amd64", goarch)
			}
			add(platformTarget{GOOS: goos, GOARCH: goarch})
		default:
			return nil, fmt.Errorf("invalid platform %q; use GOOS/GOARCH (e.g. linux/amd64), an OS name, or \"all\"", arg)
		}
	}
	return targets, nil
}

func resolveOutputPath(baseOutput string, target platformTarget, multi bool) string {
	if !multi {
		return target.binName(baseOutput)
	}
	name := baseOutput + "-" + target.label()
	if target.GOOS == "windows" && !strings.HasSuffix(name, ".exe") {
		name += ".exe"
	}
	return name
}
