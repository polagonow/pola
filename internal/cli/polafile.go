package cli

import (
	"os"

	"github.com/polagonow/pola/polafile"
	"github.com/spf13/cobra"
)

// nameOnly strips the version suffix from a "name@version" string.
func nameOnly(s string) string {
	name, _ := polafile.ParseVersioned(s)
	return name
}

// applyPolafileDefaults loads Polafile.hcl from projectDir and sets flag
// defaults for any flags the user did not explicitly provide.
// Resolution order: CLI flag (explicit) > env var > Polafile > hardcoded default.
func applyPolafileDefaults(cmd *cobra.Command, projectDir string) {
	pf, err := polafile.Load(projectDir)
	if err != nil || pf == nil {
		return
	}
	applyIfUnchanged(cmd, "renderer", "POLA_RENDERER", nameOnly(pf.Renderer))
	applyIfUnchanged(cmd, "bundler", "POLA_BUNDLER", nameOnly(pf.Bundler))
	applyIfUnchanged(cmd, "router", "POLA_ROUTER", nameOnly(pf.Router))
	applyIfUnchanged(cmd, "css", "POLA_CSS", nameOnly(pf.CSS))
	applyIfUnchanged(cmd, "vm", "POLA_VM", nameOnly(pf.Engine))
	applyIfUnchanged(cmd, "cache", "POLA_CACHE", nameOnly(pf.Cache))
	applyIfUnchanged(cmd, "pm", "POLA_PM", nameOnly(pf.PackageManager))
}

// applyIfUnchanged sets a flag's value from the Polafile only if the user
// did not explicitly pass it via CLI and the corresponding env var is not set.
func applyIfUnchanged(cmd *cobra.Command, flagName, envKey, val string) {
	if val == "" {
		return
	}
	f := cmd.Flags().Lookup(flagName)
	if f == nil || f.Changed {
		return // user explicitly set this flag
	}
	if os.Getenv(envKey) != "" {
		return // env var takes precedence over Polafile
	}
	f.Value.Set(val)
}
