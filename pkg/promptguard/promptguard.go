// Package promptguard detects and redacts prompt injection attempts
// from user-supplied content before it flows into AI prompts.
package promptguard

import (
	"regexp"

	"github.com/rs/zerolog"
)

// injectionPatterns covers common prompt injection techniques.
// Patterns are applied in order; all matches are replaced with [REDACTED].
var injectionPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)ignore\s+(previous|above|all)\s+instructions?`),
	regexp.MustCompile(`(?i)you\s+are\s+now\s+`),
	regexp.MustCompile(`(?i)system\s*:\s*`),
	regexp.MustCompile(`(?i)assistant\s*:\s*`),
	regexp.MustCompile(`(?i)forget\s+everything`),
	regexp.MustCompile(`(?i)new\s+instructions?\s*:`),
	regexp.MustCompile(`(?i)jailbreak`),
	regexp.MustCompile(`(?i)<\s*\|?\s*im_start\s*\|?\s*>`),
}

// Guard sanitizes user-provided content before it reaches an AI prompt.
type Guard struct {
	logger zerolog.Logger
}

// New creates a Guard. Inject zerolog.Nop() in tests.
func New(logger zerolog.Logger) *Guard {
	return &Guard{logger: logger}
}

// Sanitize scans content for injection patterns and replaces matches with [REDACTED].
// Returns the (possibly modified) content and a boolean indicating whether any
// changes were made. All 8 patterns are checked regardless of early matches.
func (g *Guard) Sanitize(content string) (string, bool) {
	modified := false
	for _, pattern := range injectionPatterns {
		if pattern.MatchString(content) {
			g.logger.Warn().
				Str("pattern", pattern.String()).
				Msg("prompt injection attempt detected")
			content = pattern.ReplaceAllString(content, "[REDACTED]")
			modified = true
		}
	}
	return content, modified
}
