# T-158 — Bot AI Identity Disclosure

## Overview

EU AI Act Article 52(1): AI systems interacting with humans must disclose they are AI.
All Infinite Brain bots (Telegram, WhatsApp, Slack) must identify themselves as AI
on first interaction and whenever a user might reasonably believe they're talking to a human.

## Required Disclosure Points

1. **First message in any new conversation**
2. **Bot profile/bio** (Telegram and Slack support this)
3. **When the user directly asks** ("Are you a bot?", "Am I talking to a person?")
4. **When the bot makes a decision that affects the user** (e.g., classifying a note, prioritizing a task)

## Implementation Per Platform

### Telegram (T-070)

```go
// internal/bots/telegram/handler.go

func (h *TelegramHandler) HandleStart(ctx context.Context, msg *tgbotapi.Message) error {
    welcome := `Hello! I'm Infinite Brain, an AI assistant that helps you capture,
organize, and act on your knowledge.

I am an AI — not a human. I use Claude (Anthropic) to understand your messages
and help you build your second brain.

You can always ask me "why did you do that?" and I'll explain my reasoning.`

    return h.send(ctx, msg.Chat.ID, welcome)
}
```

Bot profile bio: "Infinite Brain AI — your personal knowledge assistant. I am an AI."

### WhatsApp (T-071)

First message template (WhatsApp Business API requires template approval):
```
Hi! I'm Infinite Brain, your AI knowledge assistant (not a human).
I'll help you capture and organize your thoughts. Reply with anything to get started.
```

### Slack (T-072)

Bot display name: "Infinite Brain AI"
App description: "Infinite Brain AI — automated knowledge assistant"
First DM: "Hi! I'm an AI assistant from Infinite Brain, not a human team member..."

## "Are you a bot?" Handler

All bots must handle direct questions about their nature:

```go
// pkg/bots/disclosure.go

var botIdentityTriggers = []string{
    "are you a bot", "are you human", "are you real", "am i talking to ai",
    "is this automated", "who am i talking to", "are you an ai",
}

func IsIdentityQuestion(text string) bool {
    lower := strings.ToLower(text)
    for _, trigger := range botIdentityTriggers {
        if strings.Contains(lower, trigger) {
            return true
        }
    }
    return false
}

func BotIdentityResponse(botName string) string {
    return fmt.Sprintf("Yes, I'm %s — an AI assistant, not a human. "+
        "I use Claude (by Anthropic) to understand your messages and help you "+
        "manage your knowledge. All my decisions can be explained — just ask 'why'.", botName)
}
```

## Acceptance Criteria

- [ ] All bots send AI disclosure on first message in a new conversation
- [ ] All bots respond to identity questions with clear AI confirmation
- [ ] Bot profile/bio on each platform identifies as AI
- [ ] Disclosure is in the user's language (i18n for at least EN/PT/ES/FR/DE)
- [ ] `IsIdentityQuestion` handles variations and common phrasings
- [ ] 90% test coverage on disclosure logic

## Dependencies

- T-070 (Telegram bot)
- T-071 (WhatsApp bot)
- T-072 (Slack bot)
- T-154 (EU AI Act — this implements Article 52(1))
