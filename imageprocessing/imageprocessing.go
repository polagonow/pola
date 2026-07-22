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
	"github.com/polagonow/pola/core/reserved"
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
	// AllowedHosts, when non-empty, restricts outbound image fetches to the
	// listed hostnames (case-insensitive, exact match). When empty (the
	// default), any public host is permitted subject to the private-range
	// blocking below. Production deployments SHOULD set an allowlist to
	// eliminate SSRF exposure entirely — see WithAllowedHosts.
	AllowedHosts []string
}

// Option configures a Service. Options are applied after ServiceConfig and
// override any equivalent config field.
type Option func(*Service)

// WithAllowedHosts restricts outbound image fetches to the given hostnames
// (case-insensitive, exact match on the URL host). When set, requests to any
// other host — public or private — are rejected before dialing.
//
// This is the strongest available SSRF mitigation and production deployments
// SHOULD configure it. When unset, the service falls back to blocking only the
// private/reserved IP ranges, which still allows fetches to arbitrary public
// hosts.
func WithAllowedHosts(hosts []string) Option {
	return func(s *Service) {
		s.allowedHosts = make(map[string]struct{}, len(hosts))
		for _, h := range hosts {
			h = strings.ToLower(strings.TrimSpace(h))
			if h != "" {
				s.allowedHosts[h] = struct{}{}
			}
		}
	}
}

// DefaultConfig returns configuration from environment variables with defaults.
func DefaultConfig() ServiceConfig {
	return ServiceConfig{
		PathPrefix: envOr("POLA_IMAGE_PROCESSING_PATH", reserved.Image),
		MaxWidth:   envOrInt("POLA_IMAGE_PROCESSING_MAX_WIDTH", 4096),
		MaxHeight:  envOrInt("POLA_IMAGE_PROCESSING_MAX_HEIGHT", 4096),
		Format:     envOr("POLA_IMAGE_PROCESSING_FORMAT", "jpeg"),
	}
}

const maxBodySize = 10 << 20 // 10 MB

// Service is the image processing service that implements both core.Middleware
// and core.RuntimeInjector.
type Service struct {
	processor    ImageProcessor
	pathPrefix   string
	maxWidth     int
	maxHeight    int
	format       string
	httpClient   *http.Client
	allowedHosts map[string]struct{} // nil/empty means no allowlist (any public host)
}

// New creates a Service from the given processor and config. Additional
// Options (e.g. WithAllowedHosts) may be supplied and are applied after the
// config; the variadic parameter keeps existing two-argument callers working.
func New(processor ImageProcessor, cfg ServiceConfig, opts ...Option) *Service {
	s := &Service{
		processor:  processor,
		pathPrefix: cfg.PathPrefix,
		maxWidth:   cfg.MaxWidth,
		maxHeight:  cfg.MaxHeight,
		format:     cfg.Format,
	}
	if len(cfg.AllowedHosts) > 0 {
		WithAllowedHosts(cfg.AllowedHosts)(s)
	}
	for _, opt := range opts {
		opt(s)
	}
	// Build the safe client after options so redirect checks can consult the
	// configured host allowlist.
	s.httpClient = newSafeHTTPClient(s.allowedHosts)
	return s
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
	host := parsed.Hostname()
	if host == "localhost" {
		return nil, fmt.Errorf("requests to localhost are not allowed")
	}
	// Restrict destination ports to 80 and 443 only. An empty port means the
	// scheme default (80 for http, 443 for https), which is allowed.
	if port := parsed.Port(); port != "" && port != "80" && port != "443" {
		return nil, fmt.Errorf("port %q is not allowed, only 80 and 443 are permitted", port)
	}
	// Optional host allowlist: when configured, only listed hosts may be
	// fetched. This is enforced here (pre-dial) and again in CheckRedirect.
	if len(s.allowedHosts) > 0 {
		if _, ok := s.allowedHosts[strings.ToLower(host)]; !ok {
			return nil, fmt.Errorf("host %q is not in the allowed hosts list", host)
		}
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
func newSafeHTTPClient(allowedHosts map[string]struct{}) *http.Client {
	dialer := &net.Dialer{Timeout: 10 * time.Second}
	transport := &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			host, port, err := net.SplitHostPort(addr)
			if err != nil {
				return nil, err
			}
			// Re-resolve at dial time to prevent DNS rebinding, then validate
			// every resolved address.
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
			if req.URL.Scheme != "http" && req.URL.Scheme != "https" {
				return fmt.Errorf("unsupported redirect scheme %q", req.URL.Scheme)
			}
			host := req.URL.Hostname()
			if host == "localhost" {
				return fmt.Errorf("requests to localhost are not allowed")
			}
			if port := req.URL.Port(); port != "" && port != "80" && port != "443" {
				return fmt.Errorf("redirect to port %q is not allowed", port)
			}
			if ip := net.ParseIP(host); ip != nil && isPrivateIP(ip) {
				return fmt.Errorf("requests to private addresses are not allowed")
			}
			// Enforce the host allowlist on redirect targets as well, so a
			// redirect cannot escape the configured allowlist.
			if len(allowedHosts) > 0 {
				if _, ok := allowedHosts[strings.ToLower(host)]; !ok {
					return fmt.Errorf("redirect host %q is not in the allowed hosts list", host)
				}
			}
			return nil
		},
	}
}

// blockedCIDRStrings enumerates the IP ranges that must never be reached by an
// outbound image fetch. Parsed once at package init into blockedCIDRs.
var blockedCIDRStrings = []string{
	// IPv4 private / special-use ranges.
	"10.0.0.0/8",      // RFC1918 private
	"172.16.0.0/12",   // RFC1918 private
	"192.168.0.0/16",  // RFC1918 private
	"169.254.0.0/16",  // link-local
	"127.0.0.0/8",     // loopback
	"0.0.0.0/8",       // "this host" / unspecified
	"100.64.0.0/10",   // CGNAT (RFC6598)
	"192.0.0.0/24",    // IETF protocol assignments (RFC6890)
	"192.0.2.0/24",    // TEST-NET-1 (RFC5737)
	"198.51.100.0/24", // TEST-NET-2 (RFC5737)
	"203.0.113.0/24",  // TEST-NET-3 (RFC5737)
	"198.18.0.0/15",   // benchmarking (RFC2544)
	"240.0.0.0/4",     // reserved / Class E (includes 255.255.255.255)
	// IPv6 ranges.
	"::1/128",      // loopback
	"::/128",       // unspecified
	"64:ff9b::/96", // NAT64 well-known prefix (RFC6052)
	"fc00::/7",     // unique-local
	"fe80::/10",    // link-local
	"ff00::/8",     // multicast
}

// blockedCIDRs is the parsed form of blockedCIDRStrings, built once at init.
var blockedCIDRs = func() []*net.IPNet {
	nets := make([]*net.IPNet, 0, len(blockedCIDRStrings))
	for _, cidr := range blockedCIDRStrings {
		if _, network, err := net.ParseCIDR(cidr); err == nil {
			nets = append(nets, network)
		}
	}
	return nets
}()

func isPrivateIP(ip net.IP) bool {
	if ip == nil {
		return true
	}
	// Unwrap IPv4-mapped IPv6 addresses (e.g. ::ffff:169.254.169.254) so the
	// IPv4 range checks below apply to them.
	if v4 := ip.To4(); v4 != nil {
		ip = v4
	}
	if ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() ||
		ip.IsInterfaceLocalMulticast() || ip.IsMulticast() || ip.IsUnspecified() {
		return true
	}
	for _, network := range blockedCIDRs {
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
