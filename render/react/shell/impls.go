package shell

import "github.com/polagonow/pola/framework/contract"

// ReactHTMLShell implements framework.HTMLShell using the React RSC HTML shell template.
type ReactHTMLShell struct{}

// Render returns the complete HTML document for a page shell.
func (s *ReactHTMLShell) Render(p contract.ShellParams) string {
	return Render(p)
}
