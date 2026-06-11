package i18n

import (
	"io/fs"
	"log"

	"github.com/polagonow/pola/core"
)

type PluginOption func(*pluginConfig)

type pluginConfig struct {
	defaultLocale string
	directory     string
	fsys          fs.FS
}

func WithDefaultLocale(locale string) PluginOption {
	return func(c *pluginConfig) { c.defaultLocale = locale }
}

// WithDirectory sets the OS directory path to load locale files from.
// Used in development when files are on disk.
func WithDirectory(dir string) PluginOption {
	return func(c *pluginConfig) { c.directory = dir }
}

// WithFS sets an fs.FS to load locale files from.
// Use with embed.FS to bundle locales into the binary.
//
//	//go:embed locales
//	var locales embed.FS
//	i18n.Plugin(i18n.WithFS(locales))
func WithFS(fsys fs.FS) PluginOption {
	return func(c *pluginConfig) { c.fsys = fsys }
}

func Plugin(opts ...PluginOption) core.Plugin {
	cfg := &pluginConfig{
		defaultLocale: "en",
		directory:     "locales",
	}
	for _, o := range opts {
		o(cfg)
	}
	return core.PluginFunc{
		PluginName: "i18n",
		Fn: func(r *core.Registry) {
			b := NewBundle(cfg.defaultLocale)
			var err error
			if cfg.fsys != nil {
				err = b.LoadFS(cfg.fsys)
			} else {
				err = b.LoadDir(cfg.directory)
			}
			if err != nil {
				log.Printf("i18n: warning: %v", err)
			}
			defaultBundle = b
			core.ProvideValue[core.Translator](r, b)
		},
	}
}
