// Package reserved defines Pola's reserved HTTP route namespace.
//
// All framework-internal endpoints live under a single prefix (Prefix) so they
// are predictable and never collide with application routes — mirroring the way
// other RSC frameworks namespace their machinery (e.g. rari's "/_rari").
//
// It is a leaf package (no imports) so any subsystem can reference these
// constants without risking an import cycle.
package reserved

// Prefix is the reserved namespace under which all framework-internal HTTP
// endpoints are served.
const Prefix = "/_pola"

// Reserved framework endpoints, each derived from Prefix.
const (
	// Image is the on-the-fly image optimization endpoint. Overridable via
	// POLA_IMAGE_PROCESSING_PATH or Polafile image_processing.path.
	Image = Prefix + "/image"

	// Hot is the dev-mode hot-reload WebSocket endpoint.
	Hot = Prefix + "/hot"

	// Assets is the URL prefix for framework-built JS/CSS chunks (served from
	// <publicDir>/assets). User static files remain under "/public/".
	Assets = Prefix + "/assets"

	// Metrics is the default metrics endpoint. Overridable via POLA_METRICS_PATH.
	Metrics = Prefix + "/metrics"

	// Pprof is the default pprof endpoint prefix. Overridable via POLA_PPROF_PATH.
	Pprof = Prefix + "/pprof"
)
