package watcher

import "strings"

// buildUsername generates a deterministic(결정론적) DB username from the secret name.
// Format: karden_{secret-name} with hyphens replaced by underscores.
// Truncated to 32 chars to satisfy MySQL's username length limit.
func buildUsername(secretName string) string {
	normalized := strings.ReplaceAll(secretName, "-", "_")
	username := "karden_" + normalized

	if len(username) > 32 {
		username = username[:32]
	}

	return username
}
