package tasks

import (
	"encoding/base64"
	"encoding/binary"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func testPublicKey(t *testing.T) string {
	t.Helper()
	blob := make([]byte, 4+len("ssh-ed25519")+4+32)
	binary.BigEndian.PutUint32(blob[:4], uint32(len("ssh-ed25519")))
	copy(blob[4:], "ssh-ed25519")
	offset := 4 + len("ssh-ed25519")
	binary.BigEndian.PutUint32(blob[offset:offset+4], 32)
	for index := 0; index < 32; index++ {
		blob[offset+4+index] = byte(index + 1)
	}
	return "ssh-ed25519 " + base64.StdEncoding.EncodeToString(blob) + " workstation"
}

func TestParseSSHEd25519PublicKey(t *testing.T) {
	publicKey, fingerprint, err := parseSSHEd25519PublicKey(testPublicKey(t))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(publicKey, " "+managedSSHComment) {
		t.Fatalf("public key does not have managed marker: %q", publicKey)
	}
	if !strings.HasPrefix(fingerprint, "SHA256:") {
		t.Fatalf("unexpected fingerprint: %q", fingerprint)
	}
}

func TestParseSSHEd25519PublicKeyRejectsUnsafeInput(t *testing.T) {
	inputs := []string{
		"",
		"-----BEGIN OPENSSH PRIVATE KEY-----",
		testPublicKey(t) + "\n" + testPublicKey(t),
		"command=whoami " + testPublicKey(t),
		"ssh-rsa AAAA invalid",
	}
	for _, input := range inputs {
		if _, _, err := parseSSHEd25519PublicKey(input); err == nil {
			t.Fatalf("expected input to be rejected: %q", input)
		}
	}
}

func TestReconcileManagedSSHKeyPreservesUnrelatedKeys(t *testing.T) {
	directory := t.TempDir()
	authorizedKeysPath := filepath.Join(directory, "authorized_keys")
	original := "# operator key\nssh-ed25519 AAAAoperator operator\n"
	if err := os.WriteFile(authorizedKeysPath, []byte(original), 0644); err != nil {
		t.Fatal(err)
	}

	publicKey, _, err := parseSSHEd25519PublicKey(testPublicKey(t))
	if err != nil {
		t.Fatal(err)
	}
	if err := reconcileManagedSSHKey(directory, publicKey); err != nil {
		t.Fatal(err)
	}
	if err := reconcileManagedSSHKey(directory, publicKey); err != nil {
		t.Fatal(err)
	}

	contents, err := os.ReadFile(authorizedKeysPath)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(string(contents), managedSSHComment) != 1 {
		t.Fatalf("managed key is not idempotent: %s", contents)
	}
	if !strings.Contains(string(contents), original) {
		t.Fatalf("unrelated content was changed: %s", contents)
	}
	if info, err := os.Stat(authorizedKeysPath); err != nil || info.Mode().Perm() != 0600 {
		t.Fatalf("authorized_keys mode is not 0600: %v, %v", info, err)
	}

	if err := reconcileManagedSSHKey(directory, ""); err != nil {
		t.Fatal(err)
	}
	contents, err = os.ReadFile(authorizedKeysPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(contents) != original {
		t.Fatalf("disable changed unrelated content: %q", contents)
	}
}

func TestReconcileManagedSSHKeyRejectsSymlink(t *testing.T) {
	directory := t.TempDir()
	target := filepath.Join(directory, "target")
	if err := os.WriteFile(target, []byte("preserve me"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, filepath.Join(directory, "authorized_keys")); err != nil {
		t.Fatal(err)
	}

	if err := reconcileManagedSSHKey(directory, ""); err == nil {
		t.Fatal("expected symlink to be rejected")
	}
}
