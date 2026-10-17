package handler

import (
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"strconv"
	"time"

	"justixauto/internal/modules/identity/model"
	"justixauto/internal/modules/identity/service"
)

// revision formats an optimistic-locking version for an ETag/If-Match value.
func revision(v int64) string { return strconv.FormatInt(v, 10) }

// Response DTOs. Models never go to JSON directly, so internal fields such as
// password hashes or login counters cannot leak.

type companyDTO struct {
	ID                  string                    `json:"id"`
	Kind                model.CompanyKind         `json:"kind"`
	Name                string                    `json:"name"`
	LegalName           string                    `json:"legalName"`
	Country             service.Label             `json:"country"`
	Region              *service.Label            `json:"region"`
	Registration        string                    `json:"registration"`
	LegalForm           model.LegalForm           `json:"legalForm"`
	TaxID               string                    `json:"taxId"`
	BankRequisites      string                    `json:"bankRequisites"`
	Email               string                    `json:"email"`
	Address             string                    `json:"address"`
	Phone               string                    `json:"phone"`
	Access              model.CompanyAccess       `json:"access"`
	AccessReason        string                    `json:"accessReason"`
	UserType            *model.SellerUserType     `json:"userType"`
	Verification        model.CompanyVerification `json:"verification"`
	VerificationReason  string                    `json:"verificationReason"`
	VerificationActorID *string                   `json:"verificationActorId"`
	VerificationAt      *time.Time                `json:"verificationAt"`
	Revision            string                    `json:"revision"`
	CreatedAt           time.Time                 `json:"createdAt"`
	UpdatedAt           time.Time                 `json:"updatedAt"`
}

func toCompany(c *model.Company) companyDTO {
	d := companyDTO{
		ID: c.ID, Kind: c.Kind, Name: c.Name, LegalName: c.LegalName,
		Country: service.Label{Key: c.CountryKey, Label: c.Country}, Registration: c.RegistrationNumber,
		LegalForm: c.LegalForm, TaxID: c.TaxID, BankRequisites: c.BankRequisites,
		Email: c.Email, Address: c.Address, Phone: c.Phone, Access: c.Status, AccessReason: c.StatusReason,
		UserType: c.UserType, Verification: c.Verification, VerificationReason: c.VerificationReason,
		VerificationActorID: c.VerificationActorID, VerificationAt: c.VerificationAt,
		Revision: revision(c.Version), CreatedAt: c.CreatedAt, UpdatedAt: c.UpdatedAt,
	}
	if c.Region != "" {
		d.Region = &service.Label{Key: c.RegionKey, Label: c.Region}
	}
	return d
}

type userDTO struct {
	ID           string            `json:"id"`
	DisplayName  string            `json:"displayName"`
	Email        string            `json:"email"`
	Login        *string           `json:"login"`
	Status       model.UserStatus  `json:"status"`
	StatusReason string            `json:"statusReason"`
	Roles        []service.RoleRef `json:"roles"`
	CompanyIDs   []string          `json:"companyIds"` // active memberships
	Revision     string            `json:"revision"`
	CreatedAt    time.Time         `json:"createdAt"`
}

func toUser(d *service.UserDetail) userDTO {
	u := d.User
	out := userDTO{
		ID: u.ID, DisplayName: u.DisplayName, Email: u.Email, Login: u.Login, Status: u.Status,
		StatusReason: u.StatusReason, Roles: make([]service.RoleRef, len(d.Roles)), CompanyIDs: d.CompanyIDs,
		Revision: revision(u.Version), CreatedAt: u.CreatedAt,
	}
	for i, r := range d.Roles {
		out.Roles[i] = service.RoleRef{ID: r.ID, Name: r.Name}
	}
	return out
}

type roleDTO struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	System bool   `json:"system"`
	// Scope follows from the permissions: "platform" (staff) or "company".
	Scope          string   `json:"scope"`
	PermissionKeys []string `json:"permissionKeys"`
	Revision       string   `json:"revision"`
}

func toRole(r *model.Role) roleDTO {
	perms := r.Permissions // services fill built-in roles' grants
	if perms == nil {
		perms = []string{}
	}
	return roleDTO{ID: r.ID, Name: r.Name, System: r.System(), Scope: r.Scope, PermissionKeys: perms, Revision: revision(r.Version)}
}

// permissionDTO is one permission of the catalog in PostgreSQL.
type permissionDTO struct {
	Key        string `json:"key"`
	Scope      string `json:"scope"`
	Name       string `json:"name"`
	Assignable bool   `json:"assignable"`
}

func toPermission(p *model.Permission) permissionDTO {
	return permissionDTO{Key: p.Key, Scope: p.Scope, Name: p.Name, Assignable: p.Assignable}
}

type membershipDTO struct {
	ID           string                    `json:"id"`
	UserID       string                    `json:"userId"`
	CompanyID    string                    `json:"companyId"`
	Status       model.MembershipStatus    `json:"status"`
	BranchAccess service.BranchAccessInput `json:"branchAccess"`
	StatusReason string                    `json:"statusReason"`
	Revision     string                    `json:"revision"`
}

func toMembership(m *model.Membership) membershipDTO {
	ids := m.BranchIDs
	if ids == nil {
		ids = []string{}
	}
	return membershipDTO{
		ID: m.ID, UserID: m.UserID, CompanyID: m.CompanyID, Status: m.Status,
		BranchAccess: service.BranchAccessInput{Mode: m.BranchAccess, BranchIDs: ids}, StatusReason: m.StatusReason,
		Revision: revision(m.Version),
	}
}

type branchDTO struct {
	ID        string `json:"id"`
	CompanyID string `json:"companyId"`
	Name      string `json:"name"`
	Address   string `json:"address"`
	Revision  string `json:"revision"`
}

func toBranch(b *model.Branch) branchDTO {
	return branchDTO{ID: b.ID, CompanyID: b.CompanyID, Name: b.Name, Address: b.Address, Revision: revision(b.Version)}
}

type auditDTO struct {
	ID               string            `json:"id"`
	OccurredAt       time.Time         `json:"occurredAt"`
	ActorID          *string           `json:"actorId"`
	Action           string            `json:"action"`
	ResourceType     string            `json:"resourceType"`
	ResourceID       *string           `json:"resourceId"`
	CompanyID        *string           `json:"companyId"`
	Reason           string            `json:"reason"`
	Details          json.RawMessage   `json:"details"`
	ActorLogin       *string           `json:"actorLogin"`
	ActorDisplayName *string           `json:"actorDisplayName"`
	Outcome          *string           `json:"outcome"`
	ErrorCode        *string           `json:"errorCode"`
	RequestID        *string           `json:"requestId"`
	ClientIP         *string           `json:"clientIp"`
	SessionHash      *string           `json:"sessionHash"`
	DeviceHash       *string           `json:"deviceHash"`
	Metadata         json.RawMessage   `json:"metadata"`
	Before           json.RawMessage   `json:"before"`
	After            json.RawMessage   `json:"after"`
	Integrity        auditIntegrityDTO `json:"integrity"`
}

type auditIntegrityDTO struct {
	Version  *int16  `json:"version"`
	StreamID *string `json:"streamId"`
	Index    *string `json:"index"`
	PrevHash *string `json:"prevHash"`
	Hash     *string `json:"hash"`
	Status   string  `json:"status"`
}

func auditDecimal(v *int64) *string {
	if v == nil {
		return nil
	}
	s := strconv.FormatInt(*v, 10)
	return &s
}

func auditHex(v []byte) *string {
	if len(v) == 0 {
		return nil
	}
	s := hex.EncodeToString(v)
	return &s
}

func auditStream(e *model.AuditEvent) *string {
	if e.ChainIndex == nil {
		return nil
	}
	return &e.StreamID
}

func toAudit(e *model.AuditEvent) auditDTO {
	return auditDTO{
		ID: e.ID, OccurredAt: e.OccurredAt.UTC(), ActorID: e.ActorUserID, Action: e.Action,
		ResourceType: e.ResourceType, ResourceID: e.ResourceID, CompanyID: e.CompanyID, Reason: e.Reason,
		Details:    json.RawMessage(e.Details),
		ActorLogin: e.ActorLogin, ActorDisplayName: e.ActorDisplayName, Outcome: e.Outcome, ErrorCode: e.ErrorCode,
		RequestID: e.RequestID, ClientIP: e.ClientIP, SessionHash: e.SessionHash, DeviceHash: e.DeviceHash,
		Metadata: json.RawMessage(e.Metadata), Before: json.RawMessage(e.BeforeValues), After: json.RawMessage(e.AfterValues),
		Integrity: toAuditIntegrity(e),
	}
}

func toAuditIntegrity(e *model.AuditEvent) auditIntegrityDTO {
	status := "prechain_unverified"
	if e.ChainIndex != nil {
		status = "chained_unverified"
	}
	return auditIntegrityDTO{Version: e.ChainVersion, StreamID: auditStream(e), Index: auditDecimal(e.ChainIndex), PrevHash: auditHex(e.PrevHash), Hash: auditHex(e.EventHash), Status: status}
}

// Export exposes exactly the canonical logical fields, without compatibility
// aliases or duplicated integrity properties that could diverge in handoff.
type auditExportEventDTO struct {
	StreamID               *string         `json:"streamId"`
	ChainVersion           *int16          `json:"chainVersion"`
	ChainIndex             *string         `json:"chainIndex"`
	Seq                    string          `json:"seq"`
	ID                     string          `json:"id"`
	OccurredAt             string          `json:"occurredAt"`
	ActorUserID            *string         `json:"actorUserId"`
	ActorLogin             *string         `json:"actorLogin"`
	ActorDisplayName       *string         `json:"actorDisplayName"`
	Action                 string          `json:"action"`
	ResourceType           string          `json:"resourceType"`
	ResourceID             *string         `json:"resourceId"`
	CompanyID              *string         `json:"companyId"`
	Reason                 string          `json:"reason"`
	Outcome                *string         `json:"outcome"`
	ErrorCode              *string         `json:"errorCode"`
	ClientIP               *string         `json:"clientIp"`
	Details                json.RawMessage `json:"details"`
	Before                 json.RawMessage `json:"before"`
	After                  json.RawMessage `json:"after"`
	Metadata               json.RawMessage `json:"metadata"`
	RequestID              *string         `json:"requestId"`
	SessionHash            *string         `json:"sessionHash"`
	DeviceHash             *string         `json:"deviceHash"`
	PrevHash               *string         `json:"prevHash"`
	Hash                   *string         `json:"hash"`
	CanonicalPayloadBase64 *string         `json:"canonicalPayloadBase64"`
}

const auditTimeLayout = "2006-01-02T15:04:05.000000Z"

func toAuditExportEvent(e *model.AuditEvent) auditExportEventDTO {
	var payload *string
	if e.ChainIndex != nil {
		s := base64.StdEncoding.EncodeToString(e.CanonicalPayload)
		payload = &s
	}
	return auditExportEventDTO{
		StreamID: auditStream(e), ChainVersion: e.ChainVersion, ChainIndex: auditDecimal(e.ChainIndex), Seq: strconv.FormatInt(e.Seq, 10),
		ID: e.ID, OccurredAt: e.OccurredAt.UTC().Format(auditTimeLayout), ActorUserID: e.ActorUserID,
		ActorLogin: e.ActorLogin, ActorDisplayName: e.ActorDisplayName, Action: e.Action, ResourceType: e.ResourceType,
		ResourceID: e.ResourceID, CompanyID: e.CompanyID, Reason: e.Reason, Outcome: e.Outcome, ErrorCode: e.ErrorCode,
		ClientIP: e.ClientIP, Details: json.RawMessage(e.Details), Before: json.RawMessage(e.BeforeValues), After: json.RawMessage(e.AfterValues),
		Metadata: json.RawMessage(e.Metadata), RequestID: e.RequestID, SessionHash: e.SessionHash, DeviceHash: e.DeviceHash,
		PrevHash: auditHex(e.PrevHash), Hash: auditHex(e.EventHash), CanonicalPayloadBase64: payload,
	}
}

type auditCheckpointDTO struct {
	StreamID     string `json:"streamId"`
	Index        string `json:"index"`
	Hash         string `json:"hash"`
	BoundaryHash string `json:"boundaryHash"`
}

func toAuditCheckpoint(c service.AuditCheckpoint) auditCheckpointDTO {
	return auditCheckpointDTO{StreamID: c.StreamID, Index: strconv.FormatInt(c.Index, 10), Hash: hex.EncodeToString(c.Hash), BoundaryHash: hex.EncodeToString(c.BoundaryHash)}
}

type auditBoundaryDTO struct {
	StreamID               string `json:"streamId"`
	LegacyMaxSeq           string `json:"legacyMaxSeq"`
	LegacyCount            string `json:"legacyCount"`
	CreatedAt              string `json:"createdAt"`
	CanonicalPayloadBase64 string `json:"canonicalPayloadBase64"`
	GenesisHash            string `json:"genesisHash"`
	BoundaryHash           string `json:"boundaryHash"`
}

type auditLegacyExportDTO struct {
	Status    string                `json:"status"`
	FromSeq   string                `json:"fromSeq"`
	ToSeq     string                `json:"toSeq"`
	FromCount string                `json:"fromCount"`
	ToCount   string                `json:"toCount"`
	Events    []auditExportEventDTO `json:"events"`
}

type auditExportDTO struct {
	FormatVersion      int                   `json:"formatVersion"`
	StreamID           string                `json:"streamId"`
	Segment            string                `json:"segment"`
	VerificationStatus string                `json:"verificationStatus"`
	Boundary           auditBoundaryDTO      `json:"boundary"`
	Fence              auditCheckpointDTO    `json:"fence"`
	FromCheckpoint     auditCheckpointDTO    `json:"fromCheckpoint"`
	ToCheckpoint       auditCheckpointDTO    `json:"toCheckpoint"`
	Events             []auditExportEventDTO `json:"events"`
	Legacy             auditLegacyExportDTO  `json:"legacy"`
	NextCursor         *string               `json:"nextCursor"`
	ChainComplete      bool                  `json:"chainComplete"`
	Complete           bool                  `json:"complete"`
}

func toAuditExport(p *service.AuditExport) auditExportDTO {
	h := p.Chain
	status := "segment_consistent_unanchored"
	if p.Segment == "legacy" {
		status = "prechain_unverified"
	}
	return auditExportDTO{
		FormatVersion: 1, StreamID: h.StreamID, Segment: p.Segment, VerificationStatus: status,
		Boundary: auditBoundaryDTO{StreamID: h.StreamID, LegacyMaxSeq: strconv.FormatInt(h.LegacyMaxSeq, 10), LegacyCount: strconv.FormatInt(h.LegacyCount, 10),
			CreatedAt: h.CreatedAt.UTC().Format(auditTimeLayout), CanonicalPayloadBase64: base64.StdEncoding.EncodeToString(h.BoundaryPayload),
			GenesisHash: hex.EncodeToString(h.GenesisHash), BoundaryHash: hex.EncodeToString(p.Fence.BoundaryHash)},
		Fence: toAuditCheckpoint(p.Fence), FromCheckpoint: toAuditCheckpoint(p.From), ToCheckpoint: toAuditCheckpoint(p.To),
		Events: mapSlice(p.Events, toAuditExportEvent),
		Legacy: auditLegacyExportDTO{Status: "prechain_unverified", FromSeq: strconv.FormatInt(p.LegacyFromSeq, 10), ToSeq: strconv.FormatInt(p.LegacyToSeq, 10),
			FromCount: strconv.FormatInt(p.LegacyFromCount, 10), ToCount: strconv.FormatInt(p.LegacyToCount, 10), Events: mapSlice(p.LegacyEvents, toAuditExportEvent)},
		NextCursor: p.NextCursor, ChainComplete: p.To.Index == p.Fence.Index, Complete: p.NextCursor == nil,
	}
}

func mapSlice[T, D any](items []T, f func(*T) D) []D {
	out := make([]D, len(items))
	for i := range items {
		out[i] = f(&items[i])
	}
	return out
}
