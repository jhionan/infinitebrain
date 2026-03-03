# Feature: Focus Timer (ADHD Workflow Engine)

**Task ID**: T-040
**Status**: planned
**Epic**: ADHD Workflow Engine

## Goal

Implement a flexible Pomodoro-style focus timer tailored for ADHD users.
Sessions are linked to tasks, tracked for time analytics, and integrated with
Apple Watch notifications at key moments (start, break time, end).

## Acceptance Criteria

- [ ] `internal/adhd/timer_model.go` — FocusSession, TimerState structs
- [ ] `internal/adhd/timer_service.go` — FocusTimerService interface
- [ ] `internal/adhd/timer_service_impl.go` — Business logic
- [ ] `internal/adhd/timer_handler.go` — HTTP handlers
- [ ] Timer persisted in Redis (active session) + PostgreSQL (history)
- [ ] Apple Watch notification on session start, break prompt, time exceeded
- [ ] Configurable intervals (user can override default 25/5 pomodoro)
- [ ] Session linked to a specific task
- [ ] Time analytics per task and per day
- [ ] Unit tests for all service methods
- [ ] HTTP tests for all endpoints

## Data Model

```go
type FocusSession struct {
    ID          uuid.UUID
    UserID      uuid.UUID
    TaskID      *uuid.UUID     // which task this session is for
    StartedAt   time.Time
    PlannedMins int            // planned duration
    ActualMins  *int           // set when ended
    Status      SessionStatus  // running | paused | completed | abandoned
    Notes       string         // quick reflection on session end
    CreatedAt   time.Time
}

type SessionStatus string
const (
    StatusRunning   SessionStatus = "running"
    StatusPaused    SessionStatus = "paused"
    StatusCompleted SessionStatus = "completed"
    StatusAbandoned SessionStatus = "abandoned"
)
```

## API Endpoints

| Method | Path | Description |
|---|---|---|
| `POST` | `/api/v1/focus/start` | Start a focus session |
| `POST` | `/api/v1/focus/pause` | Pause current session |
| `POST` | `/api/v1/focus/resume` | Resume paused session |
| `POST` | `/api/v1/focus/end` | End session (with optional reflection) |
| `GET` | `/api/v1/focus/current` | Get current active session |
| `GET` | `/api/v1/focus/history` | List past sessions (paginated) |
| `GET` | `/api/v1/focus/stats` | Time analytics summary |

## ADHD-Specific Behaviours

### Hyperfocus Guard
If a session runs longer than `planned_mins * 1.5`, the system sends an
Apple Watch haptic and push notification: *"You've been focused for X minutes.
Time for a break!"*

### Break Reminder
After a completed session, a 5-minute break timer starts automatically.
Watch notification fires at end of break: *"Break over. Ready for the next session?"*

### Distraction Safety
Users can pause a session with a distraction note without losing progress.
The distraction is logged as a separate capture note tagged `#distraction`.

### Task Context
When a session starts, the app sends a watch notification with the task name
so the user always knows what they're supposed to be working on.

## Notification Triggers

| Event | Notification |
|---|---|
| Session start | "🎯 Starting: [Task Name] — 25 min session" |
| 5 min before end | "⏰ 5 minutes left on [Task Name]" |
| Session end | "✅ Session complete! Take a break." |
| Hyperfocus detected | "⚠️ You've been at this for [X] min. Take a break!" |
| Break end | "🚀 Break over. Ready to focus?" |
