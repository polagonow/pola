// Package generators defines the pluggable generator interface and registry
// for the Pola CLI. Each generator is a self-contained package that registers
// itself via init().
package generators

import (
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strings"

	"github.com/spf13/cobra"
)

// HookFunc is a Go function hook. It receives the project directory.
type HookFunc func(projectDir string) error

// Hook is a single after-hook: either a shell command or a Go function.
// Exactly one of Cmd or Func must be set.
type Hook struct {
	// Cmd is a shell command to exec (e.g. []string{"go", "mod", "tidy"}).
	Cmd []string
	// Func is a Go function to call.
	Func HookFunc
	// FuncName is a display name for logging (used when Func is set).
	FuncName string
}

// CmdHook creates a shell command hook.
func CmdHook(args ...string) Hook {
	return Hook{Cmd: args}
}

// FuncHook creates a Go function hook with a display name for logging.
func FuncHook(name string, fn HookFunc) Hook {
	return Hook{Func: fn, FuncName: name}
}

// Generator is a pluggable scaffold generator for the Pola CLI.
type Generator interface {
	// Name returns the generator identifier (e.g. "action", "route", "model").
	Name() string
	// Description returns a short human-readable description.
	Description() string
	// Command returns the cobra.Command for this generator.
	// The command is added as a subcommand of `pola generate`.
	Command() *cobra.Command
	// AfterHooks returns hooks to run after successful generation.
	// Hooks can be shell commands (CmdHook) or Go functions (FuncHook).
	// Return nil if no after-hooks are needed.
	AfterHooks() []Hook
}

var registry = map[string]Generator{}

// Register adds a generator to the global registry.
func Register(g Generator) {
	registry[g.Name()] = g
}

// All returns all registered generators sorted by name.
func All() []Generator {
	out := make([]Generator, 0, len(registry))
	for _, g := range registry {
		out = append(out, g)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].Name() < out[j].Name()
	})
	return out
}

// Get returns a generator by name, or an error if not found.
func Get(name string) (Generator, error) {
	g, ok := registry[name]
	if !ok {
		valid := make([]string, 0, len(registry))
		for k := range registry {
			valid = append(valid, k)
		}
		sort.Strings(valid)
		return nil, fmt.Errorf("unknown generator %q; available: %v", name, valid)
	}
	return g, nil
}

// Run invokes a registered generator by name with the given args.
// parentCmd is used to propagate persistent flags (--force, --skip-collision-check).
func Run(name string, parentCmd *cobra.Command, args []string) error {
	g, err := Get(name)
	if err != nil {
		return err
	}
	cmd := g.Command()
	// Add the persistent flags that CheckCollision and SkipTests expect.
	cmd.Flags().Bool("force", false, "")
	cmd.Flags().Bool("skip-collision-check", false, "")
	cmd.Flags().Bool("skip-tests", false, "")
	// Propagate values from parent.
	if parentCmd != nil {
		if v, _ := parentCmd.Flags().GetBool("force"); v {
			cmd.Flags().Set("force", "true")
		}
		if v, _ := parentCmd.Flags().GetBool("skip-collision-check"); v {
			cmd.Flags().Set("skip-collision-check", "true")
		}
		if v, _ := parentCmd.Flags().GetBool("skip-tests"); v {
			cmd.Flags().Set("skip-tests", "true")
		}
		if v, _ := parentCmd.Flags().GetString("service"); v != "" {
			if cmd.Flags().Lookup("service") != nil {
				cmd.Flags().Set("service", v)
			}
		}
	}
	cmd.SetArgs(args)
	return cmd.Execute()
}

// ShouldGenerateTests reports whether test files should be emitted for the
// current invocation. It returns false when --skip-tests is set on cmd; when
// the flag is absent it consults the Polafile (testing { generate_tests })
// via the supplied default, which callers pass as polafile.GenerateTests()
// (true when unset). The default is also true when polafileDefault is unsupplied.
func ShouldGenerateTests(cmd *cobra.Command, polafileDefault ...bool) bool {
	if cmd != nil {
		if skip, _ := cmd.Flags().GetBool("skip-tests"); skip {
			return false
		}
	}
	if len(polafileDefault) > 0 {
		return polafileDefault[0]
	}
	return true
}

// CheckCollision checks if a file already exists and applies --force /
// --skip-collision-check flag logic. Returns nil if generation should proceed,
// or an error if the file exists and --force was not set. Prints a message
// when overwriting. These flags are defined as persistent flags on the parent
// generate command and inherited by all subcommands.
func CheckCollision(cmd *cobra.Command, path string) error {
	skipCollision, _ := cmd.Flags().GetBool("skip-collision-check")
	if skipCollision {
		return nil
	}
	if _, err := os.Stat(path); err == nil {
		force, _ := cmd.Flags().GetBool("force")
		if !force {
			return fmt.Errorf("%s already exists; use --force to overwrite or --skip-collision-check to skip", path)
		}
		fmt.Printf("Overwriting %s\n", path)
	}
	return nil
}

// RunAfterHooks executes the after-hooks for a generator, streaming output
// to stdout/stderr. Returns the first error encountered.
func RunAfterHooks(g Generator, projectDir string) error {
	hooks := g.AfterHooks()
	if len(hooks) == 0 {
		return nil
	}
	for _, hook := range hooks {
		switch {
		case hook.Func != nil:
			name := hook.FuncName
			if name == "" {
				name = "func"
			}
			fmt.Printf("Running: %s\n", name)
			if err := hook.Func(projectDir); err != nil {
				return fmt.Errorf("after-hook %q failed: %w", name, err)
			}
		case len(hook.Cmd) > 0:
			fmt.Printf("Running: %s\n", strings.Join(hook.Cmd, " "))
			cmd := exec.Command(hook.Cmd[0], hook.Cmd[1:]...)
			cmd.Dir = projectDir
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			if err := cmd.Run(); err != nil {
				return fmt.Errorf("after-hook %q failed: %w", strings.Join(hook.Cmd, " "), err)
			}
		default:
			return fmt.Errorf("after-hook has neither Cmd nor Func set")
		}
	}
	return nil
}
