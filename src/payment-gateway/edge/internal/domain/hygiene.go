package domain

import "strings"

var cardFields = map[string]struct{}{
	"card_number":        {},
	"number":             {},
	"pan":                {},
	"cvv":                {},
	"cvc":                {},
	"card_security_code": {},
	"card":               {},
	"expiry":             {},
}

func IsCardField(name string) bool {
	_, found := cardFields[strings.ToLower(name)]
	return found
}

// NormalizeCardNumber strips grouping characters; allow-list membership is decided only by the vault.
func NormalizeCardNumber(raw string) (string, bool) {
	var digits strings.Builder
	for _, character := range raw {
		switch {
		case character >= '0' && character <= '9':
			digits.WriteRune(character)
		case character == ' ' || character == '-':
		default:
			return "", false
		}
	}
	if digits.Len() < 12 || digits.Len() > 19 {
		return "", false
	}
	return digits.String(), true
}

func UserAgentFamily(userAgent string) string {
	switch {
	case strings.Contains(userAgent, "Edg/"), strings.Contains(userAgent, "EdgA/"), strings.Contains(userAgent, "EdgiOS/"):
		return "edge"
	case strings.Contains(userAgent, "Firefox/"), strings.Contains(userAgent, "FxiOS/"):
		return "firefox"
	case strings.Contains(userAgent, "Chrome/"), strings.Contains(userAgent, "CriOS/"):
		return "chrome"
	case strings.Contains(userAgent, "Safari/"):
		return "safari"
	default:
		return "other"
	}
}
