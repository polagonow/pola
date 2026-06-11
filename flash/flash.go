package flash

import (
	"context"

	"github.com/polagonow/pola/core"
	"github.com/polagonow/pola/middleware/session"
)

// Set adds a flash message under the given category.
func Set(ctx context.Context, category, message string) {
	data := session.Get(ctx)
	var flashes map[string]string
	if raw, ok := data["_flash"]; ok {
		if fm, ok := raw.(map[string]string); ok {
			flashes = fm
		}
	}
	if flashes == nil {
		flashes = make(map[string]string)
	}
	flashes[category] = message
	session.Set(ctx, "_flash", flashes)
}

// Get returns flash messages from context (set by the flash middleware).
func Get(ctx context.Context) map[string]string {
	if v, ok := ctx.Value(core.FlashContextKey).(map[string]string); ok {
		return v
	}
	return nil
}
