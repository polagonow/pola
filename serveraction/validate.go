package serveraction

import (
	"fmt"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

// MaxBoundArgs caps the number of arguments a server action call may carry.
// MAX_BOUND_ARGS.
const MaxBoundArgs = 1000

// maxBigIntDigits caps the magnitude of a number argument.
// MAX_BIGINT_DIGITS.
const maxBigIntDigits = 300

// ValidationConfig bounds the shape of decoded action arguments to prevent
// resource-exhaustion and prototype-pollution attacks.
type ValidationConfig struct {
	MaxDepth         int
	MaxStringLength  int
	MaxArrayLength   int
	MaxObjectKeys    int
	MaxTotalElements int
	AllowSpecialNum  bool
}

// ProductionConfig returns the strict default limits.
func ProductionConfig() ValidationConfig {
	return ValidationConfig{
		MaxDepth:         10,
		MaxStringLength:  10_000,
		MaxArrayLength:   1_000,
		MaxObjectKeys:    100,
		MaxTotalElements: 1_000_000,
		AllowSpecialNum:  false,
	}
}

// DevelopmentConfig returns the relaxed limits used during development.
func DevelopmentConfig() ValidationConfig {
	return ValidationConfig{
		MaxDepth:         20,
		MaxStringLength:  50_000,
		MaxArrayLength:   5_000,
		MaxObjectKeys:    500,
		MaxTotalElements: 5_000_000,
		AllowSpecialNum:  false,
	}
}

// ConfigFor returns the development config when dev is true, else production.
func ConfigFor(dev bool) ValidationConfig {
	if dev {
		return DevelopmentConfig()
	}
	return ProductionConfig()
}

type validationContext struct {
	totalElements int
	hasFork       bool
}

func (c *validationContext) bump(count int, cfg ValidationConfig) error {
	c.totalElements += count
	if c.totalElements > cfg.MaxTotalElements && c.hasFork {
		return fmt.Errorf("maximum array nesting exceeded: %d > %d", c.totalElements, cfg.MaxTotalElements)
	}
	return nil
}

// ValidateAndSanitizeArgs walks each decoded argument, enforcing depth/size
// limits and stripping dangerous object keys (prototype-pollution vectors).
// Returns a sanitized copy.
func ValidateAndSanitizeArgs(args []any, cfg ValidationConfig) ([]any, error) {
	ctx := &validationContext{}
	out := make([]any, len(args))
	for i, a := range args {
		v, err := validateValue(a, cfg, 0, ctx)
		if err != nil {
			return nil, err
		}
		out[i] = v
	}
	return out, nil
}

func validateValue(value any, cfg ValidationConfig, depth int, ctx *validationContext) (any, error) {
	if depth > cfg.MaxDepth {
		return nil, fmt.Errorf("maximum nesting depth exceeded: %d > %d", depth, cfg.MaxDepth)
	}

	switch v := value.(type) {
	case nil, bool:
		return v, nil
	case string:
		if len(v) > cfg.MaxStringLength {
			return nil, fmt.Errorf("string too long: %d > %d", len(v), cfg.MaxStringLength)
		}
		if err := ctx.bump(len(v), cfg); err != nil {
			return nil, err
		}
		return v, nil
	case float64:
		if !cfg.AllowSpecialNum && (math.IsInf(v, 0) || math.IsNaN(v)) {
			return nil, fmt.Errorf("invalid number: Infinity or NaN not allowed")
		}
		if abs := math.Abs(v); abs > 1e100 {
			digits := 1
			if abs != 0 {
				digits = int(math.Floor(math.Log10(abs))) + 1
			}
			if digits > maxBigIntDigits {
				return nil, fmt.Errorf("number too large: estimated %d digits but the limit is %d", digits, maxBigIntDigits)
			}
		}
		return v, nil
	case []any:
		if len(v) > cfg.MaxArrayLength {
			return nil, fmt.Errorf("array too large: %d > %d", len(v), cfg.MaxArrayLength)
		}
		if len(v) > 1 {
			ctx.hasFork = true
		}
		if err := ctx.bump(len(v)+1, cfg); err != nil {
			return nil, err
		}
		out := make([]any, len(v))
		for i, e := range v {
			ev, err := validateValue(e, cfg, depth+1, ctx)
			if err != nil {
				return nil, err
			}
			out[i] = ev
		}
		return out, nil
	case map[string]any:
		if len(v) > cfg.MaxObjectKeys {
			return nil, fmt.Errorf("too many object keys: %d > %d", len(v), cfg.MaxObjectKeys)
		}
		out := make(map[string]any, len(v))
		for key, val := range v {
			if IsDangerousProperty(key) {
				continue
			}
			vv, err := validateValue(val, cfg, depth+1, ctx)
			if err != nil {
				return nil, err
			}
			out[key] = vv
		}
		return out, nil
	default:
		// Unknown concrete type from JSON decoding — pass through unchanged.
		return v, nil
	}
}

// IsDangerousProperty reports object keys that enable prototype pollution and
// must be stripped from action arguments.
func IsDangerousProperty(key string) bool {
	switch key {
	case "__proto__", "constructor", "prototype",
		"__defineGetter__", "__defineSetter__",
		"__lookupGetter__", "__lookupSetter__":
		return true
	}
	return false
}

// IsReservedExportName reports export names reserved for internal use that must
// not be callable as actions.
func IsReservedExportName(name string) bool {
	switch name {
	case "then", "catch", "finally", "toString", "valueOf",
		"toLocaleString", "constructor", "Symbol",
		"@@iterator", "@@toStringTag":
		return true
	}
	return false
}

// CheckOrigin performs a same-origin CSRF check on a server-action request,
// comparing the Origin (or, failing that, Referer) header against the request's
// own scheme+host. Returns nil when the request is same-origin.
func CheckOrigin(r *http.Request) error {
	host := r.Host
	if host == "" {
		return fmt.Errorf("missing host header")
	}
	server := normalizeOrigin(requestScheme(r), host)

	if origin := r.Header.Get("Origin"); origin != "" {
		if u, err := url.Parse(origin); err == nil && originsEqual(originTupleOf(u), server) {
			return nil
		}
		return fmt.Errorf("origin mismatch")
	}

	if referer := r.Header.Get("Referer"); referer != "" {
		if u, err := url.Parse(referer); err == nil && originsEqual(originTupleOf(u), server) {
			return nil
		}
		return fmt.Errorf("referer mismatch")
	}

	return fmt.Errorf("missing origin and referer headers")
}

type originTuple struct {
	scheme string
	host   string
	port   int
}

func originsEqual(a, b originTuple) bool { return a == b }

func requestScheme(r *http.Request) string {
	if p := r.Header.Get("X-Forwarded-Proto"); p != "" {
		return p
	}
	if p := r.Header.Get("X-Forwarded-Protocol"); p != "" {
		return p
	}
	if r.TLS != nil {
		return "https"
	}
	return "http"
}

// normalizeOrigin builds a normalized origin tuple from a scheme and a host that
// may include a port.
func normalizeOrigin(scheme, hostport string) originTuple {
	host := hostport
	port := defaultPort(scheme)
	if i := strings.LastIndex(hostport, ":"); i != -1 {
		if p, err := strconv.Atoi(hostport[i+1:]); err == nil {
			host = hostport[:i]
			port = p
		}
	}
	return originTuple{scheme: scheme, host: host, port: port}
}

func originTupleOf(u *url.URL) originTuple {
	port := defaultPort(u.Scheme)
	if p := u.Port(); p != "" {
		if n, err := strconv.Atoi(p); err == nil {
			port = n
		}
	}
	return originTuple{scheme: u.Scheme, host: u.Hostname(), port: port}
}

func defaultPort(scheme string) int {
	switch scheme {
	case "https":
		return 443
	case "http":
		return 80
	default:
		return 0
	}
}
