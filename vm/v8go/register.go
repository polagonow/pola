package v8go

import "github.com/polagonow/pola/framework"

func init() {
	framework.RegisterDefaults(framework.Defaults{
		VMFactory: func(bundle []byte) (framework.VMFactory, error) {
			return NewV8VMFactory(bundle)
		},
	})
}
