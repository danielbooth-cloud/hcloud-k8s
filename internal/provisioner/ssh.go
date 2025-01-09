package provisioner

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"golang.org/x/crypto/ssh"
	"hcloud-k8s/internal/errors"
	"github.com/hetznercloud/hcloud-go/v2/hcloud"
)

func (p *ServerProvisioner) generateSSHKey(ctx context.Context, clusterName string) (*hcloud.SSHKey, error) {
	sshKeyName := fmt.Sprintf("%s-key", clusterName)
	privateKeyPath := fmt.Sprintf("%s_id_rsa", sshKeyName)
	publicKeyPath := fmt.Sprintf("%s_id_rsa.pub", sshKeyName)

	// Check if the key already exists in Hetzner
	if key, err := p.findExistingSSHKey(ctx, sshKeyName); err == nil {
		return key, nil
	}

	// Generate RSA key pair
	privateKey, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return nil, fmt.Errorf("failed to generate private key: %v", err)
	}

	// Save private key
	if err := savePEMKey(privateKeyPath, privateKey); err != nil {
		return nil, err
	}

	// Generate and save public key
	publicKey, err := ssh.NewPublicKey(&privateKey.PublicKey)
	if err != nil {
		return nil, fmt.Errorf("failed to generate public key: %v", err)
	}
	if err := os.WriteFile(publicKeyPath, ssh.MarshalAuthorizedKey(publicKey), 0644); err != nil {
		return nil, fmt.Errorf("failed to write public key: %v", err)
	}

	// Create SSH key in Hetzner
	hcloudKey, _, err := p.client.SSHKey.Create(ctx, hcloud.SSHKeyCreateOpts{
		Name:      sshKeyName,
		PublicKey: string(ssh.MarshalAuthorizedKey(publicKey)),
	})
	if err != nil {
		return nil, errors.NewAPIError("SSHKey", http.StatusInternalServerError, fmt.Sprintf("failed to create SSH key: %v", err))
	}

	fmt.Printf("Private key stored in active directory")
	return hcloudKey, nil
}

func (p *ServerProvisioner) findExistingSSHKey(ctx context.Context, sshKeyName string) (*hcloud.SSHKey, error) {
	keys, err := p.client.SSHKey.All(ctx)
	if err != nil {
		return nil, errors.NewAPIError("SSHKey", http.StatusInternalServerError, "failed to list SSH keys")
	}
	for _, key := range keys {
		if key.Name == sshKeyName {
			return key, nil
		}
	}
	return nil, fmt.Errorf("SSH key not found")
}

func savePEMKey(filePath string, key *rsa.PrivateKey) error {
	// Create the file with restricted permissions (0600)
	privateKeyFile, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("failed to create private key file: %v", err)
	}
	defer privateKeyFile.Close()

	return pem.Encode(privateKeyFile, &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(key),
	})
}
