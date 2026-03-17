package disk

import "gojsx/framework"

func init() {
	framework.RegisterDefaults(framework.Defaults{
		AssetServer: func(dir string) framework.AssetServer { return NewAssetServer(dir) },
	})
}
