package auth

// Permission is a capability string checked against a role.
type Permission string

// Permission constants — 15 actions covering the full RBAC matrix.
const (
	PermReadNode       Permission = "node:read"
	PermCreateNode     Permission = "node:create"
	PermEditOwnNode    Permission = "node:edit:own"
	PermEditAnyNode    Permission = "node:edit:any"
	PermDeleteOwnNode  Permission = "node:delete:own"
	PermDeleteAnyNode  Permission = "node:delete:any"
	PermManageMembers  Permission = "members:manage"
	PermChangeRoles    Permission = "members:roles"
	PermViewAuditLog   Permission = "audit:read"
	PermManageSettings Permission = "org:settings"
	PermBilling        Permission = "org:billing"
	PermDeleteOrg      Permission = "org:delete"
	PermTransferOwner  Permission = "org:transfer"
	PermCreateToken    Permission = "tokens:create"
	PermManageHoneypot Permission = "honeypot:manage"
)

var rolePermissions = map[string][]Permission{
	"owner": {
		PermReadNode, PermCreateNode, PermEditOwnNode, PermEditAnyNode,
		PermDeleteOwnNode, PermDeleteAnyNode, PermManageMembers, PermChangeRoles,
		PermViewAuditLog, PermManageSettings, PermBilling, PermDeleteOrg,
		PermTransferOwner, PermCreateToken, PermManageHoneypot,
	},
	"admin": {
		PermReadNode, PermCreateNode, PermEditOwnNode, PermEditAnyNode,
		PermDeleteOwnNode, PermDeleteAnyNode, PermManageMembers, PermChangeRoles,
		PermViewAuditLog, PermManageSettings, PermCreateToken, PermManageHoneypot,
	},
	"editor": {
		PermReadNode, PermCreateNode, PermEditOwnNode,
		PermDeleteOwnNode, PermCreateToken,
	},
	"viewer": {
		PermReadNode,
	},
}

// Can returns true if role has permission perm.
// Returns false for unknown roles or permissions not in the role's set.
func Can(role string, perm Permission) bool {
	perms, ok := rolePermissions[role]
	if !ok {
		return false
	}
	for _, p := range perms {
		if p == perm {
			return true
		}
	}
	return false
}

// PermissionsForRole returns all permissions for the given role.
// Returns an empty slice for unknown roles.
func PermissionsForRole(role string) []Permission {
	return rolePermissions[role]
}
