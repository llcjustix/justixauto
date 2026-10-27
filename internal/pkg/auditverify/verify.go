// Package auditverify verifies exported identity audit evidence without a database
// or network. A retained checkpoint authenticates only its historical prefix;
// local consistency alone cannot prove that a privileged writer kept the tail.
package auditverify

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

// Limits bound memory, parsing depth, numeric work and total input. Pages are
// processed individually; history is never accumulated in memory. Oversized
// otherwise valid evidence must be exported in smaller pages.
const (
	MaxPageBytes       = 16 << 20
	MaxPayloadBytes    = 1 << 20
	MaxEntryBytes      = 4 << 20
	MaxBoundaryBytes   = 8 << 10
	MaxCheckpointBytes = 4 << 10
	MaxTotalBytes      = 1 << 30
	MaxPages           = 100000
	MaxEvents          = 5000000
	MaxPageEvents      = 500
	MaxDepth           = 64
	MaxNodes           = 250000
	MaxStringBytes     = 2 << 20
	MaxNumberBytes     = 4096
	MaxExponent        = 1000000
)

// ErrLimit distinguishes a resource-policy refusal from damaged evidence. No
// evidence is silently skipped when a limit is reached.
var ErrLimit = errors.New("verification resource limit exceeded")

func limit(detail string) error { return fmt.Errorf("%w: %s", ErrLimit, detail) }

const (
	eventPrefix   = "justixauto-audit-v1\n"
	genesisPrefix = "justixauto-audit-genesis-v1\n"
	timestampForm = "2006-01-02T15:04:05.000000Z"
)

var (
	canonicalFields         = strings.Fields("streamId chainVersion chainIndex seq id occurredAt actorUserId actorLogin actorDisplayName action resourceType resourceId companyId reason outcome errorCode clientIp details before after metadata requestId sessionHash deviceHash")
	eventFields             = append(append([]string{}, canonicalFields...), "prevHash", "hash", "canonicalPayloadBase64")
	boundaryFields          = strings.Fields("streamId legacyMaxSeq legacyCount createdAt canonicalPayloadBase64 genesisHash boundaryHash")
	boundaryCanonicalFields = strings.Fields("streamId legacyMaxSeq legacyCount createdAt")
	checkpointFields        = strings.Fields("streamId index hash boundaryHash")
	pageFields              = strings.Fields("formatVersion streamId segment verificationStatus boundary fence fromCheckpoint toCheckpoint events legacy nextCursor chainComplete complete")
	legacyFields            = strings.Fields("status fromSeq toSeq fromCount toCount events")
)

// Checkpoint has the same format as export checkpoints. Creating one locally
// does not establish external custody; callers supply that provenance.
type Checkpoint struct {
	StreamID     string `json:"streamId"`
	Index        string `json:"index"`
	Hash         string `json:"hash"`
	BoundaryHash string `json:"boundaryHash"`
}

// Result separates the externally retained prefix from the locally consistent
// tail and the unauthenticated legacy segment. Checkpoint is suitable for later
// operator-controlled external storage, not an automatically trusted receipt.
type Result struct {
	Status           string      `json:"status"`
	Pages            int64       `json:"pages"`
	VerifiedEvents   int64       `json:"verifiedEvents"`
	LegacyEvents     int64       `json:"legacyEvents"`
	LegacyStatus     string      `json:"legacyStatus"`
	Coverage         string      `json:"coverage"`
	VerificationFrom Checkpoint  `json:"verificationFrom"`
	AnchoredThrough  *Checkpoint `json:"anchoredThrough,omitempty"`
	PrefixStatus     string      `json:"prefixStatus"`
	TailStatus       string      `json:"tailStatus"`
	Checkpoint       Checkpoint  `json:"checkpoint"`
}

// Verifier consumes complete page objects in order, across files or JSONL.
// It is single-use and not safe for concurrent access. Any error poisons it.
type Verifier struct {
	external                 *Checkpoint
	anchorSeen, initialized  bool
	complete, legacy, failed bool
	bytes, pages, events     int64
	legacySeq, legacyCount   int64
	legacyMax, legacyTotal   int64
	current, fence, start    Checkpoint
	boundaryPayload          []byte
}

// New accepts an independently retained checkpoint, or nil for unanchored work.
// A continuation may start exactly at that checkpoint, never an arbitrary row.
func New(external *Checkpoint) (*Verifier, error) {
	v := &Verifier{}
	if external != nil {
		c := *external
		if err := validCheckpoint(c); err != nil {
			return nil, fmt.Errorf("external checkpoint: %w", err)
		}
		v.external = &c
	}
	return v, nil
}

// ReadCheckpoint decodes exactly one strict checkpoint object (not a Result).
func ReadCheckpoint(r io.Reader) (*Checkpoint, error) {
	b, err := io.ReadAll(io.LimitReader(r, MaxCheckpointBytes+1))
	if err != nil {
		return nil, err
	}
	if len(b) > MaxCheckpointBytes {
		return nil, limit("checkpoint exceeds 4 KiB")
	}
	m, err := parseObject(b)
	if err != nil {
		return nil, err
	}
	c, err := checkpoint(m)
	return &c, err
}

// Consume accepts whitespace-separated JSON objects, including JSONL and one
// page per file. The top-level format is an export envelope, not a JSON array.
func (v *Verifier) Consume(r io.Reader) error {
	if v.failed {
		return errors.New("verifier has already failed")
	}
	br := bufio.NewReaderSize(r, 32<<10)
	for {
		b, err := v.readPage(br)
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err == nil {
			err = v.consumePage(b)
		}
		if err != nil {
			v.failed = true
			return fmt.Errorf("page %d: %w", v.pages, err)
		}
	}
}

func (v *Verifier) consumePage(b []byte) error {
	switch {
	case v.complete:
		return errors.New("page supplied after completed export")
	case v.pages >= MaxPages:
		return limit("page count exceeds 100000")
	}
	v.pages++
	m, err := parseObject(b)
	if err != nil {
		return err
	}
	return v.page(m)
}

// Finish succeeds only after all fenced chain rows and all legacy positions
// have been consumed and an optional external checkpoint has been encountered.
func (v *Verifier) Finish() (Result, error) {
	if v.failed || !v.initialized || !v.complete || v.current != v.fence || v.legacyCount != v.legacyTotal {
		v.failed = true
		return Result{}, errors.New("incomplete or invalid export: final fence/legacy coverage not reached")
	}
	if v.external != nil && !v.anchorSeen {
		v.failed = true
		return Result{}, errors.New("external checkpoint not encountered (missing or changed anchored tail)")
	}
	r := Result{
		Status: "internally_consistent_unanchored", Pages: v.pages, VerifiedEvents: v.events,
		LegacyEvents: v.legacyCount, LegacyStatus: "prechain_unverified", Coverage: "genesis_through_supplied_fence",
		VerificationFrom: v.start, PrefixStatus: "internally_consistent_unanchored",
		TailStatus: "internally_consistent_unanchored", Checkpoint: v.current,
	}
	if v.external != nil {
		c := *v.external
		r.Status, r.AnchoredThrough, r.PrefixStatus = "externally_anchored_prefix", &c, "externally_anchored_through_checkpoint_only"
	}
	if v.start.Index != "0" {
		r.Coverage, r.PrefixStatus = "external_checkpoint_through_supplied_fence", "externally_checkpointed_not_replayed"
	}
	return r, nil
}

func (v *Verifier) readPage(r *bufio.Reader) ([]byte, error) {
	var b []byte
	var s pageScanner
	for {
		c, err := r.ReadByte()
		if err != nil {
			if errors.Is(err, io.EOF) && len(b) > 0 {
				return nil, errors.New("truncated JSON page")
			}
			return nil, err
		}
		v.bytes++
		if v.bytes > MaxTotalBytes {
			return nil, limit("total export input exceeds 1 GiB")
		}
		if len(b) == 0 {
			if c == ' ' || c == '\t' || c == '\n' || c == '\r' {
				continue
			}
			if c != '{' {
				return nil, errors.New("expected export object")
			}
		}
		if len(b) >= MaxPageBytes {
			return nil, limit("page exceeds 16 MiB; export smaller pages")
		}
		b = append(b, c)
		done, err := s.step(c)
		if err != nil {
			return nil, err
		}
		if done {
			return b, nil
		}
	}
}

// pageScanner tracks JSON strings and nesting to find where a page ends.
type pageScanner struct {
	depth           int
	quoted, escaped bool
}

// step reports whether c closes the top-level object.
func (s *pageScanner) step(c byte) (bool, error) {
	if s.quoted {
		switch {
		case s.escaped:
			s.escaped = false
		case c == '\\':
			s.escaped = true
		case c == '"':
			s.quoted = false
		}
		return false, nil
	}
	switch c {
	case '"':
		s.quoted = true
	case '{', '[':
		s.depth++
		if s.depth > MaxDepth {
			return false, limit("JSON nesting exceeds 64")
		}
	case '}', ']':
		s.depth--
		return s.depth == 0, nil
	}
	return false, nil
}

// pageHead is the validated envelope shared by every page of one export.
type pageHead struct {
	stream, bh, gh  string
	boundary        []byte
	maxSeq, count   int64
	fence, from, to Checkpoint
}

// legacyPos holds a page's pre-chain positions.
type legacyPos struct{ fromSeq, toSeq, fromCount, toCount int64 }

// pageRows holds a page's chained and pre-chain events.
type pageRows struct {
	events, legacy []any
	pos            legacyPos
}

func (v *Verifier) page(m map[string]any) error {
	h, err := readHead(m)
	if err != nil {
		return err
	}
	fi, _ := decimal(h.fence.Index, false)
	if err := v.begin(h, fi); err != nil {
		return err
	}
	if h.from != v.current {
		return errors.New("page checkpoint gap or reordering")
	}
	idx, _ := decimal(v.current.Index, false)
	if idx > fi {
		return errors.New("checkpoint exceeds fence")
	}
	rows, err := v.readRows(m)
	if err != nil {
		return err
	}
	switch m["segment"] {
	case "chain":
		if v.legacy || len(rows.legacy) != 0 || rows.pos != (legacyPos{}) || m["verificationStatus"] != "segment_consistent_unanchored" {
			return errors.New("invalid chain phase or legacy marker")
		}
		idx, err = v.chain(rows.events, idx, fi, h.stream)
	case "legacy":
		if idx != fi || v.current != h.fence || len(rows.events) != 0 || m["verificationStatus"] != "prechain_unverified" ||
			rows.pos.fromSeq != v.legacySeq || rows.pos.fromCount != v.legacyCount {
			return errors.New("legacy phase before fence or position mismatch")
		}
		err = v.legacyRows(rows.legacy, rows.pos)
	default:
		err = errors.New("unknown export segment")
	}
	if err != nil {
		return err
	}
	if h.to != v.current || (idx == fi && v.current != h.fence) {
		return errors.New("page to-checkpoint or fenced hash mismatch")
	}
	return v.completion(m, idx == fi, len(rows.legacy))
}

func readHead(m map[string]any) (pageHead, error) {
	h, err := readBoundary(m)
	if err != nil {
		return h, err
	}
	if h.fence, err = checkpoint(m["fence"]); err != nil {
		return h, fmt.Errorf("fence: %w", err)
	}
	if h.from, err = checkpoint(m["fromCheckpoint"]); err != nil {
		return h, fmt.Errorf("from checkpoint: %w", err)
	}
	if h.to, err = checkpoint(m["toCheckpoint"]); err != nil {
		return h, fmt.Errorf("to checkpoint: %w", err)
	}
	for _, cp := range []Checkpoint{h.fence, h.from, h.to} {
		if cp.StreamID != h.stream || cp.BoundaryHash != h.bh || (cp.Index == "0" && cp.Hash != h.gh) {
			return h, errors.New("checkpoint stream/boundary/genesis mismatch")
		}
	}
	return h, nil
}

func readBoundary(m map[string]any) (pageHead, error) {
	if !keys(m, pageFields) || !one(m["formatVersion"]) {
		return pageHead{}, errors.New("unsupported envelope schema/version")
	}
	bm, ok := m["boundary"].(map[string]any)
	if !ok || !keys(bm, boundaryFields) {
		return pageHead{}, errors.New("invalid boundary schema")
	}
	boundary, err := payload(bm["canonicalPayloadBase64"], MaxBoundaryBytes)
	if err != nil {
		return pageHead{}, fmt.Errorf("boundary payload: %w", err)
	}
	bc, err := parseObject(boundary)
	if err != nil {
		return pageHead{}, fmt.Errorf("canonical boundary: %w", err)
	}
	if !keys(bc, boundaryCanonicalFields) {
		return pageHead{}, errors.New("invalid canonical boundary object")
	}
	for _, k := range boundaryCanonicalFields {
		if !equal(bm[k], bc[k]) {
			return pageHead{}, fmt.Errorf("boundary field mismatch: %s", k)
		}
	}
	stream, _ := bm["streamId"].(string)
	maxSeq, e1 := decimal(bm["legacyMaxSeq"], false)
	count, e2 := decimal(bm["legacyCount"], false)
	if !uuid(stream) || m["streamId"] != stream || !timestamp(bm["createdAt"]) || e1 != nil || e2 != nil ||
		count > maxSeq || (count == 0) != (maxSeq == 0) {
		return pageHead{}, errors.New("invalid boundary fields")
	}
	bh, gh := digest(boundary), prefixedDigest(genesisPrefix, nil, boundary)
	if bm["boundaryHash"] != bh || bm["genesisHash"] != gh {
		return pageHead{}, errors.New("boundary/genesis hash mismatch")
	}
	return pageHead{stream: stream, bh: bh, gh: gh, boundary: boundary, maxSeq: maxSeq, count: count}, nil
}

// begin starts verification on the first page and checks that later pages
// keep the same boundary and fence.
func (v *Verifier) begin(h pageHead, fi int64) error {
	if v.initialized {
		if h.fence != v.fence || !bytes.Equal(h.boundary, v.boundaryPayload) {
			return errors.New("boundary or immutable fence changed between pages")
		}
		return nil
	}
	genesis := Checkpoint{StreamID: h.stream, Index: "0", Hash: h.gh, BoundaryHash: h.bh}
	if v.external != nil {
		ai, _ := decimal(v.external.Index, false)
		if v.external.StreamID != h.stream || v.external.BoundaryHash != h.bh || ai > fi {
			return errors.New("external checkpoint stream/boundary mismatch or fence precedes anchored tail")
		}
	}
	if h.from != genesis && (v.external == nil || h.from != *v.external) {
		return errors.New("export must start at genesis or exactly at external checkpoint")
	}
	v.initialized, v.start, v.current, v.fence = true, h.from, h.from, h.fence
	v.boundaryPayload, v.legacyMax, v.legacyTotal = h.boundary, h.maxSeq, h.count
	return v.encounter(h.from)
}

func (v *Verifier) readRows(m map[string]any) (pageRows, error) {
	events, ok := m["events"].([]any)
	lm, lok := m["legacy"].(map[string]any)
	if !ok || !lok || !keys(lm, legacyFields) || lm["status"] != "prechain_unverified" {
		return pageRows{}, errors.New("invalid events/legacy schema")
	}
	legacyEvents, ok := lm["events"].([]any)
	if !ok {
		return pageRows{}, errors.New("invalid legacy events array")
	}
	if len(events)+len(legacyEvents) > MaxPageEvents {
		return pageRows{}, limit("page contains more than 500 events; export smaller pages")
	}
	if v.events+v.legacyCount+int64(len(events)+len(legacyEvents)) > MaxEvents {
		return pageRows{}, limit("total events exceed 5000000")
	}
	var pos legacyPos
	var e1, e2, e3, e4 error
	pos.fromSeq, e1 = decimal(lm["fromSeq"], false)
	pos.toSeq, e2 = decimal(lm["toSeq"], false)
	pos.fromCount, e3 = decimal(lm["fromCount"], false)
	pos.toCount, e4 = decimal(lm["toCount"], false)
	if e1 != nil || e2 != nil || e3 != nil || e4 != nil {
		return pageRows{}, errors.New("invalid legacy positions")
	}
	return pageRows{events: events, legacy: legacyEvents, pos: pos}, nil
}

// chain verifies chained events after index idx and returns the new index.
func (v *Verifier) chain(events []any, idx, fi int64, stream string) (int64, error) {
	for _, raw := range events {
		if idx >= fi {
			return idx, errors.New("event exceeds fenced chain")
		}
		ei, err := v.chainEvent(raw, idx, stream)
		if err != nil {
			return idx, err
		}
		idx = ei
	}
	if len(events) == 0 && idx < fi {
		return idx, errors.New("empty chain page before fence")
	}
	return idx, nil
}

func (v *Verifier) chainEvent(raw any, idx int64, stream string) (int64, error) {
	e, err := event(raw, false)
	if err != nil {
		return 0, err
	}
	ei, _ := decimal(e["chainIndex"], true)
	seq, _ := decimal(e["seq"], true)
	if ei != idx+1 || e["streamId"] != stream || e["prevHash"] != v.current.Hash || seq <= v.legacyMax {
		return 0, errors.New("event index/stream/previous hash/legacy boundary mismatch")
	}
	canonical, err := payload(e["canonicalPayloadBase64"], MaxPayloadBytes)
	if err != nil {
		return 0, err
	}
	cm, err := parseObject(canonical)
	if err != nil {
		return 0, fmt.Errorf("canonical event: %w", err)
	}
	if !keys(cm, canonicalFields) {
		return 0, errors.New("invalid canonical event object")
	}
	for _, k := range canonicalFields {
		if !equal(cm[k], e[k]) {
			return 0, fmt.Errorf("canonical event field mismatch: %s", k)
		}
	}
	prev, _ := hex.DecodeString(v.current.Hash)
	if e["hash"] != prefixedDigest(eventPrefix, prev, canonical) {
		return 0, errors.New("event hash mismatch")
	}
	v.current.Index, v.current.Hash = strconv.FormatInt(ei, 10), e["hash"].(string)
	v.events++
	return ei, v.encounter(v.current)
}

func (v *Verifier) legacyRows(events []any, pos legacyPos) error {
	v.legacy = true
	for _, raw := range events {
		e, err := event(raw, true)
		if err != nil {
			return err
		}
		seq, _ := decimal(e["seq"], true)
		if seq <= v.legacySeq || seq > v.legacyMax || v.legacyCount >= v.legacyTotal {
			return errors.New("legacy event position/boundary mismatch")
		}
		v.legacySeq, v.legacyCount = seq, v.legacyCount+1
	}
	if pos.toSeq != v.legacySeq || pos.toCount != v.legacyCount {
		return errors.New("legacy to-position mismatch")
	}
	return nil
}

// completion checks the page's completion flags against verified progress.
func (v *Verifier) completion(m map[string]any, chainDone bool, legacyRows int) error {
	chainComplete, ok := m["chainComplete"].(bool)
	complete, cok := m["complete"].(bool)
	if !ok || !cok || chainComplete != chainDone || complete != (m["nextCursor"] == nil) {
		return errors.New("invalid completion flags")
	}
	if m["nextCursor"] != nil {
		s, ok := m["nextCursor"].(string)
		if !ok || s == "" || len(s) > 2048 {
			return errors.New("invalid next cursor")
		}
	}
	if complete {
		if !chainComplete || v.legacyCount != v.legacyTotal || v.legacySeq != v.legacyMax {
			return errors.New("premature completion: missing fenced chain or legacy rows")
		}
		v.complete = true
		return nil
	}
	if (v.legacy && (legacyRows == 0 || v.legacyCount >= v.legacyTotal)) || (chainComplete && v.legacyTotal == 0) {
		return errors.New("non-progressing or unnecessary continuation")
	}
	return nil
}

func (v *Verifier) encounter(c Checkpoint) error {
	if v.external != nil && c.Index == v.external.Index {
		if c != *v.external {
			return errors.New("external checkpoint hash mismatch")
		}
		v.anchorSeen = true
	}
	return nil
}

func event(raw any, legacy bool) (map[string]any, error) {
	m, ok := raw.(map[string]any)
	if !ok || !keys(m, eventFields) {
		return nil, errors.New("invalid event schema")
	}
	if valueSize(m) > MaxEntryBytes {
		return nil, limit("event decoded structural size exceeds 4 MiB")
	}
	if _, err := decimal(m["seq"], true); err != nil {
		return nil, errors.New("invalid event sequence")
	}
	id, _ := m["id"].(string)
	if !uuid(id) || !timestamp(m["occurredAt"]) {
		return nil, errors.New("invalid event ID/timestamp")
	}
	if err := eventStrings(m); err != nil {
		return nil, err
	}
	if legacy {
		for _, k := range []string{"streamId", "chainVersion", "chainIndex", "prevHash", "hash", "canonicalPayloadBase64"} {
			if m[k] != nil {
				return nil, errors.New("legacy event claims chain evidence")
			}
		}
		return m, nil
	}
	s, _ := m["streamId"].(string)
	_, err := decimal(m["chainIndex"], true)
	if !uuid(s) || !one(m["chainVersion"]) || err != nil || !hash(m["prevHash"]) || !hash(m["hash"]) {
		return nil, errors.New("invalid event integrity fields")
	}
	return m, nil
}

// eventStrings checks the UUID, nullable string and required string fields.
func eventStrings(m map[string]any) error {
	for _, k := range []string{"actorUserId", "resourceId", "companyId", "requestId"} {
		if m[k] != nil {
			s, ok := m[k].(string)
			if !ok || !uuid(s) {
				return fmt.Errorf("invalid event UUID: %s", k)
			}
		}
	}
	for _, k := range []string{"actorLogin", "actorDisplayName", "outcome", "errorCode", "clientIp", "sessionHash", "deviceHash"} {
		if m[k] != nil {
			if _, ok := m[k].(string); !ok {
				return fmt.Errorf("invalid nullable event string: %s", k)
			}
		}
	}
	for _, k := range []string{"action", "resourceType", "reason"} {
		if _, ok := m[k].(string); !ok {
			return fmt.Errorf("invalid event string: %s", k)
		}
	}
	return nil
}

func checkpoint(raw any) (Checkpoint, error) {
	m, ok := raw.(map[string]any)
	if !ok || !keys(m, checkpointFields) {
		return Checkpoint{}, errors.New("invalid checkpoint schema")
	}
	var c Checkpoint
	for k, dest := range map[string]*string{"streamId": &c.StreamID, "index": &c.Index, "hash": &c.Hash, "boundaryHash": &c.BoundaryHash} {
		s, ok := m[k].(string)
		if !ok {
			return c, errors.New("checkpoint fields must be strings")
		}
		*dest = s
	}
	return c, validCheckpoint(c)
}

func validCheckpoint(c Checkpoint) error {
	_, err := decimal(c.Index, false)
	if !uuid(c.StreamID) || !hash(c.Hash) || !hash(c.BoundaryHash) || err != nil {
		return errors.New("invalid checkpoint fields")
	}
	return nil
}

func decimal(v any, positive bool) (int64, error) {
	s, ok := v.(string)
	if !ok || s == "" || len(s) > 19 || (len(s) > 1 && s[0] == '0') {
		return 0, errors.New("expected int64 decimal string")
	}
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0, errors.New("expected unsigned decimal digits")
		}
	}
	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil || (positive && n == 0) {
		return 0, errors.New("decimal outside supported range")
	}
	return n, nil
}

func uuid(s string) bool {
	if len(s) != 36 {
		return false
	}
	for i, c := range s {
		if i == 8 || i == 13 || i == 18 || i == 23 {
			if c != '-' {
				return false
			}
		} else if !lowerHex(c) {
			return false
		}
	}
	return true
}

func lowerHex(c rune) bool { return c >= '0' && c <= '9' || c >= 'a' && c <= 'f' }

func hash(v any) bool {
	s, ok := v.(string)
	if !ok || len(s) != 64 {
		return false
	}
	for _, c := range s {
		if !lowerHex(c) {
			return false
		}
	}
	return true
}

func timestamp(v any) bool {
	s, ok := v.(string)
	if !ok || len(s) != len(timestampForm) {
		return false
	}
	t, err := time.Parse(timestampForm, s)
	return err == nil && t.Format(timestampForm) == s
}

func payload(v any, maxBytes int) ([]byte, error) {
	s, ok := v.(string)
	if !ok || s == "" {
		return nil, errors.New("missing canonical payload")
	}
	if len(s) > base64.StdEncoding.EncodedLen(maxBytes) {
		return nil, limit(fmt.Sprintf("canonical payload exceeds %d decoded bytes", maxBytes))
	}
	b, err := base64.StdEncoding.Strict().DecodeString(s)
	if len(b) > maxBytes {
		return nil, limit(fmt.Sprintf("canonical payload exceeds %d decoded bytes", maxBytes))
	}
	if err != nil || base64.StdEncoding.EncodeToString(b) != s {
		return nil, errors.New("noncanonical/invalid payload base64")
	}
	return b, nil
}

func digest(b []byte) string { h := sha256.Sum256(b); return hex.EncodeToString(h[:]) }

func prefixedDigest(prefix string, previous, b []byte) string {
	h := sha256.New()
	h.Write([]byte(prefix))
	h.Write(previous)
	h.Write(b)
	return hex.EncodeToString(h.Sum(nil))
}

func keys(m map[string]any, expected []string) bool {
	if len(m) != len(expected) {
		return false
	}
	for _, k := range expected {
		if _, ok := m[k]; !ok {
			return false
		}
	}
	return true
}

// parseObject rejects duplicate keys at every depth, including canonical JSON.
// UseNumber and normalized decimal comparison never convert through float64.
func parseObject(b []byte) (map[string]any, error) {
	if !utf8.Valid(b) || !validSurrogates(b) {
		return nil, errors.New("invalid Unicode JSON")
	}
	d := json.NewDecoder(bytes.NewReader(b))
	d.UseNumber()
	nodes := 0
	v, err := readValue(d, 0, &nodes)
	if err != nil {
		return nil, err
	}
	if _, err = d.Token(); !errors.Is(err, io.EOF) {
		return nil, errors.New("trailing JSON data")
	}
	m, ok := v.(map[string]any)
	if !ok {
		return nil, errors.New("expected JSON object")
	}
	return m, nil
}

// encoding/json replaces unpaired UTF-16 escapes with U+FFFD. Reject them so
// different malformed strings cannot silently become logically equal evidence.
func validSurrogates(b []byte) bool {
	quoted := false
	for i := 0; i < len(b); i++ {
		if b[i] == '"' {
			quoted = !quoted
			continue
		}
		if !quoted || b[i] != '\\' {
			continue
		}
		i++
		if i >= len(b) {
			return false
		}
		if b[i] != 'u' {
			continue
		}
		if i+4 >= len(b) {
			return false
		}
		x, err := strconv.ParseUint(string(b[i+1:i+5]), 16, 16)
		if err != nil {
			return false
		}
		i += 4
		if x >= 0xdc00 && x <= 0xdfff {
			return false
		}
		if x >= 0xd800 && x <= 0xdbff {
			if i+6 >= len(b) || b[i+1] != '\\' || b[i+2] != 'u' {
				return false
			}
			y, err := strconv.ParseUint(string(b[i+3:i+7]), 16, 16)
			if err != nil || y < 0xdc00 || y > 0xdfff {
				return false
			}
			i += 6
		}
	}
	return true
}

func readValue(d *json.Decoder, depth int, nodes *int) (any, error) {
	*nodes++
	if depth > MaxDepth || *nodes > MaxNodes {
		return nil, limit("JSON nesting exceeds 64 or node count exceeds 250000")
	}
	t, err := d.Token()
	if err != nil {
		return nil, errors.New("malformed JSON")
	}
	switch x := t.(type) {
	case json.Delim:
		switch x {
		case '{':
			return readObject(d, depth, nodes)
		case '[':
			return readArray(d, depth, nodes)
		default:
			return nil, errors.New("unexpected delimiter")
		}
	case json.Number:
		if _, _, _, err := normalizedNumber(x); err != nil {
			return nil, err
		}
	case string:
		if len(x) > MaxStringBytes {
			return nil, limit("JSON string exceeds 2 MiB")
		}
	}
	return t, nil
}

func readObject(d *json.Decoder, depth int, nodes *int) (map[string]any, error) {
	m := map[string]any{}
	for d.More() {
		kt, err := d.Token()
		k, ok := kt.(string)
		if err != nil || !ok {
			return nil, errors.New("invalid object key")
		}
		if len(k) > MaxStringBytes {
			return nil, limit("JSON key exceeds 2 MiB")
		}
		if _, exists := m[k]; exists {
			return nil, errors.New("duplicate JSON object key")
		}
		val, err := readValue(d, depth+1, nodes)
		if err != nil {
			return nil, err
		}
		m[k] = val
	}
	end, err := d.Token()
	if err != nil || end != json.Delim('}') {
		return nil, errors.New("malformed object")
	}
	return m, nil
}

func readArray(d *json.Decoder, depth int, nodes *int) ([]any, error) {
	a := []any{}
	for d.More() {
		val, err := readValue(d, depth+1, nodes)
		if err != nil {
			return nil, err
		}
		a = append(a, val)
	}
	end, err := d.Token()
	if err != nil || end != json.Delim(']') {
		return nil, errors.New("malformed array")
	}
	return a, nil
}

// normalizedNumber compares coefficient/exponent pairs without expanding an
// exponent into zeroes or allocating enormous big integers/rationals.
func normalizedNumber(n json.Number) (negative bool, digits string, exponent int64, err error) {
	s := string(n)
	if len(s) > MaxNumberBytes {
		return false, "", 0, limit("JSON number exceeds 4096 bytes")
	}
	negative = strings.HasPrefix(s, "-")
	s = strings.TrimPrefix(s, "-")
	if i := strings.IndexAny(s, "eE"); i >= 0 {
		exponent, err = strconv.ParseInt(s[i+1:], 10, 64)
		if err != nil || exponent < -MaxExponent || exponent > MaxExponent {
			return false, "", 0, limit("JSON numeric exponent outside -1000000..1000000")
		}
		s = s[:i]
	}
	if i := strings.IndexByte(s, '.'); i >= 0 {
		exponent -= int64(len(s) - i - 1)
		s = s[:i] + s[i+1:]
	}
	digits = strings.TrimLeft(s, "0")
	if digits == "" {
		return false, "0", 0, nil
	}
	trimmed := strings.TrimRight(digits, "0")
	exponent += int64(len(digits) - len(trimmed))
	return negative, trimmed, exponent, nil
}

func one(v any) bool {
	n, ok := v.(json.Number)
	if !ok {
		return false
	}
	negative, digits, exponent, err := normalizedNumber(n)
	return err == nil && !negative && digits == "1" && exponent == 0
}

func equal(a, b any) bool {
	switch x := a.(type) {
	case nil:
		return b == nil
	case string:
		y, ok := b.(string)
		return ok && x == y
	case bool:
		y, ok := b.(bool)
		return ok && x == y
	case json.Number:
		y, ok := b.(json.Number)
		if !ok {
			return false
		}
		xn, xd, xe, xerr := normalizedNumber(x)
		yn, yd, ye, yerr := normalizedNumber(y)
		return xerr == nil && yerr == nil && xn == yn && xd == yd && xe == ye
	case []any:
		y, ok := b.([]any)
		if !ok || len(x) != len(y) {
			return false
		}
		for i := range x {
			if !equal(x[i], y[i]) {
				return false
			}
		}
		return true
	case map[string]any:
		y, ok := b.(map[string]any)
		if !ok || len(x) != len(y) {
			return false
		}
		for k, v := range x {
			w, exists := y[k]
			if !exists || !equal(v, w) {
				return false
			}
		}
		return true
	}
	return false
}

func valueSize(v any) int {
	switch x := v.(type) {
	case string:
		return len(x) + 2
	case json.Number:
		return len(x)
	case []any:
		n := 2
		for _, val := range x {
			n += 1 + valueSize(val)
		}
		return n
	case map[string]any:
		n := 2
		for k, val := range x {
			n += len(k) + 4 + valueSize(val)
		}
		return n
	default:
		return 5
	}
}
