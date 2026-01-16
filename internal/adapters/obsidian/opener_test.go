package obsidian

import (
	"testing"
)

func TestNewOpener_DerivesVaultName(t *testing.T) {
	tests := []struct {
		name          string
		vaultPath     string
		wantVaultName string
	}{
		{
			name:          "simple vault path",
			vaultPath:     "/Users/test/MyVault",
			wantVaultName: "MyVault",
		},
		{
			name:          "vault with spaces",
			vaultPath:     "/Users/test/My Obsidian Vault",
			wantVaultName: "My Obsidian Vault",
		},
		{
			name:          "nested vault path",
			vaultPath:     "/Users/test/documents/notes/PersonalVault",
			wantVaultName: "PersonalVault",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opener := NewOpener(tt.vaultPath)
			if opener.vaultName != tt.wantVaultName {
				t.Errorf("vaultName = %q, want %q", opener.vaultName, tt.wantVaultName)
			}
		})
	}
}

func TestBuildURI(t *testing.T) {
	tests := []struct {
		name      string
		vaultPath string
		filePath  string
		wantURI   string
		wantErr   bool
	}{
		{
			name:      "simple file path",
			vaultPath: "/Users/test/MyVault",
			filePath:  "/Users/test/MyVault/S01.11.15 Theatre/README.md",
			wantURI:   "obsidian://open?vault=MyVault&file=S01.11.15%20Theatre%2FREADME.md",
			wantErr:   false,
		},
		{
			name:      "nested file path",
			vaultPath: "/Users/test/MyVault",
			filePath:  "/Users/test/MyVault/S01 Me/S01.10-19 Lifestyle/S01.11 Entertainment/S01.11.15 Theatre/README.md",
			wantURI:   "obsidian://open?vault=MyVault&file=S01%20Me%2FS01.10-19%20Lifestyle%2FS01.11%20Entertainment%2FS01.11.15%20Theatre%2FREADME.md",
			wantErr:   false,
		},
		{
			name:      "vault name with spaces",
			vaultPath: "/Users/test/My Vault",
			filePath:  "/Users/test/My Vault/notes/README.md",
			wantURI:   "obsidian://open?vault=My%20Vault&file=notes%2FREADME.md",
			wantErr:   false,
		},
		{
			name:      "file outside vault",
			vaultPath: "/Users/test/MyVault",
			filePath:  "/Users/test/OtherFolder/file.md",
			wantURI:   "",
			wantErr:   true,
		},
		{
			name:      "file at vault root",
			vaultPath: "/Users/test/MyVault",
			filePath:  "/Users/test/MyVault/README.md",
			wantURI:   "obsidian://open?vault=MyVault&file=README.md",
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opener := NewOpener(tt.vaultPath)
			gotURI, err := opener.BuildURI(tt.filePath)

			if (err != nil) != tt.wantErr {
				t.Errorf("BuildURI() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if gotURI != tt.wantURI {
				t.Errorf("BuildURI() = %q, want %q", gotURI, tt.wantURI)
			}
		})
	}
}
