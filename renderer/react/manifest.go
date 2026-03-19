package react

import (
	"encoding/json"
	"fmt"

	"github.com/polagonow/pola/core"
)

// LoadManifest parses the client component manifest JSON.
func LoadManifest(data []byte) (ClientManifest, error) {
	var m ClientManifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("manifest: %w", err)
	}
	return m, nil
}

// WebpackManifestEncoder builds the webpack-format client manifest consumed
// by react-server-dom-webpack.
type WebpackManifestEncoder struct{}

// InjectGlobal returns the JS define name used to embed the manifest in the
// server bundle.
func (e *WebpackManifestEncoder) InjectGlobal() string { return "__CLIENT_MANIFEST__" }

// Encode builds the client manifest and import URL map from the list of
// discovered client components and their compiled output files.
//
// clientComponents is the list of absolute source paths.
// clientFiles maps relative output path → compiled bytes.
// appDir is the root of the user's frontend app.
// assetsURLPath is the URL prefix for hashed asset files.
// inputChunkURLs maps source path → browser chunk URL.
func (e *WebpackManifestEncoder) Encode(
	clientComponents []string,
	clientFiles map[string][]byte,
	appDir, assetsURLPath string,
	inputChunkURLs map[string]string,
) (core.ClientManifest, map[string]string, error) {
	// Build a manifest that maps each client component's moduleID to a ClientRef.
	// The moduleID is derived from the relative path within appDir, slash-separated
	// without extension, matching how react-server-dom-webpack identifies modules.
	manifest := make(core.ClientManifest, len(clientComponents))
	importURLs := make(map[string]string, len(clientComponents))

	for _, absPath := range clientComponents {
		chunkURL, ok := inputChunkURLs[absPath]
		if !ok {
			continue
		}
		ref := core.ClientRef{
			ID:     absPath,
			Name:   "*",
			Chunks: []string{chunkURL},
			Async:  false,
		}
		manifest[absPath] = ref
		importURLs[absPath] = chunkURL
	}

	return manifest, importURLs, nil
}

// MarshalManifest JSON-encodes a ClientManifest for embedding in build output.
func MarshalManifest(m core.ClientManifest) ([]byte, error) {
	b, err := json.Marshal(m)
	if err != nil {
		return nil, fmt.Errorf("manifest: marshal: %w", err)
	}
	return b, nil
}
