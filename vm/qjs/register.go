package qjs

import "gojsx/framework"

func init() {
	framework.RegisterDefaults(framework.Defaults{
		VMFactory: func(bundle []byte) (framework.VMFactory, error) {
			return NewFastSchemaQJSVMFactory(bundle)
		},
	})
}
