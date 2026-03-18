// Package combo registers bundler+renderer combinations for the e2e test suite.
// Each file registers one combination via fixture.RegisterBundlerRenderer.
//
// To add a new bundler+renderer pair: create a new file here, implement
// BundlerRendererFixture, and call fixture.RegisterBundlerRenderer from init().
// Tests automatically run every VM × every registered combo.
package combo

import (
	esbuild "github.com/polagonow/pola/bundler/esbuild"
	"github.com/polagonow/pola/framework"
	"github.com/polagonow/pola/framework/contract"
	react "github.com/polagonow/pola/render/react"
	_ "github.com/polagonow/pola/render/react/discovery/nextjs"
	_ "github.com/polagonow/pola/render/react/shell"
	"github.com/polagonow/pola/test/fixture"
)

func init() { fixture.RegisterBundlerRenderer(&esbuildReactCombo{}) }

type esbuildReactCombo struct{}

func (c *esbuildReactCombo) BundlerName() string  { return "esbuild" }
func (c *esbuildReactCombo) RendererName() string { return "react" }

func (c *esbuildReactCombo) NewBundler() framework.Bundler { return esbuild.NewBundler() }

func (c *esbuildReactCombo) NewRendererFactory() func(framework.VMPool, framework.StreamProtocol, contract.BridgeConfig) framework.Renderer {
	return func(pool framework.VMPool, protocol framework.StreamProtocol, bridge contract.BridgeConfig) framework.Renderer {
		return react.NewVMRenderer(pool, protocol, bridge)
	}
}
