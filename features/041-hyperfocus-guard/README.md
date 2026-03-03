# Feature: Hyperfocus Guard

**Task ID**: T-041
**Status**: planned
**Epic**: ADHD Workflow Engine

## Goal

Detect and gently interrupt hyperfocus episodes — when a user has been working on
the same task for too long without breaks. This is a core ADHD safety feature.

## Acceptance Criteria

- [ ] Background worker checks active sessions every 5 minutes
- [ ] Configurable thresholds per user (default: warn at 90 min, escalate at 120 min)
- [ ] Notifications escalate in urgency (gentle → firm → urgent)
- [ ] Users can snooze guard (15/30 min options)
- [ ] Guard log stored for review in weekly digest
- [ ] Can be disabled per session ("deep work mode")
- [ ] Unit tests for threshold detection logic
- [ ] Integration tests for worker scheduling

## Guard Levels

| Level | Threshold | Notification | Action |
|---|---|---|---|
| Warning | 90 min | 🟡 "You've been at this a while" | Suggest break |
| Firm | 120 min | 🟠 "Seriously, take a break" | Offer to pause timer |
| Urgent | 150 min | 🔴 "This is a hyperfocus episode" | Auto-pause + calendar block |

## Business Rules

1. Guard is reset when user takes a break (≥ 5 minutes of no active session)
2. "Deep work mode" disables guard for a specified duration (max 3 hours)
3. Guard events are logged and surfaced in weekly reviews
4. Consecutive hyperfocus episodes in a day trigger a daily summary notification

## Background Worker

```
Asynq periodic job: adhd:check_focus (every 5 minutes)
  For each user with an active session:
    1. Calculate session duration
    2. Check against user thresholds
    3. If threshold exceeded and not snoozed:
       - Send notification
       - Log guard event
       - Escalate level if already warned
```
