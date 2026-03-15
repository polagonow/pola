// Package esbuild implements the GoJSX Bundler interface using esbuild.
package esbuild

import (
	"gojsx/bundler"
	"gojsx/framework/contract"
)

// EsbuildBundler implements framework.Bundler using the two-pass esbuild
// pipeline. It accepts a contract.BundleInput (with pre-generated
// ServerEntryContent) and delegates to build.Bundle.
type EsbuildBundler struct{}

// Bundle runs both esbuild passes and returns the combined output.
func (b *EsbuildBundler) Bundle(input contract.BundleInput) (contract.BundleOutput, error) {
	cfg := bundler.BundlerConfig{
		AppDir:                 input.AppDir,
		OutDir:                 input.OutDir,
		AssetsURLPath:          input.AssetsURLPath,
		ClientEntry:            input.ClientEntry,
		ServerEntry:            input.ServerEntry,
		ClientComponents:       input.ClientComponents,
		External:               input.External,
		Dev:                    input.Dev,
		ServerEntryContent:     input.ServerEntryContent,
		ServerBundleConditions: input.ServerBundleConditions,
		ServerBundleDefines:    input.ServerBundleDefines,
	}
	result, err := bundler.Bundle(cfg)
	if err != nil {
		return contract.BundleOutput{}, err
	}
	return contract.BundleOutput{
		ServerBundle:   result.ServerBundle,
		ClientFiles:    result.ClientFiles,
		ClientEntryURL: result.ClientEntryOutput,
		ManifestJSON:   result.Manifest,
		ImportURLs:     result.ImportURLs,
	}, nil
}
