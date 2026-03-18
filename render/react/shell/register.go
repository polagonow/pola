package shell

import "github.com/polagonow/pola/framework"

func init() {
	framework.RegisterDefaults(framework.Defaults{
		HTMLShell: func() framework.HTMLShell { return &ReactHTMLShell{} },
	})
}
