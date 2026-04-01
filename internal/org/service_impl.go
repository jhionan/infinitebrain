package org

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	apperrors "github.com/rian/infinite_brain/pkg/errors"
)

type serviceImpl struct {
	repo Repository
}

// NewService returns an org Service backed by the given repository.
func NewService(repo Repository) Service {
	return &serviceImpl{repo: repo}
}

func (s *serviceImpl) Get(ctx context.Context, slug string) (*Org, error) {
	return s.repo.FindBySlug(ctx, slug)
}

func (s *serviceImpl) Update(ctx context.Context, orgID, callerID uuid.UUID, name string, settings OrgSettings) (*Org, error) {
	if err := s.requireRole(ctx, orgID, callerID, "admin"); err != nil {
		return nil, err
	}
	if name == "" {
		return nil, apperrors.ErrValidation.Wrap(fmt.Errorf("org name cannot be empty"))
	}
	return s.repo.Update(ctx, orgID, name, settings)
}

func (s *serviceImpl) Delete(ctx context.Context, orgID, callerID uuid.UUID) error {
	if err := s.requireRole(ctx, orgID, callerID, "owner"); err != nil {
		return err
	}
	return s.repo.SoftDelete(ctx, orgID)
}

func (s *serviceImpl) ListMembers(ctx context.Context, orgID uuid.UUID) ([]Member, error) {
	return s.repo.ListMembers(ctx, orgID)
}

func (s *serviceImpl) AddMember(ctx context.Context, orgID, userID uuid.UUID, role string, inviterID uuid.UUID) error {
	if err := validateRole(role); err != nil {
		return err
	}
	o, err := s.repo.FindByID(ctx, orgID)
	if err != nil {
		return fmt.Errorf("get org for add member: %w", err)
	}
	count, err := s.repo.CountMembers(ctx, orgID)
	if err != nil {
		return fmt.Errorf("count members: %w", err)
	}
	// Per-org MaxMembers overrides the plan-level cap when set.
	if o.MaxMembers != nil {
		if int(count) >= *o.MaxMembers {
			return apperrors.ErrPlanLimitReached.Wrap(
				fmt.Errorf("org member limit of %d reached", *o.MaxMembers),
			)
		}
	} else if err := CheckMemberLimit(o.Plan, count); err != nil {
		return err
	}
	return s.repo.AddMember(ctx, orgID, userID, role, &inviterID)
}

func (s *serviceImpl) UpdateMemberRole(ctx context.Context, orgID, targetUserID, callerID uuid.UUID, role string) error {
	if err := validateRole(role); err != nil {
		return err
	}
	if err := s.requireRole(ctx, orgID, callerID, "admin"); err != nil {
		return err
	}
	// Prevent demoting the last owner.
	target, err := s.repo.FindMember(ctx, orgID, targetUserID)
	if err != nil {
		return fmt.Errorf("find target member: %w", err)
	}
	if target.Role == "owner" && role != "owner" {
		if err := s.ensureNotLastOwner(ctx, orgID, targetUserID); err != nil {
			return err
		}
	}
	return s.repo.UpdateMemberRole(ctx, orgID, targetUserID, role)
}

func (s *serviceImpl) RemoveMember(ctx context.Context, orgID, targetUserID, callerID uuid.UUID) error {
	if targetUserID != callerID {
		if err := s.requireRole(ctx, orgID, callerID, "admin"); err != nil {
			return err
		}
	}
	target, err := s.repo.FindMember(ctx, orgID, targetUserID)
	if err != nil {
		return fmt.Errorf("find target member: %w", err)
	}
	if target.Role == "owner" {
		if err := s.ensureNotLastOwner(ctx, orgID, targetUserID); err != nil {
			return err
		}
	}
	return s.repo.RemoveMember(ctx, orgID, targetUserID)
}

// ── internal helpers ──────────────────────────────────────────────────────────

// requireRole verifies callerID has at least the given role in orgID.
// Role order: owner > admin > editor > viewer > member.
func (s *serviceImpl) requireRole(ctx context.Context, orgID, callerID uuid.UUID, minRole string) error {
	member, err := s.repo.FindMember(ctx, orgID, callerID)
	if err != nil {
		if errors.Is(err, apperrors.ErrNotFound) {
			return apperrors.ErrForbidden.Wrap(fmt.Errorf("not a member of this org"))
		}
		return fmt.Errorf("find caller member: %w", err)
	}
	if !hasAtLeastRole(member.Role, minRole) {
		return apperrors.ErrForbidden.Wrap(fmt.Errorf("requires %s role, caller has %s", minRole, member.Role))
	}
	return nil
}

// ensureNotLastOwner returns an error if targetUserID is the only owner.
func (s *serviceImpl) ensureNotLastOwner(ctx context.Context, orgID, targetUserID uuid.UUID) error {
	members, err := s.repo.ListMembers(ctx, orgID)
	if err != nil {
		return fmt.Errorf("list members for owner check: %w", err)
	}
	ownerCount := 0
	for _, m := range members {
		if m.Role == "owner" {
			ownerCount++
		}
	}
	if ownerCount <= 1 {
		return apperrors.ErrValidation.Wrap(fmt.Errorf("cannot remove or demote the last owner"))
	}
	return nil
}

var roleRank = map[string]int{
	"owner":  5,
	"admin":  4,
	"editor": 3,
	"viewer": 2,
	"member": 1,
}

func hasAtLeastRole(actual, required string) bool {
	return roleRank[actual] >= roleRank[required]
}

var validRoles = map[string]bool{
	"owner": true, "admin": true, "editor": true, "viewer": true, "member": true,
}

func validateRole(role string) error {
	if !validRoles[role] {
		return apperrors.ErrValidation.Wrap(fmt.Errorf("invalid role %q", role))
	}
	return nil
}
