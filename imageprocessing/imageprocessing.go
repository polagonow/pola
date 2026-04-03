// Package imageprocessing provides an imgproxy-style image processing plugin
// for the Pola framework. It registers both an HTTP middleware (for direct
// browser requests at a configurable path prefix) and a RuntimeInjector that
// exposes ProcessURL to JS server components via the __DEPENDENCY_INJECTION__
// bridge.
package imageprocessing

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/polagonow/pola/core"
	"github.com/polagonow/pola/core/globals"
)

// ImageProcessor is the interface that adapters (e.g. disintegration/imaging)
// must implement to perform actual pixel manipulation.
type ImageProcessor interface {
	Process(src []byte, opts ProcessOptions) ([]byte, string, error)
}

// ProcessOptions holds the parameters for an image processing operation.
type ProcessOptions struct {
	Width   int     `json:"width,omitempty"`
	Height  int     `json:"height,omitempty"`
	Fit     string  `json:"fit,omitempty"`     // "cover", "contain", "fill"
	Format  string  `json:"format,omitempty"`  // "jpeg", "png", "gif"
	Quality int     `json:"quality,omitempty"` // 1-100
	Blur    float64 `json:"blur,omitempty"`
	Sharpen float64 `json:"sharpen,omitempty"`
	Rotate  int     `json:"rotate,omitempty"` // 0, 90, 180, 270
}

// ServiceConfig holds configuration read from environment/Polafile.
type ServiceConfig struct {
	PathPrefix string
	MaxWidth   int
	MaxHeight  int
	Format     string
}

// DefaultConfig returns configuration from environment variables with defaults.
func DefaultConfig() ServiceConfig {
	return ServiceConfig{
		PathPrefix: envOr("POLA_IMAGE_PROCESSING_PATH", "/_image"),
		MaxWidth:   envOrInt("POLA_IMAGE_PROCESSING_MAX_WIDTH", 4096),
		MaxHeight:  envOrInt("POLA_IMAGE_PROCESSING_MAX_HEIGHT", 4096),
		Format:     envOr("POLA_IMAGE_PROCESSING_FORMAT", "jpeg"),
	}
}

const maxBodySize = 10 << 20 // 10 MB

// Service is the image processing service that implements both core.Middleware
// and core.RuntimeInjector.
type Service struct {
	processor  ImageProcessor
	pathPrefix string
	maxWidth   int
	maxHeight  int
	format     string
	httpClient *http.Client
}

// New creates a Service from the given processor and config.
func New(processor ImageProcessor, cfg ServiceConfig) *Service {
	return &Service{
		processor:  processor,
		pathPrefix: cfg.PathPrefix,
		maxWidth:   cfg.MaxWidth,
		maxHeight:  cfg.MaxHeight,
		format:     cfg.Format,
		httpClient: newSafeHTTPClient(),
	}
}

// --- core.Middleware ---

func (s *Service) Name() string { return "imageprocessing" }

func (s *Service) Wrap(next http.Handler) http.Handler {
	prefix := s.pathPrefix
	if !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, prefix) && r.URL.Path != s.pathPrefix {
			next.ServeHTTP(w, r)
			return
		}
		s.serveImage(w, r)
	})
}

func (s *Service) serveImage(w http.ResponseWriter, r *http.Request) {
	opts := s.parseOptions(r)

	var src []byte
	var err error

	switch r.Method {
	case http.MethodGet:
		rawURL := r.URL.Query().Get("url")
		if rawURL == "" {
			http.Error(w, "missing url parameter", http.StatusBadRequest)
			return
		}
		src, err = s.fetchURL(rawURL)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
	case http.MethodPost:
		body := http.MaxBytesReader(w, r.Body, maxBodySize)
		src, err = io.ReadAll(body)
		if err != nil {
			http.Error(w, "failed to read request body", http.StatusBadRequest)
			return
		}
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	processed, contentType, err := s.processor.Process(src, opts)
	if err != nil {
		http.Error(w, "image processing failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	w.Header().Set("Content-Length", strconv.Itoa(len(processed)))
	w.Write(processed) //nolint:errcheck
}

// --- core.RuntimeInjector ---

func (s *Service) Capabilities() []core.InjectionCapability {
	return []core.InjectionCapability{
		{Name: "ImageProcessing.processURL", Description: "Process a remote image by URL and return a data URI"},
	}
}

// asyncDIRuntime is the optional interface for runtimes that support
// async (Promise + goroutine) dependency injection.
type asyncDIRuntime interface {
	SetDependencyInjection(map[string]func(args []any) (any, error)) error
}

func (s *Service) Inject(_ context.Context, runtime core.JSRuntime) error {
	fns := map[string]func(args []any) (any, error){
		"ImageProcessing.processURL": func(args []any) (any, error) {
			if len(args) < 1 {
				return nil, fmt.Errorf("processURL requires a url argument")
			}
			rawURL, ok := args[0].(string)
			if !ok {
				return nil, fmt.Errorf("processURL: url must be a string")
			}
			var opts ProcessOptions
			if len(args) > 1 {
				if b, err := json.Marshal(args[1]); err == nil {
					_ = json.Unmarshal(b, &opts)
				}
			}
			opts = s.applyDefaults(opts)
			return s.ProcessURL(rawURL, opts)
		},
	}

	if ar, ok := runtime.(asyncDIRuntime); ok {
		return ar.SetDependencyInjection(fns)
	}

	bridge := make(map[string]any, len(fns))
	for k, v := range fns {
		bridge[k] = v
	}
	return runtime.Set(globals.BridgeObject, bridge)
}

// --- Public API ---

// ProcessURL fetches an image from the given URL, processes it according to
// opts, and returns a data URI string suitable for use in <img src>.
func (s *Service) ProcessURL(rawURL string, opts ProcessOptions) (string, error) {
	src, err := s.fetchURL(rawURL)
	if err != nil {
		return "", err
	}

	processed, contentType, err := s.processor.Process(src, opts)
	if err != nil {
		return "", fmt.Errorf("image processing failed: %w", err)
	}

	encoded := base64.StdEncoding.EncodeToString(processed)
	return fmt.Sprintf("data:%s;base64,%s", contentType, encoded), nil
}

// --- Internals ---

func (s *Service) parseOptions(r *http.Request) ProcessOptions {
	q := r.URL.Query()
	opts := ProcessOptions{
		Width:   queryInt(q, "width"),
		Height:  queryInt(q, "height"),
		Fit:     q.Get("fit"),
		Format:  q.Get("format"),
		Quality: queryInt(q, "quality"),
		Blur:    queryFloat(q, "blur"),
		Sharpen: queryFloat(q, "sharpen"),
		Rotate:  queryInt(q, "rotate"),
	}
	return s.applyDefaults(opts)
}

func (s *Service) applyDefaults(opts ProcessOptions) ProcessOptions {
	if opts.Format == "" {
		opts.Format = s.format
	}
	// Clamp to configured maximums.
	if opts.Width > s.maxWidth {
		opts.Width = s.maxWidth
	}
	if opts.Height > s.maxHeight {
		opts.Height = s.maxHeight
	}
	return opts
}

func (s *Service) fetchURL(rawURL string) ([]byte, error) {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("invalid url: %w", err)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return nil, fmt.Errorf("unsupported scheme %q, only http and https are allowed", parsed.Scheme)
	}
	if host := parsed.Hostname(); host == "localhost" {
		return nil, fmt.Errorf("requests to localhost are not allowed")
	}

	resp, err := s.httpClient.Get(rawURL)
	if err != nil {
		return nil, fmt.Errorf("fetch image: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetch image: status %d", resp.StatusCode)
	}

	body := io.LimitReader(resp.Body, maxBodySize+1)
	data, err := io.ReadAll(body)
	if err != nil {
		return nil, fmt.Errorf("read image body: %w", err)
	}
	if len(data) > maxBodySize {
		return nil, fmt.Errorf("image too large (max %d bytes)", maxBodySize)
	}
	return data, nil
}

// newSafeHTTPClient returns an http.Client that validates resolved IPs at
// connection time (preventing DNS rebinding) and validates redirect targets
// (preventing SSRF via redirects).
func newSafeHTTPClient() *http.Client {
	dialer := &net.Dialer{Timeout: 10 * time.Second}
	transport := &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			host, port, err := net.SplitHostPort(addr)
			if err != nil {
				return nil, err
			}
			addrs, err := net.DefaultResolver.LookupHost(ctx, host)
			if err != nil {
				return nil, err
			}
			for _, resolved := range addrs {
				ip := net.ParseIP(resolved)
				if ip != nil && isPrivateIP(ip) {
					return nil, fmt.Errorf("requests to private addresses are not allowed")
				}
			}
			// Connect using the first resolved address to avoid a second lookup.
			return dialer.DialContext(ctx, network, net.JoinHostPort(addrs[0], port))
		},
	}
	return &http.Client{
		Timeout:   10 * time.Second,
		Transport: transport,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 10 {
				return fmt.Errorf("too many redirects")
			}
			host := req.URL.Hostname()
			if host == "localhost" {
				return fmt.Errorf("requests to localhost are not allowed")
			}
			if ip := net.ParseIP(host); ip != nil && isPrivateIP(ip) {
				return fmt.Errorf("requests to private addresses are not allowed")
			}
			return nil
		},
	}
}

func isPrivateIP(ip net.IP) bool {
	if ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsUnspecified() {
		return true
	}
	privateRanges := []string{
		"10.0.0.0/8",
		"172.16.0.0/12",
		"192.168.0.0/16",
		"169.254.0.0/16",
		"::1/128",
		"fc00::/7",
		"fe80::/10",
	}
	for _, cidr := range privateRanges {
		_, network, _ := net.ParseCIDR(cidr)
		if network.Contains(ip) {
			return true
		}
	}
	return false
}

func queryInt(q url.Values, key string) int {
	v := q.Get(key)
	if v == "" {
		return 0
	}
	n, _ := strconv.Atoi(v)
	return n
}

func queryFloat(q url.Values, key string) float64 {
	v := q.Get(key)
	if v == "" {
		return 0
	}
	f, _ := strconv.ParseFloat(v, 64)
	return f
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func envOrInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}
