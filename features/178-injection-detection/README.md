# T-178 — AI Behavioral Anomaly Detection

## Overview

PromptGuard (T-177) catches known injection patterns before the AI sees content.
This feature catches injections that *succeeded* — where the AI's behavior deviated
from expected in ways that suggest the content influenced its instructions.

Pattern matching is a whitelist approach. Anomaly detection is a behavioral approach.
Both are needed.

---

## What "Anomalous Behavior" Looks Like

A classification AI call should return a valid PARA category and up to 10 tags.
That's the entire expected output space. Any deviation is a signal:

| Anomaly | What injection it indicates |
|---|---|
| Output contains email addresses | Exfiltration attempt partially succeeded |
| Output contains URLs not in input | AI referencing external resources |
| Output JSON has extra fields | AI was instructed to include additional data |
| Tags contain long sentences | Tag field was used to embed extracted data |
| Classification changed for same content on retry | Context was poisoned |
| Output length >> expected (e.g. 2000 tokens for a classification) | AI including extra content |
| Canary phrase appears in output | System prompt was leaked — critical |
| Business rule created without user action | Persistence injection succeeded |

---

## Behavioral Baseline

For each AI operation type, IB maintains a statistical baseline:

```go
// internal/ai/anomaly/baseline.go

type OperationBaseline struct {
    Operation      string
    AvgOutputLen   float64
    StdOutputLen   float64
    ExpectedFields []string      // exact set of JSON fields expected
    MaxFieldLen    map[string]int // max character length per field
    AllowedValues  map[string][]string // enum fields → allowed values
}

var ClassificationBaseline = OperationBaseline{
    Operation:    "classify",
    AvgOutputLen: 120,  // characters
    StdOutputLen: 40,
    ExpectedFields: []string{"para", "tags"},
    MaxFieldLen:    map[string]int{"para": 20, "tags": 500},
    AllowedValues:  map[string][]string{"para": {"project", "area", "resource", "archive"}},
}
```

An output that falls outside `avg ± 3*std` on any dimension is flagged as anomalous.

---

## Canary Phrase Monitoring

System prompt canary phrases (T-177) are checked on every AI output.
Canary detection is the highest-confidence signal: it means the AI exposed
information from the system prompt context — injection succeeded in extracting privileged data.

```go
func (d *AnomalyDetector) CheckOutput(ctx context.Context, op AIOperation, output string) AnomalyResult {
    // Canary check first — highest priority
    if d.canaries.FoundIn(output) {
        return AnomalyResult{
            Detected:  true,
            Severity:  "critical",
            Type:      "canary_leaked",
            Action:    ActionBlockAndAlert,
        }
    }

    // Schema validation
    if violations := d.baseline.Validate(op, output); len(violations) > 0 {
        return AnomalyResult{
            Detected:  true,
            Severity:  d.scoreSeverity(violations),
            Type:      "schema_violation",
            Violations: violations,
            Action:    ActionFlagForReview,
        }
    }

    // Statistical anomaly
    if d.isStatisticalAnomaly(op, output) {
        return AnomalyResult{
            Detected: true,
            Severity: "medium",
            Type:     "statistical_anomaly",
            Action:   ActionFlagForReview,
        }
    }

    return AnomalyResult{Detected: false}
}
```

---

## Source Reputation Tracking

Sources that repeatedly trigger detections are tracked:

```go
type SourceReputation struct {
    SourceID        string     // email address, IP, webhook ID
    DetectionCount  int
    LastDetectedAt  time.Time
    QuarantinedAt   *time.Time
    TrustScore      float64    // 1.0 = fully trusted, 0.0 = blocked
}
```

When `DetectionCount >= threshold` (default: 3 in 7 days):
- Source is quarantined: content still ingested but held for human review
- Human reviews the queue: approve (process normally) or reject (discard + block source)
- Approved after quarantine: trust score restored
- Rejected repeatedly: source permanently blocked + security_incident logged

---

## Acceptance Criteria

- [ ] `AnomalyDetector` checks every AI output before it is applied
- [ ] Canary phrase detection on all outputs — critical incident on match
- [ ] Schema + field length validation per operation type
- [ ] Statistical anomaly detection with configurable thresholds
- [ ] Source reputation tracking + automatic quarantine
- [ ] All anomalies logged to audit log with output hash (not content)
- [ ] Quarantine queue with human review workflow
- [ ] Dashboard: anomaly rate per source, per operation type over time
- [ ] 100% test coverage on detection paths

## Dependencies

- T-177 (PromptGuard — pre-processing; this is post-processing)
- T-104 (security incidents + audit log)
- T-098 (security hardening)
