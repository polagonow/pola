package cli

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// envFileCandidates are the .env files loaded (and detected for the startup
// banner), in ascending precedence order: the most specific file is last, so
// values from later files override earlier ones. Variables already present in
// the real process environment (i.e. exported in the shell) always win over
// every file.
var envFileCandidates = []string{
	".env",
	".env.local",
	".env.development",
	".env.development.local",
}

// loadDotenv reads the detected .env files in envFileCandidates order and
// injects their values into the current process environment via os.Setenv.
// Existing environment variables are never overwritten (the shell wins). It
// returns the base names of the files that were loaded, in banner order.
//
// Because the CLI's child processes (`go run` under `pola dev`) inherit
// os.Environ(), and `pola build` / `pola db` run in-process, loading here makes
// .env values visible to every subcommand and to the running app.
func loadDotenv(projectDir string) ([]string, error) {
	var loaded []string
	// Merge files first so that, among files, the most specific (last) wins.
	merged := map[string]string{}
	for _, name := range envFileCandidates {
		path := filepath.Join(projectDir, name)
		vars, err := parseEnvFile(path)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return loaded, fmt.Errorf("load %s: %w", name, err)
		}
		for k, v := range vars {
			merged[k] = v
		}
		loaded = append(loaded, name)
	}
	// Apply to the process, letting the shell win over every file.
	for k, v := range merged {
		if _, ok := os.LookupEnv(k); ok {
			continue
		}
		if err := os.Setenv(k, v); err != nil {
			return loaded, fmt.Errorf("set %s: %w", k, err)
		}
	}
	return loaded, nil
}

// parseEnvFile parses a single .env file into a map of key/value pairs. It
// supports blank lines, "#" comments, an optional "export " prefix, and
// single-/double-quoted values (double quotes honor \n, \t, \r, \\ and \"
// escapes; single quotes are literal). Unquoted values may carry a trailing
// " #" inline comment, which is stripped.
func parseEnvFile(path string) (map[string]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()

	vars := map[string]string{}
	scanner := bufio.NewScanner(f)
	// Allow long lines (e.g. base64 keys) beyond bufio's default 64KB.
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		line = strings.TrimPrefix(line, "export ")

		eq := strings.IndexByte(line, '=')
		if eq < 0 {
			continue // not a KEY=VALUE line
		}
		key := strings.TrimSpace(line[:eq])
		if key == "" {
			continue
		}
		vars[key] = parseEnvValue(strings.TrimSpace(line[eq+1:]))
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return vars, nil
}

// parseEnvValue interprets the right-hand side of a KEY=VALUE line.
func parseEnvValue(v string) string {
	if len(v) >= 2 {
		switch v[0] {
		case '"':
			if v[len(v)-1] == '"' {
				return unescapeDoubleQuoted(v[1 : len(v)-1])
			}
		case '\'':
			if v[len(v)-1] == '\'' {
				return v[1 : len(v)-1] // literal, no escaping
			}
		}
	}
	// Unquoted: strip a trailing inline comment introduced by whitespace + '#'.
	if i := strings.Index(v, " #"); i >= 0 {
		v = strings.TrimSpace(v[:i])
	}
	return v
}

// unescapeDoubleQuoted expands the escape sequences allowed inside a
// double-quoted .env value.
func unescapeDoubleQuoted(s string) string {
	r := strings.NewReplacer(
		`\n`, "\n",
		`\r`, "\r",
		`\t`, "\t",
		`\"`, `"`,
		`\\`, `\`,
	)
	return r.Replace(s)
}
