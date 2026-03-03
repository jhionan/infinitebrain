# T-157 — Employment Use Prohibition

## Overview

IB's org intelligence tier (T-138–T-153) is designed for collective organizational intelligence,
NOT for employment decisions. Using it for hiring, firing, performance evaluation, or
worker monitoring would make IB a high-risk AI system under EU AI Act Annex III §4.

This task implements the technical and legal controls that keep IB out of high-risk territory.

## The Legal Risk

EU AI Act Annex III classifies as HIGH-RISK any AI system used for:
- Recruitment and CV screening
- Decisions on employment terms, promotion, or termination
- Task allocation and monitoring at work
- Evaluation and behavioral assessment of workers

If an employer deploys IB's org brain and uses it to evaluate employees → high-risk.
High-risk obligations include: conformity assessment, human oversight, bias testing,
registration in EU database, extensive documentation.

The anonymization architecture (T-146) and aggregated-only API (T-149) are the technical
defense. This task adds the enforcement layer.

## Technical Controls

### Org Admin Attestation on Setup

When an org activates the org intelligence tier, the admin must attest:

```go
// internal/org/onboarding.go

type OrgBrainAttestation struct {
    OrgID       uuid.UUID
    AdminUserID uuid.UUID
    SignedAt    time.Time
    // The admin attests to all of the following:
    Attestations []string // must include all required items
}

// Required attestations (all must be checked):
var RequiredAttestations = []string{
    "org_brain_not_used_for_employment_decisions",
    "org_brain_not_used_for_individual_performance_evaluation",
    "org_brain_not_used_for_hiring_or_termination",
    "employees_informed_of_knowledge_contribution",
    "individual_data_not_accessible_to_management",
}
```

The attestation is stored in the audit log (T-104). If a violation is later discovered,
this record shifts liability to the deployer.

### API-Level Blocks

The org intelligence API enforces at the service layer:

```go
// internal/org/intelligence_service.go

// OrgIntelligenceService rejects any query that could expose individual data.
func (s *OrgIntelligenceService) Query(ctx context.Context, q OrgQuery) (OrgInsight, error) {
    // Block: individual identification
    if q.RequestsIndividual() {
        return OrgInsight{}, apperrors.ErrForbidden.Wrap(
            "org intelligence queries must be aggregated; individual queries are not permitted")
    }
    // Block: results below k-anonymity threshold
    result, err := s.repo.AggregateForOrg(ctx, q)
    if result.ContributorCount < s.kAnonymityThreshold {
        return OrgInsight{}, apperrors.ErrInsufficientData
    }
    return result, nil
}
```

### Prohibited Query Detection

Any query that resembles an employment evaluation pattern is rejected:

```go
// Detected patterns that indicate employment evaluation intent:
// - "show me employee X's contributions"
// - "who has the lowest engagement in team Y"
// - "which employees are underperforming"
// - "rank employees by..."
// These are detected via keyword matching + intent classifier
```

## Legal Controls

### Terms of Service Clause

```
Section X — Prohibited Uses (Organizational Brain Features)

You may not use the Infinite Brain Organizational Intelligence features to:
(a) evaluate, score, or rank individual employees for employment purposes;
(b) support decisions regarding hiring, promotion, or termination;
(c) monitor individual employee productivity, engagement, or behavior;
(d) create profiles of individual employees for management review;
(e) comply with or evade any employment law obligation.

Violation of this section terminates your license to the Organizational Intelligence
features and may result in account suspension. You remain liable for any regulatory
obligations triggered by your prohibited use.
```

### Self-Hoster Liability Notice

In README and COMPLIANCE.md:

```
If you self-host Infinite Brain and deploy the Organizational Brain features for your
employees, you become a provider under EU AI Act Regulation 2024/1689. You are responsible
for ensuring your deployment complies with applicable AI regulations, including ensuring
you do not use these features for purposes that would classify your system as high-risk
under Annex III.
```

## Acceptance Criteria

- [ ] `OrgBrainAttestation` model + table
- [ ] Org admin must complete attestation before org intelligence features activate
- [ ] Attestation stored in audit log
- [ ] `OrgIntelligenceService` rejects individual queries at service layer
- [ ] k-anonymity threshold enforced (n≥5); configurable per deployment
- [ ] Prohibited query pattern detection (keyword + intent)
- [ ] ToS clause in `docs/TERMS.md`
- [ ] Self-hoster liability notice in README and COMPLIANCE.md
- [ ] 100% test coverage on prohibition enforcement (security-critical)

## Dependencies

- T-146 (org anonymization — technical foundation for this task)
- T-149 (org insights API — where enforcement runs)
- T-154 (EU AI Act — legal context and documentation)
- T-104 (audit log — attestation stored here)
