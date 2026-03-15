package shell

import "gojsx/framework"

func init() {
	framework.RegisterDefaults(framework.Defaults{
		HTMLShell: func() framework.HTMLShell { return &ReactHTMLShell{} },
	})
}
