package service

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/netip"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/google/uuid"

	"justixauto/internal/modules/identity/model"
	"justixauto/internal/pkg/apperr"
	"justixauto/internal/pkg/auth"
)

func NewAudit(d Deps) *Audit { return &Audit{d} }

type Audit struct{ Deps }

type AuditPage struct {
	Events     []model.AuditEvent
	NextCursor *string
}

type AuditCheckpoint struct {
	StreamID     string
	Index        int64
	Hash         []byte
	BoundaryHash []byte
}

type AuditExport struct {
	Chain           *model.AuditChain
	Fence           AuditCheckpoint
	From            AuditCheckpoint
	To              AuditCheckpoint
	Events          []model.AuditEvent
	LegacyEvents    []model.AuditEvent
	Segment         string
	NextCursor      *string
	LegacyFromSeq   int64
	LegacyToSeq     int64
	LegacyFromCount int64
	LegacyToCount   int64
}

// These cursors are bounded, versioned view coordinates, not credentials or
// signed attestations. Coordinates are checked against the current stream;
// possession cannot expand the audit permission or bypass any filter.
type auditCursor struct {
	Version      int    `json:"v"`
	Mode         string `json:"mode"`
	StreamID     string `json:"streamId"`
	BoundaryHash string `json:"boundaryHash"`
	FenceIndex   int64  `json:"fenceIndex,string"`
	FenceHash    string `json:"fenceHash"`
	LegacyMaxSeq int64  `json:"legacyMaxSeq,string"`
	LegacyCount  int64  `json:"legacyCount,string"`
	Phase        string `json:"phase"`
	AfterIndex   int64  `json:"afterIndex,string"`
	AfterSeq     int64  `json:"afterSeq,string"`
	AfterHash    string `json:"afterHash"`
	LegacyRead   int64  `json:"legacyRead,string"`
	Filters      string `json:"filters"`
}

func auditDigest(b []byte) string { h := sha256.Sum256(b); return hex.EncodeToString(h[:]) }

func auditIntegrityError() error {
	return apperr.New(apperr.ErrConflict, "audit_integrity_mismatch", "audit evidence does not match its stream; export stopped")
}

func normalizeAuditFilter(f model.AuditFilter, export bool) (model.AuditFilter, error) {
	var v apperr.Validation
	for field, id := range map[string]*string{"resourceId": &f.ResourceID, "actorId": &f.ActorID} {
		if *id != "" {
			parsed, err := uuid.Parse(*id)
			if err != nil {
				v.Add(field, "must be a valid ID")
			} else {
				*id = parsed.String()
			}
		}
	}
	for field, val := range map[string]string{"actor": f.Actor, "action": f.Action, "resourceType": f.ResourceType} {
		if !utf8.ValidString(val) || len(val) > 128 || strings.IndexFunc(val, unicode.IsControl) >= 0 {
			v.Add(field, "must be at most 128 UTF-8 bytes without control characters")
		}
	}
	if f.IP != "" {
		ip, err := netip.ParseAddr(f.IP)
		if err != nil || ip.Zone() != "" {
			v.Add("ip", "must be an IP address without a prefix or zone")
		} else {
			f.IP = ip.Unmap().String()
		}
	}
	maxLimit := 100
	if export {
		maxLimit = 500
	}
	if f.Limit < 0 || f.Limit > maxLimit {
		v.Add("limit", fmt.Sprintf("must be between 1 and %d", maxLimit))
	}
	if f.Limit == 0 {
		f.Limit = 50
	}
	if f.Offset < 0 {
		v.Add("offset", "must not be negative")
	}
	if f.Cursor != "" && f.Offset != 0 {
		v.Add("offset", "cannot be combined with cursor")
	}
	if len(f.Cursor) > 2048 {
		v.Add("cursor", "must be at most 2048 bytes")
	}
	if export && (f.Offset != 0 || f.ResourceType != "" || f.ResourceID != "" || f.ActorID != "" || f.Actor != "" || f.Action != "" || f.IP != "") {
		v.Add("filters", "export requires the complete stream; filters and offset are not supported")
	}
	return f, v.Err()
}

func auditFilterHash(f model.AuditFilter) string {
	b, _ := json.Marshal([]string{f.ResourceType, f.ResourceID, f.ActorID, f.Actor, f.IP, f.Action})
	return auditDigest(b)
}

func encodeAuditCursor(c auditCursor) *string {
	b, _ := json.Marshal(c)
	s := base64.RawURLEncoding.EncodeToString(b)
	return &s
}

func errAuditCursor() error {
	return apperr.FieldError("cursor", "invalid or changed audit view; start a new view")
}

func (s *Audit) readCursor(ctx context.Context, f model.AuditFilter, mode string, head *model.AuditChain) (auditCursor, error) {
	if f.Cursor == "" {
		return auditCursor{
			Version: 1, Mode: mode, StreamID: head.StreamID, BoundaryHash: auditDigest(head.BoundaryPayload),
			FenceIndex: head.LastIndex, FenceHash: hex.EncodeToString(head.LastHash), LegacyMaxSeq: head.LegacyMaxSeq,
			LegacyCount: head.LegacyCount, Phase: "chain", Filters: auditFilterHash(f),
		}, nil
	}
	c, ok := decodeAuditCursor(f.Cursor)
	if !ok || !validAuditCursor(c, f, mode, head) {
		return auditCursor{}, errAuditCursor()
	}
	fenceHash := head.GenesisHash
	switch {
	case c.FenceIndex == head.LastIndex:
		fenceHash = head.LastHash
	case c.FenceIndex > 0:
		h, err := s.cursorEventHash(ctx, c.FenceIndex)
		if err != nil {
			return auditCursor{}, err
		}
		fenceHash = h
	}
	if len(fenceHash) != 32 || c.FenceHash != hex.EncodeToString(fenceHash) {
		return auditCursor{}, errAuditCursor()
	}
	if mode != "export" {
		if c.AfterHash != "" {
			return auditCursor{}, errAuditCursor()
		}
		return c, nil
	}
	if c.Phase == "legacy" && (c.AfterIndex != c.FenceIndex || c.AfterHash != c.FenceHash) {
		return auditCursor{}, errAuditCursor()
	}
	afterHash := head.GenesisHash
	if c.AfterIndex > 0 {
		h, err := s.cursorEventHash(ctx, c.AfterIndex)
		if err != nil {
			return auditCursor{}, err
		}
		afterHash = h
	}
	if len(afterHash) != 32 || c.AfterHash != hex.EncodeToString(afterHash) {
		return auditCursor{}, errAuditCursor()
	}
	return c, nil
}

// decodeAuditCursor strictly decodes a canonical base64url JSON cursor.
func decodeAuditCursor(raw string) (auditCursor, bool) {
	b, err := base64.RawURLEncoding.Strict().DecodeString(raw)
	if err != nil || base64.RawURLEncoding.EncodeToString(b) != raw {
		return auditCursor{}, false
	}
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.DisallowUnknownFields()
	var c auditCursor
	if dec.Decode(&c) != nil || !errors.Is(dec.Decode(new(any)), io.EOF) {
		return auditCursor{}, false
	}
	return c, true
}

// validAuditCursor checks a decoded cursor against the current view.
func validAuditCursor(c auditCursor, f model.AuditFilter, mode string, head *model.AuditChain) bool {
	return c.Version == 1 && c.Mode == mode && c.StreamID == head.StreamID && c.BoundaryHash == auditDigest(head.BoundaryPayload) &&
		c.LegacyMaxSeq == head.LegacyMaxSeq && c.LegacyCount == head.LegacyCount && c.Filters == auditFilterHash(f) &&
		c.FenceIndex >= 0 && c.FenceIndex <= head.LastIndex && c.AfterIndex >= 0 && c.AfterIndex <= c.FenceIndex &&
		c.AfterSeq >= 0 && (c.Phase == "chain" || c.Phase == "legacy") &&
		(c.Phase != "legacy" || c.AfterSeq <= c.LegacyMaxSeq) &&
		(c.Phase != "chain" || c.AfterSeq == 0) &&
		c.LegacyRead >= 0 && c.LegacyRead <= c.LegacyCount && (c.Phase != "chain" || c.LegacyRead == 0) &&
		(mode != "list" || (c.LegacyRead == 0 && (c.Phase != "chain" || c.AfterIndex != 0) && (c.Phase != "legacy" || c.AfterIndex == 0))) &&
		(c.Phase != "legacy" || mode != "list" || c.AfterSeq != 0)
}

// cursorEventHash returns the hash of the chained event at index; a missing
// event means the cursor no longer matches the chain.
func (s *Audit) cursorEventHash(ctx context.Context, index int64) ([]byte, error) {
	e, err := s.store.Audit().GetByIndex(ctx, index)
	if errors.Is(err, apperr.ErrNotFound) {
		return nil, errAuditCursor()
	}
	if err != nil {
		return nil, err
	}
	return e.EventHash, nil
}

func (s *Audit) ListPage(ctx context.Context, actor *auth.Principal, f model.AuditFilter) (*AuditPage, error) {
	if err := actor.Allow(model.PermPlatformAuditRead); err != nil {
		return nil, err
	}
	return s.listPage(ctx, f)
}

func (s *Audit) listPage(ctx context.Context, f model.AuditFilter) (*AuditPage, error) {
	f, err := normalizeAuditFilter(f, false)
	if err != nil {
		return nil, err
	}
	head, err := s.store.Audit().Chain(ctx)
	if err != nil {
		return nil, err
	}
	c, err := s.readCursor(ctx, f, "list", head)
	if err != nil {
		return nil, err
	}
	w := model.AuditWindow{StreamID: c.StreamID, FenceIndex: c.FenceIndex, LegacyMaxSeq: c.LegacyMaxSeq, Limit: f.Limit + 1, Offset: f.Offset}
	if f.Cursor != "" {
		w.After = &model.AuditOrder{Seq: c.AfterSeq}
		if c.Phase == "chain" {
			w.After.ChainIndex = &c.AfterIndex
		}
	}
	rows, err := s.store.Audit().Page(ctx, f, w)
	if err != nil {
		return nil, err
	}
	out := &AuditPage{Events: []model.AuditEvent{}}
	more := len(rows) > f.Limit
	if more {
		rows = rows[:f.Limit]
	}
	for _, row := range rows {
		e := row.AuditEvent
		e.StreamID = head.StreamID
		out.Events = append(out.Events, e)
	}
	if more {
		last := out.Events[len(out.Events)-1]
		c.AfterIndex, c.AfterSeq = 0, 0
		if last.ChainIndex == nil {
			c.Phase, c.AfterSeq = "legacy", last.Seq
		} else {
			c.Phase, c.AfterIndex = "chain", *last.ChainIndex
		}
		out.NextCursor = encodeAuditCursor(c)
	}
	return out, nil
}

func (s *Audit) Get(ctx context.Context, actor *auth.Principal, id string) (*model.AuditEvent, error) {
	if err := actor.Allow(model.PermPlatformAuditRead); err != nil {
		return nil, err
	}
	parsed, err := uuid.Parse(id)
	if err != nil {
		return nil, apperr.FieldError("id", "must be a valid ID")
	}
	e, err := s.store.Audit().Get(ctx, parsed.String())
	if err != nil {
		return nil, err
	}
	head, err := s.store.Audit().Chain(ctx)
	if err != nil {
		return nil, err
	}
	e.StreamID = head.StreamID
	return e, nil
}

// Export emits complete-stream pages. Chained rows ascend by committed index;
// the separate legacy segment never implies hash or externally anchored coverage.
func (s *Audit) Export(ctx context.Context, actor *auth.Principal, f model.AuditFilter) (*AuditExport, error) {
	if err := actor.Allow(model.PermPlatformAuditRead); err != nil {
		return nil, err
	}
	f, err := normalizeAuditFilter(f, true)
	if err != nil {
		return nil, err
	}
	head, err := s.store.Audit().Chain(ctx)
	if err != nil {
		return nil, err
	}
	if !validAuditHead(head) {
		return nil, auditIntegrityError()
	}
	c, err := s.readCursor(ctx, f, "export", head)
	if err != nil {
		return nil, err
	}
	if f.Cursor == "" {
		c.AfterHash = hex.EncodeToString(head.GenesisHash)
	}
	if c.FenceIndex == 0 {
		c.Phase = "legacy"
	}
	boundaryHash := sha256.Sum256(head.BoundaryPayload)
	checkpoint := func(index int64, hash []byte) AuditCheckpoint {
		return AuditCheckpoint{StreamID: c.StreamID, Index: index, Hash: hash, BoundaryHash: boundaryHash[:]}
	}
	fenceHash, _ := hex.DecodeString(c.FenceHash)
	afterHash, _ := hex.DecodeString(c.AfterHash)
	out := &AuditExport{
		Chain: head, Fence: checkpoint(c.FenceIndex, fenceHash), From: checkpoint(c.AfterIndex, afterHash),
		To: checkpoint(c.AfterIndex, afterHash), Events: []model.AuditEvent{}, LegacyEvents: []model.AuditEvent{}, Segment: c.Phase,
	}
	out.LegacyFromSeq, out.LegacyToSeq = c.AfterSeq, c.AfterSeq
	out.LegacyFromCount, out.LegacyToCount = c.LegacyRead, c.LegacyRead
	rows, err := s.store.Audit().Page(ctx, f, exportWindow(c, f.Limit+1))
	if err != nil {
		return nil, err
	}
	more := len(rows) > f.Limit
	if more {
		rows = rows[:f.Limit]
	}
	for _, row := range rows {
		e := row.AuditEvent
		e.StreamID = head.StreamID
		if c.Phase == "legacy" {
			err = appendLegacyRow(out, &c, e)
		} else {
			err = appendChainRow(out, &c, e, row.ReconstructedPayload)
		}
		if err != nil {
			return nil, err
		}
	}
	if c.Phase == "chain" && !more {
		if out.To.Index != c.FenceIndex || !bytes.Equal(out.To.Hash, fenceHash) {
			return nil, auditIntegrityError()
		}
		if c.LegacyCount > 0 {
			c.Phase, c.AfterSeq, more = "legacy", 0, true
		}
	}
	if out.Segment == "legacy" && (!more && c.LegacyRead != c.LegacyCount || c.LegacyRead > c.LegacyCount) {
		return nil, auditIntegrityError()
	}
	if more {
		out.NextCursor = encodeAuditCursor(c)
	}
	return out, nil
}

// validAuditHead checks the chain head's boundary, genesis and last link.
func validAuditHead(head *model.AuditChain) bool {
	genesis := sha256.Sum256(append([]byte("justixauto-audit-genesis-v1\n"), head.BoundaryPayload...))
	return bytes.Equal(head.BoundaryPayload, head.ReconstructedBoundary) && bytes.Equal(genesis[:], head.GenesisHash) &&
		head.ChainVersion == 1 && head.LastIndex >= 0 && len(head.LastHash) == 32 && head.LegacyCount >= 0 && head.LegacyMaxSeq >= 0 &&
		(head.LastIndex != 0 || bytes.Equal(head.LastHash, head.GenesisHash))
}

// exportWindow selects the rows after the cursor in its current phase.
func exportWindow(c auditCursor, limit int) model.AuditWindow {
	w := model.AuditWindow{
		StreamID: c.StreamID, FenceIndex: c.FenceIndex, LegacyMaxSeq: c.LegacyMaxSeq, Limit: limit,
		Ascending: true, LegacyOnly: c.Phase == "legacy", Reconstruct: c.Phase == "chain",
	}
	if c.Phase == "chain" && c.AfterIndex > 0 {
		afterIndex := c.AfterIndex
		w.After = &model.AuditOrder{ChainIndex: &afterIndex}
	}
	if c.Phase == "legacy" && c.AfterSeq > 0 {
		w.After = &model.AuditOrder{Seq: c.AfterSeq}
	}
	return w
}

// appendLegacyRow checks that a pre-chain row carries no chain evidence and
// follows the cursor, then adds it to the export.
func appendLegacyRow(out *AuditExport, c *auditCursor, e model.AuditEvent) error {
	if e.ChainIndex != nil || e.ChainVersion != nil || len(e.PrevHash) != 0 || len(e.EventHash) != 0 ||
		len(e.CanonicalPayload) != 0 || e.Seq <= c.AfterSeq || e.Seq > c.LegacyMaxSeq {
		return auditIntegrityError()
	}
	out.LegacyEvents = append(out.LegacyEvents, e)
	c.AfterSeq = e.Seq
	c.LegacyRead++
	out.LegacyToSeq, out.LegacyToCount = c.AfterSeq, c.LegacyRead
	return nil
}

// appendChainRow checks that a chained row links to the previous one and
// matches its hash, then adds it to the export.
func appendChainRow(out *AuditExport, c *auditCursor, e model.AuditEvent, reconstructed []byte) error {
	if e.ChainIndex == nil || *e.ChainIndex != out.To.Index+1 || e.ChainVersion == nil || *e.ChainVersion != 1 ||
		!bytes.Equal(e.CanonicalPayload, reconstructed) || len(e.CanonicalPayload) == 0 || !bytes.Equal(e.PrevHash, out.To.Hash) {
		return auditIntegrityError()
	}
	h := sha256.New()
	h.Write([]byte("justixauto-audit-v1\n"))
	h.Write(e.PrevHash)
	h.Write(e.CanonicalPayload)
	if !bytes.Equal(h.Sum(nil), e.EventHash) {
		return auditIntegrityError()
	}
	out.Events = append(out.Events, e)
	out.To = AuditCheckpoint{StreamID: c.StreamID, Index: *e.ChainIndex, Hash: e.EventHash, BoundaryHash: out.To.BoundaryHash}
	c.AfterIndex, c.AfterHash = *e.ChainIndex, hex.EncodeToString(e.EventHash)
	return nil
}
