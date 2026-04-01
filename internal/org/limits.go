package org

import (
	"fmt"

	apperrors "github.com/rian/infinite_brain/pkg/errors"
)

// OrgLimits defines resource caps for a plan tier.
// Zero means unlimited.
type OrgLimits struct {
	MaxMembers      int
	MaxNodes        int
	AICallsPerMonth int
	MaxProjects     int
}

var planLimits = map[string]OrgLimits{
	"personal":   {MaxMembers: 1, MaxNodes: 10_000, AICallsPerMonth: 500, MaxProjects: 5},
	"pro":        {MaxMembers: 1, MaxNodes: 0, AICallsPerMonth: 0, MaxProjects: 0},
	"teams":      {MaxMembers: 25, MaxNodes: 0, AICallsPerMonth: 0, MaxProjects: 0},
	"enterprise": {MaxMembers: 0, MaxNodes: 0, AICallsPerMonth: 0, MaxProjects: 0},
}

// LimitsFor returns the limits for the given plan.
// Unknown plans fall back to personal limits (most restrictive).
func LimitsFor(plan string) OrgLimits {
	if l, ok := planLimits[plan]; ok {
		return l
	}
	return planLimits["personal"]
}

// CheckMemberLimit returns ErrPlanLimitReached if adding one more member would
// exceed the plan's member cap. Zero MaxMembers means unlimited.
func CheckMemberLimit(plan string, current int64) error {
	limits := LimitsFor(plan)
	if limits.MaxMembers == 0 {
		return nil
	}
	if int(current) >= limits.MaxMembers {
		return apperrors.ErrPlanLimitReached.Wrap(
			fmt.Errorf("plan %q allows max %d members", plan, limits.MaxMembers),
		)
	}
	return nil
}
