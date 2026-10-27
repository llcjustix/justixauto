// Package requestmeta carries bounded security evidence without HTTP framework
// dependencies. Browser claims are evidence of what a client sent, not identity.
package requestmeta

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"time"
	"unicode/utf8"
)

const MaxSnapshotBytes = 8 * 1024

// Claim is untrusted client input. Status is captured, sanitized, invalid,
// combined, combined_sanitized, truncated or omitted_budget;
// absence from ClientSupplied means not supplied. Combined means multiple XFF
// header lines joined with a comma and space, never a verified proxy chain.
// XFF total-budget trimming appends _budget_truncated to its prior status.
type Claim struct {
	Value     string `json:"value"`
	Source    string `json:"source"`
	Status    string `json:"status"`
	Truncated bool   `json:"truncated"`
}

type Network struct {
	PeerIP           *string `json:"peerIp"`
	PeerPort         *uint16 `json:"peerPort"`
	ClientIP         *string `json:"clientIp"`
	ClientPort       *uint16 `json:"clientPort"`
	ClientIPSource   string  `json:"clientIpSource"`
	ExtractionStatus string  `json:"extractionStatus"`
}

// BackendTLS describes only the API connection, never the browser-to-edge leg.
type BackendTLS struct {
	Version string `json:"version"`
	Cipher  string `json:"cipher"`
	ALPN    string `json:"alpn"`
}

type Edge struct {
	Status     string  `json:"status"`
	HTTP       *string `json:"http"`
	TLSVersion *string `json:"tlsVersion"`
	TLSCipher  *string `json:"tlsCipher"`
	ALPN       *string `json:"alpn"`
	JA3        *string `json:"ja3"`
	JA4        *string `json:"ja4"`
	SourcePort *uint16 `json:"sourcePort"`
}

// AuthFacts may only be added from a server-resolved principal/session. Hashes
// are domain-separated SHA-256 hex, never raw cookies, tokens or session UUIDs.
type AuthFacts struct {
	ActorID              string     `json:"actorId,omitempty"`
	Login                string     `json:"login,omitempty"`
	DisplayName          string     `json:"displayName,omitempty"`
	SessionHash          string     `json:"sessionHash,omitempty"`
	SessionStartedAt     *time.Time `json:"sessionStartedAt,omitempty"`
	AuthenticationMethod string     `json:"authenticationMethod,omitempty"`
	DeviceHash           string     `json:"deviceHash,omitempty"`
	KnownDevice          *bool      `json:"knownDevice,omitempty"`
}

type Snapshot struct {
	RequestID         string           `json:"requestId"`
	Method            string           `json:"method"`
	Route             string           `json:"route"`
	Network           Network          `json:"network"`
	BackendHTTP       string           `json:"backendHttp"`
	BackendTLS        *BackendTLS      `json:"backendTls"`
	Edge              Edge             `json:"edge"`
	ClientSupplied    map[string]Claim `json:"clientSupplied"`
	Auth              *AuthFacts       `json:"auth,omitempty"`
	MetadataTruncated bool             `json:"metadataTruncated"`
}

type contextKey struct{}

// WithSnapshot and FromContext defensively copy all mutable data so consumers
// can enrich their context without changing another consumer's evidence.
func WithSnapshot(ctx context.Context, value Snapshot) context.Context {
	value = clone(value)
	bound(&value)
	return context.WithValue(ctx, contextKey{}, value)
}

func FromContext(ctx context.Context) (Snapshot, bool) {
	value, ok := ctx.Value(contextKey{}).(Snapshot)
	return clone(value), ok
}

// WithAuth enriches existing request evidence only; it never manufactures an
// HTTP request when invoked outside a captured request context.
func WithAuth(ctx context.Context, facts AuthFacts) context.Context {
	value, ok := FromContext(ctx)
	if !ok {
		return ctx
	}
	facts.ActorID, _, _ = sanitize(facts.ActorID, 64)
	facts.Login, _, _ = sanitize(facts.Login, 256)
	facts.DisplayName, _, _ = sanitize(facts.DisplayName, 256)
	facts.SessionHash = validHash(facts.SessionHash)
	facts.DeviceHash = validHash(facts.DeviceHash)
	if facts.AuthenticationMethod != "password" {
		facts.AuthenticationMethod = ""
	}
	value.Auth = &facts
	return WithSnapshot(ctx, value)
}

func validHash(value string) string {
	if len(value) != 64 {
		return ""
	}
	decoded, err := hex.DecodeString(value)
	if err != nil {
		return ""
	}
	return hex.EncodeToString(decoded)
}

func copyPtr[T any](p *T) *T {
	if p == nil {
		return nil
	}
	v := *p
	return &v
}

func clone(value Snapshot) Snapshot {
	value.Network.PeerIP = copyPtr(value.Network.PeerIP)
	value.Network.PeerPort = copyPtr(value.Network.PeerPort)
	value.Network.ClientIP = copyPtr(value.Network.ClientIP)
	value.Network.ClientPort = copyPtr(value.Network.ClientPort)
	value.BackendTLS = copyPtr(value.BackendTLS)
	value.Edge.HTTP = copyPtr(value.Edge.HTTP)
	value.Edge.TLSVersion = copyPtr(value.Edge.TLSVersion)
	value.Edge.TLSCipher = copyPtr(value.Edge.TLSCipher)
	value.Edge.ALPN = copyPtr(value.Edge.ALPN)
	value.Edge.JA3 = copyPtr(value.Edge.JA3)
	value.Edge.JA4 = copyPtr(value.Edge.JA4)
	value.Edge.SourcePort = copyPtr(value.Edge.SourcePort)
	if value.ClientSupplied != nil {
		claims := make(map[string]Claim, len(value.ClientSupplied))
		for name, claim := range value.ClientSupplied {
			claims[name] = claim
		}
		value.ClientSupplied = claims
	}
	value.Auth = copyPtr(value.Auth)
	if value.Auth != nil {
		value.Auth.SessionStartedAt = copyPtr(value.Auth.SessionStartedAt)
		value.Auth.KnownDevice = copyPtr(value.Auth.KnownDevice)
	}
	return value
}

// Trim in a stable field order, including JSON escaping overhead in the cap.
// Auth and API-observed connection facts take priority over browser claims.
func bound(value *Snapshot) {
	for i := len(browserFields) - 1; i >= 0; i-- {
		data, _ := json.Marshal(value)
		if len(data) <= MaxSnapshotBytes {
			return
		}
		name := browserFields[i].name
		claim, ok := value.ClientSupplied[name]
		if !ok || claim.Value == "" {
			continue
		}
		claim.Value, claim.Status, claim.Truncated = "", "omitted_budget", true
		if name == "forwardedFor" {
			// Keep the largest UTF-8 prefix that fits the remaining JSON budget.
			// The original combination/sanitization state remains in its source
			// status suffix when total-budget truncation becomes necessary.
			original := value.ClientSupplied[name]
			claim.Status = original.Status + "_budget_truncated"
			value.MetadataTruncated = true
			low, high := 0, len(original.Value)
			for low <= high {
				mid := (low + high) / 2
				end := mid
				for end > 0 && end < len(original.Value) && !utf8.RuneStart(original.Value[end]) {
					end--
				}
				candidate := claim
				candidate.Value = original.Value[:end]
				value.ClientSupplied[name] = candidate
				data, _ := json.Marshal(value)
				if len(data) <= MaxSnapshotBytes {
					claim.Value = candidate.Value
					low = mid + 1
				} else {
					high = mid - 1
				}
			}
			if claim.Value == "" {
				claim.Status = "omitted_budget"
			}
		}
		value.ClientSupplied[name] = claim
		value.MetadataTruncated = true
	}
}
