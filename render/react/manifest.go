package react

import (
	"gojsx/bundler/manifest"
	"gojsx/framework/contract"
	"gojsx/framework/globals"
)

// WebpackManifestEncoder implements framework.ManifestEncoder using the
// webpack-format client manifest consumed by react-server-dom-webpack.
type WebpackManifestEncoder struct{}

// InjectGlobal returns the JS define name used to embed the manifest in the
// server bundle.
func (e *WebpackManifestEncoder) InjectGlobal() string { return globals.ClientManifest }

// Encode builds the client manifest and import URL map from the list of
// discovered client components and their compiled output files.
func (e *WebpackManifestEncoder) Encode(
	clientComponents []string,
	clientFiles map[string][]byte,
	appDir, assetsURLPath string,
	inputChunkURLs map[string]string,
) (contract.ClientManifest, map[string]string, error) {
	m, importURLs, err := manifest.Build(clientComponents, clientFiles, appDir, assetsURLPath, inputChunkURLs)
	if err != nil {
		return nil, nil, err
	}
	result := make(contract.ClientManifest, len(m))
	for k, v := range m {
		result[k] = contract.ClientRef{
			ID:     v.ID,
			Name:   v.Name,
			Chunks: v.Chunks,
			Async:  v.Async,
		}
	}
	return result, importURLs, nil
}
