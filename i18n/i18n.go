package i18n

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/polagonow/pola/core"
	"gopkg.in/yaml.v3"
)

type Bundle struct {
	mu            sync.RWMutex
	defaultLocale string
	translations  map[string]map[string]string
}

func NewBundle(defaultLocale string) *Bundle {
	return &Bundle{
		defaultLocale: defaultLocale,
		translations:  make(map[string]map[string]string),
	}
}

func (b *Bundle) Name() string { return "i18n" }

// LoadDir loads locale files from an OS directory path.
func (b *Bundle) LoadDir(dir string) error {
	return b.LoadFS(os.DirFS(dir))
}

// LoadFS loads locale files from any fs.FS (embed.FS, os.DirFS, etc.).
// Files must be .json, .yaml, or .yml at the root of the FS.
// The filename without extension is the locale (e.g. en.json → "en").
func (b *Bundle) LoadFS(fsys fs.FS) error {
	entries, err := fs.ReadDir(fsys, ".")
	if err != nil {
		return fmt.Errorf("i18n: read dir: %w", err)
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		ext := filepath.Ext(name)
		var locale string
		switch ext {
		case ".json":
			locale = strings.TrimSuffix(name, ".json")
		case ".yaml", ".yml":
			locale = strings.TrimSuffix(name, ext)
		default:
			continue
		}
		data, err := fs.ReadFile(fsys, name)
		if err != nil {
			return fmt.Errorf("i18n: read %s: %w", name, err)
		}
		var messages map[string]string
		switch ext {
		case ".json":
			err = json.Unmarshal(data, &messages)
		case ".yaml", ".yml":
			err = yaml.Unmarshal(data, &messages)
		}
		if err != nil {
			return fmt.Errorf("i18n: parse %s: %w", name, err)
		}
		b.translations[locale] = messages
	}
	return nil
}

// T returns the translated string for key in the given locale.
//
// Supports two interpolation styles:
//
//	Positional (fmt.Sprintf): "Hello, %s!" with string args
//	Named placeholders:       "Hello, {name}!" with a map[string]any arg
//
// When the first arg is a map[string]any, named placeholders like {name}
// are replaced. Otherwise args are passed to fmt.Sprintf.
func (b *Bundle) T(locale, key string, args ...any) string {
	b.mu.RLock()
	defer b.mu.RUnlock()
	msg := b.lookup(locale, key)
	if len(args) == 0 {
		return msg
	}
	if m, ok := args[0].(map[string]any); ok {
		return interpolate(msg, m)
	}
	return fmt.Sprintf(msg, args...)
}

func (b *Bundle) lookup(locale, key string) string {
	if msgs, ok := b.translations[locale]; ok {
		if msg, ok := msgs[key]; ok {
			return msg
		}
	}
	if locale != b.defaultLocale {
		if msgs, ok := b.translations[b.defaultLocale]; ok {
			if msg, ok := msgs[key]; ok {
				return msg
			}
		}
	}
	return key
}

func interpolate(msg string, vars map[string]any) string {
	for k, v := range vars {
		msg = strings.ReplaceAll(msg, "{"+k+"}", fmt.Sprint(v))
	}
	return msg
}

func (b *Bundle) Locales() []string {
	b.mu.RLock()
	defer b.mu.RUnlock()
	locales := make([]string, 0, len(b.translations))
	for k := range b.translations {
		locales = append(locales, k)
	}
	sort.Strings(locales)
	return locales
}

// defaultBundle is set by Plugin() so package-level helpers work without DI.
var defaultBundle *Bundle

// Locale returns the locale from the request context (set by the locale middleware).
func Locale(ctx context.Context) string {
	if v, ok := ctx.Value(core.LocaleContextKey).(string); ok {
		return v
	}
	return "en"
}

// T translates a key using the locale from the request context and the
// DI-registered bundle. Supports positional args (fmt.Sprintf) and named
// placeholders (map[string]any).
//
//	i18n.T(ctx, "greeting", "Alice")
//	i18n.T(ctx, "welcome", map[string]any{"name": "Alice", "count": 3})
func T(ctx context.Context, key string, args ...any) string {
	b := defaultBundle
	if b == nil {
		return key
	}
	return b.T(Locale(ctx), key, args...)
}

var _ core.Translator = (*Bundle)(nil)
