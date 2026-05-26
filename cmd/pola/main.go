package main

import (
	"os"

	_ "ariga.io/atlas-provider-gorm/gormschema" // keep in go.mod for migration temp programs
	_ "entgo.io/ent/dialect/sql/schema"         // keep in go.mod for migration temp programs
	cli "github.com/polagonow/pola/internal/cli"
)

func main() {
	if err := cli.Execute(); err != nil {
		os.Exit(1)
	}
}
