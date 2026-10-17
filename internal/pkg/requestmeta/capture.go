package requestmeta

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

// AcceptCH requests optional browser claims for subsequent requests. They never
// become authentication requirements; Critical-CH is deliberately not used.
const AcceptCH = "Sec-CH-UA-Full-Version-List, Sec-CH-UA-Platform-Version, Sec-CH-UA-Model, Sec-CH-UA-Arch"

// ProxyConfig is immutable after parsing. Its zero value trusts no proxy.
type ProxyConfig struct {
	trusted []netip.Prefix
	xff     bool
}

// ParseProxyConfig accepts only explicit CIDRs and the single XFF contract.
// No private, loopback, link-local or vendor networks are implicitly trusted.
func ParseProxyConfig(cidrs, header string) (ProxyConfig, error) {
	var cfg ProxyConfig
	header = strings.ToLower(strings.TrimSpace(header))
	if header != "" && header != "none" && header != "x-forwarded-for" {
		return cfg, fmt.Errorf("AUDIT_FORWARDED_HEADER must be none or x-forwarded-for")
	}
	if len(cidrs) > 4096 {
		return cfg, fmt.Errorf("AUDIT_TRUSTED_PROXY_CIDRS exceeds 4096 bytes")
	}
	if strings.TrimSpace(cidrs) != "" {
		for _, raw := range strings.Split(cidrs, ",") {
			prefix, err := netip.ParsePrefix(strings.TrimSpace(raw))
			if err != nil || prefix.Addr().Is4In6() {
				return ProxyConfig{}, fmt.Errorf("AUDIT_TRUSTED_PROXY_CIDRS must contain valid IPv4 or IPv6 CIDRs")
			}
			cfg.trusted = append(cfg.trusted, prefix.Masked())
		}
	}
	cfg.xff = header == "x-forwarded-for"
	if cfg.xff != (len(cfg.trusted) > 0) {
		return ProxyConfig{}, fmt.Errorf("AUDIT_TRUSTED_PROXY_CIDRS and AUDIT_FORWARDED_HEADER=x-forwarded-for must be configured together")
	}
	return cfg, nil
}

var browserFields = []struct {
	header, name string
	limit        int
}{
	{"User-Agent", "userAgent", 512},
	{"Sec-CH-UA", "uaBrands", 512},
	{"Sec-CH-UA-Mobile", "uaMobile", 16},
	{"Sec-CH-UA-Platform", "uaPlatform", 128},
	{"Accept-Language", "acceptLanguage", 256},
	{"Origin", "origin", 512},
	{"Referer", "referer", 1024},
	{"Sec-Fetch-Site", "fetchSite", 32},
	{"Sec-Fetch-Mode", "fetchMode", 32},
	{"Sec-Fetch-Dest", "fetchDest", 32},
	{"Sec-Fetch-User", "fetchUser", 16},
	{"X-Client-Timezone", "timezone", 64},
	{"X-Client-App-Version", "appVersion", 128},
	{"X-Client-Screen", "screen", 64},
	{"Sec-CH-UA-Full-Version-List", "uaFullVersionList", 512},
	{"Sec-CH-UA-Platform-Version", "uaPlatformVersion", 128},
	{"Sec-CH-UA-Model", "uaModel", 128},
	{"Sec-CH-UA-Arch", "uaArch", 32},
	{"X-Forwarded-For", "forwardedFor", 4096},
}

// Capture reads only the allowlisted headers and API socket connection. route
// must be the framework's matched template, never the raw URL or request path.
func Capture(r *http.Request, requestID, route string, cfg ProxyConfig) Snapshot {
	value := Snapshot{
		RequestID: requestID, Network: captureNetwork(r, cfg),
		Edge: Edge{Status: "not_configured"}, ClientSupplied: map[string]Claim{},
	}
	value.Method, _, _ = sanitize(r.Method, 32)
	value.Route, _, _ = sanitize(route, 512)
	value.BackendHTTP, _, _ = sanitize(r.Proto, 32)
	if r.TLS != nil {
		alpn, _, _ := sanitize(r.TLS.NegotiatedProtocol, 64)
		value.BackendTLS = &BackendTLS{
			Version: tls.VersionName(r.TLS.Version), Cipher: tls.CipherSuiteName(r.TLS.CipherSuite), ALPN: alpn,
		}
	}
	for _, field := range browserFields {
		values := r.Header.Values(field.header)
		if len(values) == 0 {
			continue
		}
		if field.name == "forwardedFor" {
			// Preserve unverified header evidence independently of whether the
			// separate trusted-IP extraction accepts this header or socket peer.
			value.ClientSupplied[field.name] = captureForwardedClaim(values, field.limit)
			continue
		}
		claim := Claim{Source: "client_supplied", Status: "captured"}
		// Duplicate browser headers are ambiguous. Do not silently choose one.
		if len(values) != 1 {
			claim.Status = "invalid"
		} else {
			var changed bool
			claim.Value, claim.Truncated, changed = sanitize(values[0], field.limit)
			if changed {
				claim.Status = "sanitized"
			}
			if claim.Truncated {
				claim.Status = "truncated"
			}
			if field.name == "origin" || field.name == "referer" {
				// Never parse a truncated/cleaned URL into plausible but changed
				// evidence. Invalid input is omitted without retaining raw text.
				if changed || claim.Truncated {
					claim.Value, claim.Status = "", "invalid"
				} else if cleaned, ok := cleanURL(claim.Value, field.name == "referer"); ok {
					if len(cleaned) > field.limit {
						claim.Value, claim.Status, claim.Truncated = "", "invalid", true
					} else if cleaned != claim.Value {
						claim.Status = "sanitized"
						claim.Value = cleaned
					}
				} else {
					claim.Value, claim.Status = "", "invalid"
				}
			}
		}
		value.ClientSupplied[field.name] = claim
	}
	bound(&value)
	return value
}

func captureForwardedClaim(values []string, limit int) Claim {
	claim := Claim{Source: "client_supplied", Status: "captured"}
	if len(values) > 1 {
		claim.Status = "combined"
	}
	var joined strings.Builder
	appendPart := func(part string) bool {
		remaining := limit - joined.Len()
		if len(part) > remaining {
			joined.WriteString(part[:remaining])
			claim.Truncated = true
			return false
		}
		joined.WriteString(part)
		return true
	}
	for i, raw := range values {
		if i > 0 && !appendPart(", ") {
			break
		}
		if !appendPart(raw) {
			break
		}
	}
	var changed bool
	claim.Value, _, changed = sanitize(joined.String(), limit)
	if changed {
		if len(values) > 1 {
			claim.Status = "combined_sanitized"
		} else {
			claim.Status = "sanitized"
		}
	} else if claim.Truncated && len(values) == 1 {
		claim.Status = "truncated"
	}
	return claim
}

// sanitize examines at most limit input bytes, strips controls/format controls
// and malformed UTF-8, and never splits a retained UTF-8 code point.
func sanitize(raw string, limit int) (value string, truncated, changed bool) {
	truncated = len(raw) > limit
	if truncated {
		raw = raw[:limit]
	}
	var out strings.Builder
	out.Grow(len(raw))
	for len(raw) > 0 {
		r, size := utf8.DecodeRuneInString(raw)
		if (r == utf8.RuneError && size == 1) || unicode.IsControl(r) || unicode.Is(unicode.Cf, r) {
			changed = true
		} else {
			out.WriteRune(r)
		}
		raw = raw[size:]
	}
	return out.String(), truncated, changed
}

func cleanURL(raw string, path bool) (string, bool) {
	u, err := url.Parse(raw)
	if err != nil || u.Opaque != "" || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") || strings.Contains(u.Host, "%") {
		return "", false
	}
	if u.Hostname() == "" || strings.HasSuffix(u.Host, ":") {
		return "", false
	}
	// net/url parses bracketed host syntax but does not validate an IP literal.
	if strings.HasPrefix(u.Host, "[") || strings.Contains(u.Hostname(), ":") {
		addr, err := netip.ParseAddr(u.Hostname())
		if err != nil || !addr.Is6() || addr.Zone() != "" {
			return "", false
		}
	}
	if port := u.Port(); port != "" {
		if n, err := strconv.ParseUint(port, 10, 16); err != nil || n == 0 {
			return "", false
		}
	}
	u.User, u.RawQuery, u.Fragment, u.RawFragment = nil, "", "", ""
	u.ForceQuery = false
	if !path {
		u.Path, u.RawPath = "", ""
	}
	return u.String(), true
}

func (cfg ProxyConfig) trusts(addr netip.Addr) bool {
	for _, prefix := range cfg.trusted {
		if prefix.Contains(addr) {
			return true
		}
	}
	return false
}

func captureNetwork(r *http.Request, cfg ProxyConfig) Network {
	value := Network{ClientIPSource: "network_peer", ExtractionStatus: "not_configured"}
	host, port, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		value.ExtractionStatus = "invalid_peer"
		return value
	}
	peer, err := netip.ParseAddr(host)
	if err != nil || peer.Zone() != "" {
		value.ExtractionStatus = "invalid_peer"
		return value
	}
	peer = peer.Unmap()
	ip := peer.String()
	value.PeerIP, value.ClientIP = &ip, &ip
	if n, err := strconv.ParseUint(port, 10, 16); err == nil && n != 0 {
		p := uint16(n)
		value.PeerPort = &p
	}
	if !cfg.xff {
		return value
	}
	if !cfg.trusts(peer) {
		value.ExtractionStatus = "untrusted_peer"
		return value
	}
	values := r.Header.Values("X-Forwarded-For")
	if len(values) == 0 {
		value.ExtractionStatus = "missing_forwarded_header"
		return value
	}
	bytes := 0
	for i, raw := range values {
		bytes += len(raw)
		if i > 0 {
			bytes++
		}
		if bytes > 4096 {
			value.ExtractionStatus = "forwarded_too_large"
			return value
		}
	}
	chain := strings.Split(strings.Join(values, ","), ",")
	if len(chain) > 16 {
		value.ExtractionStatus = "too_many_forwarded_hops"
		return value
	}
	hops := make([]netip.Addr, 0, len(chain))
	for _, raw := range chain {
		addr, err := netip.ParseAddr(strings.TrimSpace(raw))
		if err != nil || addr.Zone() != "" {
			value.ExtractionStatus = "malformed_forwarded_header"
			return value
		}
		hops = append(hops, addr.Unmap())
	}
	for i := len(hops) - 1; i >= 0; i-- {
		if !cfg.trusts(hops[i]) {
			ip := hops[i].String()
			value.ClientIP, value.ClientIPSource = &ip, "trusted_proxy_xff"
			value.ExtractionStatus = "first_untrusted_hop"
			return value
		}
	}
	value.ExtractionStatus = "no_untrusted_hop"
	return value
}
