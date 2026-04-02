package org

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// Org is the org domain model.
type Org struct {
	ID         uuid.UUID
	Name       string
	Slug       string
	Plan       string
	MaxMembers *int // nil = unlimited
	Settings   OrgSettings
	PhiEnabled bool
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// OrgSettings holds org-level configuration stored as JSONB.
type OrgSettings struct {
	AIProvider           string   `json:"ai_provider,omitempty"`
	MCPServerURL         string   `json:"mcp_server_url,omitempty"`
	AllowedDomains       []string `json:"allowed_domains,omitempty"`
	RequireMFA           bool     `json:"require_mfa,omitempty"`
	DataRetentionDays    int      `json:"data_retention_days,omitempty"`
	ChunkDefaultDuration int      `json:"chunk_default_duration,omitempty"`
	ReviewNotifications  string   `json:"review_notifications,omitempty"`
}

// Member is a user's membership in an org.
type Member struct {
	OrgID       uuid.UUID
	UserID      uuid.UUID
	Role        string
	InvitedBy   *uuid.UUID
	JoinedAt    time.Time
	// Email and DisplayName are only populated when retrieved via ListMembers (JOIN query).
	// FindMember does not populate these fields.
	Email       string
	DisplayName string
}

// MarshalSettings serialises OrgSettings to JSON for storage.
func MarshalSettings(s OrgSettings) ([]byte, error) {
	return json.Marshal(s)
}

// UnmarshalSettings deserialises JSONB bytes into OrgSettings.
func UnmarshalSettings(b []byte, s *OrgSettings) error {
	if len(b) == 0 {
		return nil
	}
	return json.Unmarshal(b, s)
}
