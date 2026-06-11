package locale

import (
	"context"
	"net/http"
	"strings"

	"github.com/polagonow/pola/core"
)

type Option func(*config)

type config struct {
	defaultLocale string
}

func WithDefaultLocale(locale string) Option {
	return func(c *config) { c.defaultLocale = locale }
}

type mw struct {
	defaultLocale string
}

func New(opts ...Option) core.Middleware {
	cfg := &config{defaultLocale: "en"}
	for _, o := range opts {
		o(cfg)
	}
	return &mw{defaultLocale: cfg.defaultLocale}
}

func (m *mw) Name() string { return "locale" }

func (m *mw) Wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		loc := m.detect(r)
		ctx := context.WithValue(r.Context(), core.LocaleContextKey, loc)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (m *mw) detect(r *http.Request) string {
	if lang := r.URL.Query().Get("lang"); lang != "" {
		return lang
	}
	if accept := r.Header.Get("Accept-Language"); accept != "" {
		tag := parseAcceptLanguage(accept)
		if tag != "" {
			return tag
		}
	}
	return m.defaultLocale
}

func parseAcceptLanguage(header string) string {
	parts := strings.SplitN(header, ",", 2)
	if len(parts) == 0 {
		return ""
	}
	lang := strings.TrimSpace(parts[0])
	if i := strings.IndexByte(lang, ';'); i > 0 {
		lang = lang[:i]
	}
	return strings.TrimSpace(lang)
}
