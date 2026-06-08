package serveraction

import "testing"

func TestModuleID(t *testing.T) {
	cases := []struct {
		name    string
		appDir  string
		path    string
		want    string
	}{
		{"nested", "/app", "/app/actions/todo.ts", "actions/todo"},
		{"tsx ext", "/app", "/app/actions/todo.tsx", "actions/todo"},
		{"root file", "/app", "/app/page.ts", "page"},
		{"node_modules", "/app", "/app/node_modules/@pkg/lib/index.ts", "@pkg/lib/index"},
		{"above appdir", "/app/web", "/app/shared/util.ts", "shared/util"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := ModuleID(c.appDir, c.path); got != c.want {
				t.Errorf("ModuleID(%q, %q) = %q, want %q", c.appDir, c.path, got, c.want)
			}
		})
	}
}

func TestKey(t *testing.T) {
	if got := Key("actions/todo", "addTodo"); got != "actions/todo:addTodo" {
		t.Errorf("Key = %q", got)
	}
}
