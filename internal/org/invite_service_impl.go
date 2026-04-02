package org

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"

	"github.com/google/uuid"
	"github.com/rian/infinite_brain/internal/auth"
	apperrors "github.com/rian/infinite_brain/pkg/errors"
)

type inviteServiceImpl struct {
	invites InviteRepository
	orgs    Repository
}

// NewInviteService returns an InviteService backed by the given repositories.
func NewInviteService(invites InviteRepository, orgs Repository) InviteService {
	return &inviteServiceImpl{invites: invites, orgs: orgs}
}

func (s *inviteServiceImpl) CreateInvite(ctx context.Context, orgID uuid.UUID, email, role string, callerID uuid.UUID) (*Invite, error) {
	if err := validateRole(role); err != nil {
		return nil, err
	}
	if err := s.requireManageMembers(ctx, orgID, callerID); err != nil {
		return nil, err
	}
	token, err := generateInviteToken()
	if err != nil {
		return nil, fmt.Errorf("generating invite token: %w", err)
	}
	inv := &Invite{
		OrgID:     orgID,
		Email:     email,
		Role:      role,
		InvitedBy: callerID,
		Token:     token,
	}
	created, err := s.invites.Create(ctx, inv)
	if err != nil {
		return nil, fmt.Errorf("create invite: %w", err)
	}
	return created, nil
}

func (s *inviteServiceImpl) AcceptInvite(ctx context.Context, token string, userID uuid.UUID) error {
	inv, err := s.invites.FindByToken(ctx, token)
	if err != nil {
		return fmt.Errorf("find invite by token: %w", err)
	}
	callerRef := inv.InvitedBy
	if err := s.orgs.AddMember(ctx, inv.OrgID, userID, inv.Role, &callerRef); err != nil {
		return fmt.Errorf("adding member via invite: %w", err)
	}
	if err := s.invites.Accept(ctx, inv.ID); err != nil {
		return fmt.Errorf("marking invite accepted: %w", err)
	}
	return nil
}

func (s *inviteServiceImpl) requireManageMembers(ctx context.Context, orgID, callerID uuid.UUID) error {
	member, err := s.orgs.FindMember(ctx, orgID, callerID)
	if err != nil {
		return apperrors.ErrForbidden.Wrap(fmt.Errorf("caller not a member"))
	}
	if !auth.Can(member.Role, auth.PermManageMembers) {
		return apperrors.ErrForbidden.Wrap(fmt.Errorf("role %q cannot manage members", member.Role))
	}
	return nil
}

func generateInviteToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("reading random bytes: %w", err)
	}
	return base64.URLEncoding.WithPadding(base64.NoPadding).EncodeToString(b), nil
}
