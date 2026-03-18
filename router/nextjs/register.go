//go:build nextjs

package nextjs

import "github.com/polagonow/pola/core"

func init() {
	core.RegisterRouter(func() core.Router { return New() })
}
