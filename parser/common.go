package parser

import (
	"strings"
)

func extractBetweenKeywords(text, startWord, stopWord string) string {
	start := strings.Index(strings.ToLower(text), strings.ToLower(startWord))
	if start == -1 {
		return ""
	}
	sub := text[start+len(startWord):]
	end := strings.Index(strings.ToLower(sub), strings.ToLower(stopWord))
	if end != -1 {
		sub = sub[:end]
	}
	return strings.TrimSpace(sub)
}
