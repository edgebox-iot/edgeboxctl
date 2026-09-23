package tasks

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/edgebox-iot/edgeboxctl/internal/utils"
)

const (
	sshDirectory      = "/root/.ssh"
	managedSSHComment = "edgebox-dashboard-managed"
)

type sshAccessResult struct {
	Status      string `json:"status"`
	PublicKey   string `json:"public_key,omitempty"`
	Fingerprint string `json:"fingerprint,omitempty"`
}

func taskEnableSSHAccess(args taskEnableSSHAccessArgs) (string, error) {
	if err := verifyRootSSHAccess(); err != nil {
		return "", err
	}

	publicKey, fingerprint, err := parseSSHEd25519PublicKey(args.PublicKey)
	if err != nil {
		return "", err
	}

	if err := reconcileManagedSSHKey(sshDirectory, publicKey); err != nil {
		return "", err
	}

	utils.WriteOption("SSH_ACCESS_ENABLED", "true")
	utils.WriteOption("SSH_PUBLIC_KEY", publicKey)
	utils.WriteOption("SSH_KEY_FINGERPRINT", fingerprint)

	result, err := json.Marshal(sshAccessResult{
		Status:      "enabled",
		PublicKey:   publicKey,
		Fingerprint: fingerprint,
	})
	if err != nil {
		return "", err
	}

	return string(result), nil
}

func taskDisableSSHAccess() (string, error) {
	if err := reconcileManagedSSHKey(sshDirectory, ""); err != nil {
		return "", err
	}

	utils.WriteOption("SSH_ACCESS_ENABLED", "false")
	utils.DeleteOption("SSH_PUBLIC_KEY")
	utils.DeleteOption("SSH_KEY_FINGERPRINT")

	result, err := json.Marshal(sshAccessResult{Status: "disabled"})
	if err != nil {
		return "", err
	}

	return string(result), nil
}

func parseSSHEd25519PublicKey(input string) (string, string, error) {
	trimmed := strings.TrimSpace(input)
	if trimmed == "" || strings.ContainsAny(trimmed, "\r\n") {
		return "", "", fmt.Errorf("provide exactly one SSH public key")
	}

	fields := strings.Fields(trimmed)
	if len(fields) < 2 || len(fields) > 3 || fields[0] != "ssh-ed25519" {
		return "", "", fmt.Errorf("only one ssh-ed25519 public key is supported")
	}

	keyBlob, err := base64.StdEncoding.DecodeString(fields[1])
	if err != nil || !validEd25519KeyBlob(keyBlob) {
		return "", "", fmt.Errorf("invalid ssh-ed25519 public key")
	}

	digest := sha256.Sum256(keyBlob)
	publicKey := "ssh-ed25519 " + base64.StdEncoding.EncodeToString(keyBlob) + " " + managedSSHComment
	fingerprint := "SHA256:" + base64.RawStdEncoding.EncodeToString(digest[:])

	return publicKey, fingerprint, nil
}

func validEd25519KeyBlob(blob []byte) bool {
	if len(blob) < 4 {
		return false
	}

	typeLength := int(binary.BigEndian.Uint32(blob[:4]))
	if typeLength != len("ssh-ed25519") || len(blob) < 4+typeLength+4 {
		return false
	}
	if string(blob[4:4+typeLength]) != "ssh-ed25519" {
		return false
	}

	keyLengthOffset := 4 + typeLength
	keyLength := int(binary.BigEndian.Uint32(blob[keyLengthOffset : keyLengthOffset+4]))
	return keyLength == 32 && len(blob) == keyLengthOffset+4+keyLength
}

func reconcileManagedSSHKey(directory string, publicKey string) error {
	if info, err := os.Lstat(directory); err == nil {
		if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
			return fmt.Errorf("SSH directory is not a regular directory")
		}
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("inspect SSH directory: %w", err)
	} else if err := os.MkdirAll(directory, 0700); err != nil {
		return fmt.Errorf("create SSH directory: %w", err)
	}

	if err := os.Chmod(directory, 0700); err != nil {
		return fmt.Errorf("secure SSH directory: %w", err)
	}

	authorizedKeysPath := filepath.Join(directory, "authorized_keys")
	if info, err := os.Lstat(authorizedKeysPath); err == nil && info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("authorized_keys must not be a symbolic link")
	} else if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("inspect authorized_keys: %w", err)
	}

	existing, err := os.ReadFile(authorizedKeysPath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("read authorized_keys: %w", err)
	}

	lines := strings.Split(string(existing), "\n")
	kept := make([]string, 0, len(lines)+1)
	for _, line := range lines {
		if strings.TrimSpace(line) == "" || isManagedSSHKey(line) {
			continue
		}
		kept = append(kept, line)
	}
	if publicKey != "" {
		kept = append(kept, publicKey)
	}

	contents := ""
	if len(kept) > 0 {
		contents = strings.Join(kept, "\n") + "\n"
	}

	temporary, err := os.CreateTemp(directory, ".authorized_keys-*")
	if err != nil {
		return fmt.Errorf("create authorized_keys temporary file: %w", err)
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)

	if err := temporary.Chmod(0600); err != nil {
		temporary.Close()
		return fmt.Errorf("secure authorized_keys temporary file: %w", err)
	}
	if _, err := temporary.WriteString(contents); err != nil {
		temporary.Close()
		return fmt.Errorf("write authorized_keys: %w", err)
	}
	if err := temporary.Sync(); err != nil {
		temporary.Close()
		return fmt.Errorf("sync authorized_keys: %w", err)
	}
	if err := temporary.Close(); err != nil {
		return fmt.Errorf("close authorized_keys: %w", err)
	}
	if err := os.Rename(temporaryPath, authorizedKeysPath); err != nil {
		return fmt.Errorf("replace authorized_keys: %w", err)
	}

	return nil
}

func isManagedSSHKey(line string) bool {
	fields := strings.Fields(line)
	return len(fields) == 3 && fields[2] == managedSSHComment
}

func verifyRootSSHAccess() error {
	sshdPath, err := exec.LookPath("sshd")
	if err != nil {
		sshdPath = "/usr/sbin/sshd"
		if _, statErr := os.Stat(sshdPath); statErr != nil {
			return fmt.Errorf("OpenSSH server is not installed")
		}
	}

	output, err := exec.Command(sshdPath, "-T").Output()
	if err != nil {
		return fmt.Errorf("could not verify the effective SSH server configuration")
	}

	settings := string(output)
	if !strings.Contains(settings, "pubkeyauthentication yes") {
		return fmt.Errorf("SSH public-key authentication is disabled")
	}
	if strings.Contains(settings, "permitrootlogin no") {
		return fmt.Errorf("SSH root login is disabled")
	}

	return nil
}
