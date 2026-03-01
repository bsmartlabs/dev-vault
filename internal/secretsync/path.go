package secretsync

import "github.com/bsmartlabs/dev-vault/internal/pathpolicy"

func resolveFile(rootDir string, rel string) (string, error) {
	return pathpolicy.ResolveProjectFile(rootDir, rel)
}
