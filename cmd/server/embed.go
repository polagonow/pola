//go:build embed

package main

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"

	"github.com/polagonow/pola/core"
)

//go:embed all:public
var embeddedPublic embed.FS

func init() {
	sub, err := fs.Sub(embeddedPublic, "public")
	if err != nil {
		panic("embed: sub public: " + err.Error())
	}

	// Override the osfs asset server with one backed by the embedded FS.
	core.RegisterAssetServer(func(_ string) core.AssetServer {
		return &embedAssetServer{sub: sub}
	})

	// Register the prebuild loader so the pipeline skips route scanning and bundling.
	core.RegisterPrebuildLoader(func() (*core.PrebuildArtifacts, error) {
		// Read prebuild-meta.json (written during mage Bundle).
		metaBytes, err := fs.ReadFile(sub, "prebuild-meta.json")
		if err != nil {
			return nil, fmt.Errorf("embed: read prebuild-meta.json: %w", err)
		}
		var meta struct {
			Routes         []core.Route      `json:"routes"`
			ClientEntryURL string            `json:"clientEntryURL"`
			ImportURLs     map[string]string `json:"importURLs"`
			GlobalNotFound string            `json:"globalNotFound"`
			CSSURL         string            `json:"cssURL"`
		}
		if err := json.Unmarshal(metaBytes, &meta); err != nil {
			return nil, fmt.Errorf("embed: parse prebuild-meta.json: %w", err)
		}

		// Read the pre-built server bundle (_server.js lives at publicDir root).
		serverBundle, err := fs.ReadFile(sub, "_server.js")
		if err != nil {
			return nil, fmt.Errorf("embed: read _server.js: %w", err)
		}

		return &core.PrebuildArtifacts{
			Routes: meta.Routes,
			BundleOutput: &core.BundleOutput{
				ServerBundle:   serverBundle,
				ClientEntryURL: meta.ClientEntryURL,
				ImportURLs:     meta.ImportURLs,
			},
			GlobalNotFound: meta.GlobalNotFound,
			CSSURL:         meta.CSSURL,
		}, nil
	})
}

type embedAssetServer struct{ sub fs.FS }

func (s *embedAssetServer) Handler(prefix string) http.Handler {
	return http.StripPrefix(prefix, http.FileServer(http.FS(s.sub)))
}
