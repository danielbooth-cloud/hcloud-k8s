package provisioner

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"path/filepath"
	"net/http"

	"golang.org/x/crypto/ssh"
	"hcloud-k8s/internal/errors"
	"github.com/hetznercloud/hcloud-go/v2/hcloud"
)

func (p *ServerProvisioner) generateSSHKey(ctx context.Context, clusterName string) (*hcloud.SSHKey, error) {
	sshKeyName := fmt.Sprintf("%s-key", clusterName)
	privateKeyPath := fmt.Sprintf("%s_id_rsa", sshKeyName)
	publicKeyPath := fmt.Sprintf("%s_id_rsa.pub", sshKeyName)

	// Check if the key already exists in Hetzner
	keys, err := p.client.SSHKey.All(ctx)
	if err != nil {
		return nil, errors.NewAPIError("SSHKey", http.StatusInternalServerError, "failed to list SSH keys")
	}
	for _, key := range keys {
		if key.Name == sshKeyName {
			return key, nil
		}
	}

	// Check if the key files already exist locally
	if _, err := os.Stat(privateKeyPath); err == nil {
		return nil, fmt.Errorf("private key already exists at %s", privateKeyPath)
	}
	if _, err := os.Stat(publicKeyPath); err == nil {
		return nil, fmt.Errorf("public key already exists at %s", publicKeyPath)
	}

	// Generate RSA key pair
	privateKey, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return nil, fmt.Errorf("failed to generate private key: %v", err)
	}

	// Encode private key to PEM format
	privateKeyFile, err := os.Create(privateKeyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create private key file: %v", err)
	}
	defer privateKeyFile.Close()

	privateKeyPEM := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	}
	if err := pem.Encode(privateKeyFile, privateKeyPEM); err != nil {
		return nil, fmt.Errorf("failed to encode private key: %v", err)
	}

	// Generate and encode public key
	publicKey, err := ssh.NewPublicKey(&privateKey.PublicKey)
	if err != nil {
		return nil, fmt.Errorf("failed to generate public key: %v", err)
	}
	pubKeyBytes := ssh.MarshalAuthorizedKey(publicKey)

	// Write public key to file
	if err := os.WriteFile(publicKeyPath, pubKeyBytes, 0644); err != nil {
		return nil, fmt.Errorf("failed to write public key: %v", err)
	}

	// Create SSH key in Hetzner
	hcloudKey, _, err := p.client.SSHKey.Create(ctx, hcloud.SSHKeyCreateOpts{
		Name:      sshKeyName,
		PublicKey: string(pubKeyBytes),
	})
	if err != nil {
		return nil, errors.NewAPIError("SSHKey", http.StatusInternalServerError, fmt.Sprintf("failed to create SSH key: %v", err))
	}

	// Notify the user about the private key location
	fmt.Printf("Private key stored at: %s\n", filepath.Join(".", privateKeyPath))

	return hcloudKey, nil
} 