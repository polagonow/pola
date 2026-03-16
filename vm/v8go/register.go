package v8go

import "gojsx/framework"

func init() {
	framework.RegisterDefaults(framework.Defaults{
		VMFactory: func(bundle []byte) (framework.VMFactory, error) {
			return NewV8VMFactory(bundle)
		},
	})
}
