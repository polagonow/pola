package core

import (
	"errors"
	"fmt"
)

// ErrNoRenderer is returned when no renderer is configured.
var ErrNoRenderer = errors.New("pola: no Renderer registered; import a renderer plugin")

// ErrNoRouter is returned when no router is configured.
var ErrNoRouter = errors.New("pola: no Router registered; import a router plugin")

// ErrNoBundler is returned when no bundler is configured.
var ErrNoBundler = errors.New("pola: no Bundler registered; import a bundler plugin")

// ErrNoEngine is returned when no JS engine is configured.
var ErrNoEngine = errors.New("pola: no JSEngine registered; import an engine plugin")

// ErrNoFS is returned when no file system is configured.
var ErrNoFS = errors.New("pola: no FS registered; import a fs plugin")

// ErrWebAppPathRequired is returned when WebAppPath is empty.
var ErrWebAppPathRequired = errors.New("pola: Config.WebAppPath must be set")

// BuildError wraps a build pipeline error.
type BuildError struct {
	Stage string
	Err   error
}

func (e *BuildError) Error() string { return fmt.Sprintf("pola: build[%s]: %v", e.Stage, e.Err) }
func (e *BuildError) Unwrap() error { return e.Err }
