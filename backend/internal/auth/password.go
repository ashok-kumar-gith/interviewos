package auth

import (
	"bufio"
	_ "embed"
	"strings"
)

// minPasswordLength mirrors the handler-level `min=8` binding rule so the policy
// is enforced consistently at the service layer too (defense in depth).
const minPasswordLength = 8

// commonPasswordsRaw embeds a curated denylist of the most common / weak
// passwords (one per line, lowercased). New registrations and password resets
// are rejected when the chosen password appears here (NFR-SEC).
//
//go:embed common_passwords.txt
var commonPasswordsRaw string

// commonPasswords is the parsed denylist, built once at package init for O(1)
// membership checks.
var commonPasswords = parseCommonPasswords(commonPasswordsRaw)

// parseCommonPasswords parses the embedded denylist into a lookup set, skipping
// blank lines and `#` comments.
func parseCommonPasswords(raw string) map[string]struct{} {
	set := make(map[string]struct{}, 512)
	sc := bufio.NewScanner(strings.NewReader(raw))
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		set[strings.ToLower(line)] = struct{}{}
	}
	return set
}

// isCommonPassword reports whether the password is on the embedded denylist
// (case-insensitive).
func isCommonPassword(password string) bool {
	_, ok := commonPasswords[strings.ToLower(strings.TrimSpace(password))]
	return ok
}

// validatePassword enforces the service-layer password policy: a minimum length
// and rejection of common/weak passwords. It returns ErrPasswordTooShort or
// ErrPasswordTooCommon so the handler can map them to a 422 VALIDATION_ERROR.
func validatePassword(password string) error {
	if len(password) < minPasswordLength {
		return ErrPasswordTooShort
	}
	if isCommonPassword(password) {
		return ErrPasswordTooCommon
	}
	return nil
}
