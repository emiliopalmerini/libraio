package config

import "os"

const DefaultVaultPath = "~/Documents/bag_of_holding"

// VaultPath returns the vault path from LIBRAIO_VAULT env var,
// falling back to DefaultVaultPath.
func VaultPath() string {
	if env := os.Getenv("LIBRAIO_VAULT"); env != "" {
		return env
	}
	return DefaultVaultPath
}
