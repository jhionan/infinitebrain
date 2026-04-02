package auth_test

import (
	"testing"

	"github.com/rian/infinite_brain/internal/auth"
)

func TestCan_PermissionMatrix(t *testing.T) { //nolint:funlen // 60 role×perm test cases: exhaustive coverage requires this length
	tests := []struct {
		role string
		perm auth.Permission
		want bool
	}{
		// owner — can do everything
		{"owner", auth.PermReadNode, true},
		{"owner", auth.PermCreateNode, true},
		{"owner", auth.PermEditOwnNode, true},
		{"owner", auth.PermEditAnyNode, true},
		{"owner", auth.PermDeleteOwnNode, true},
		{"owner", auth.PermDeleteAnyNode, true},
		{"owner", auth.PermManageMembers, true},
		{"owner", auth.PermChangeRoles, true},
		{"owner", auth.PermViewAuditLog, true},
		{"owner", auth.PermManageSettings, true},
		{"owner", auth.PermBilling, true},
		{"owner", auth.PermDeleteOrg, true},
		{"owner", auth.PermTransferOwner, true},
		{"owner", auth.PermCreateToken, true},
		{"owner", auth.PermManageHoneypot, true},
		// admin — all but billing/delete/transfer
		{"admin", auth.PermReadNode, true},
		{"admin", auth.PermCreateNode, true},
		{"admin", auth.PermEditOwnNode, true},
		{"admin", auth.PermEditAnyNode, true},
		{"admin", auth.PermDeleteOwnNode, true},
		{"admin", auth.PermDeleteAnyNode, true},
		{"admin", auth.PermManageMembers, true},
		{"admin", auth.PermChangeRoles, true},
		{"admin", auth.PermViewAuditLog, true},
		{"admin", auth.PermManageSettings, true},
		{"admin", auth.PermBilling, false},
		{"admin", auth.PermDeleteOrg, false},
		{"admin", auth.PermTransferOwner, false},
		{"admin", auth.PermCreateToken, true},
		{"admin", auth.PermManageHoneypot, true},
		// editor — read/write own content + tokens
		{"editor", auth.PermReadNode, true},
		{"editor", auth.PermCreateNode, true},
		{"editor", auth.PermEditOwnNode, true},
		{"editor", auth.PermEditAnyNode, false},
		{"editor", auth.PermDeleteOwnNode, true},
		{"editor", auth.PermDeleteAnyNode, false},
		{"editor", auth.PermManageMembers, false},
		{"editor", auth.PermChangeRoles, false},
		{"editor", auth.PermViewAuditLog, false},
		{"editor", auth.PermManageSettings, false},
		{"editor", auth.PermBilling, false},
		{"editor", auth.PermDeleteOrg, false},
		{"editor", auth.PermTransferOwner, false},
		{"editor", auth.PermCreateToken, true},
		{"editor", auth.PermManageHoneypot, false},
		// viewer — read only
		{"viewer", auth.PermReadNode, true},
		{"viewer", auth.PermCreateNode, false},
		{"viewer", auth.PermEditOwnNode, false},
		{"viewer", auth.PermEditAnyNode, false},
		{"viewer", auth.PermDeleteOwnNode, false},
		{"viewer", auth.PermDeleteAnyNode, false},
		{"viewer", auth.PermManageMembers, false},
		{"viewer", auth.PermChangeRoles, false},
		{"viewer", auth.PermViewAuditLog, false},
		{"viewer", auth.PermManageSettings, false},
		{"viewer", auth.PermBilling, false},
		{"viewer", auth.PermDeleteOrg, false},
		{"viewer", auth.PermTransferOwner, false},
		{"viewer", auth.PermCreateToken, false},
		{"viewer", auth.PermManageHoneypot, false},
		// unknown role — all false
		{"unknown", auth.PermReadNode, false},
		{"unknown", auth.PermCreateNode, false},
	}
	for _, tt := range tests {
		t.Run(tt.role+"/"+string(tt.perm), func(t *testing.T) {
			got := auth.Can(tt.role, tt.perm)
			if got != tt.want {
				t.Errorf("Can(%q, %q) = %v, want %v", tt.role, tt.perm, got, tt.want)
			}
		})
	}
}

func TestPermissionsForRole_Owner_Returns15Perms(t *testing.T) {
	perms := auth.PermissionsForRole("owner")
	if len(perms) != 15 {
		t.Errorf("expected 15 permissions for owner, got %d", len(perms))
	}
}

func TestPermissionsForRole_Viewer_Returns1Perm(t *testing.T) {
	perms := auth.PermissionsForRole("viewer")
	if len(perms) != 1 {
		t.Errorf("expected 1 permission for viewer, got %d", len(perms))
	}
	if len(perms) > 0 && perms[0] != auth.PermReadNode {
		t.Errorf("expected PermReadNode, got %q", perms[0])
	}
}

func TestPermissionsForRole_UnknownRole_ReturnsEmpty(t *testing.T) {
	perms := auth.PermissionsForRole("superuser")
	if len(perms) != 0 {
		t.Errorf("expected 0 permissions for unknown role, got %d", len(perms))
	}
}
