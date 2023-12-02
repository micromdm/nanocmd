// Package kv implements a key-value FileVault storage.
package kv

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"fmt"

	"github.com/micromdm/nanocmd/subsystem/filevault/storage"
	"github.com/micromdm/nanocmd/utils/cryptoutil"
	"github.com/micromdm/nanocmd/utils/kv"
	"github.com/smallstep/pkcs7"
)

// KV is a FileVault storage backend based on a key-value store.
// Its primary storage duties are initially generating and then
// subsequently loading the keypair from storage.
// It uses a single keypair for all FileVault PSK encryption/decryption.
// The actual PSK storage and retrieval (i.e. once decrypted) is
// abstracted to another storage interface.
type KV struct {
	b kv.Bucket
	p storage.PRKStorage
}

const (
	certCN           = "filevault"
	certValidityDays = 10 * 365

	kvKeyKey  = "key"
	kvKeyCert = "cert"
)

func New(ctx context.Context, b kv.Bucket, p storage.PRKStorage) (*KV, error) {
	kv := &KV{b: b, p: p}
	if err := kv.assureKeypairExists(ctx); err != nil {
		return kv, err
	}
	return kv, nil
}

// assureKeypairExists checks that a keypair exists or generates a new keypair
func (s *KV) assureKeypairExists(ctx context.Context) error {
	// check for cert and key
	certOK, err := s.b.Has(ctx, kvKeyCert)
	if err != nil {
		return fmt.Errorf("checking cert exists: %w", err)
	}
	keyOK, err := s.b.Has(ctx, kvKeyKey)
	if err != nil {
		return fmt.Errorf("checking key exists: %w", err)
	}
	if certOK && keyOK {
		return nil
	}
	// generate new
	key, cert, err := cryptoutil.SelfSignedRSAKeypair(certCN, certValidityDays)
	if err != nil {
		return fmt.Errorf("generating self-signed keypair: %w", err)
	}
	if err = s.b.Set(ctx, kvKeyKey, x509.MarshalPKCS1PrivateKey(key)); err != nil {
		return fmt.Errorf("setting key: %w", err)
	}
	if err = s.b.Set(ctx, kvKeyCert, cert.Raw); err != nil {
		return fmt.Errorf("setting cert: %w", err)
	}
	return nil
}

func (s *KV) RetrievePRKCertRaw(ctx context.Context, _ string) ([]byte, error) {
	return s.b.Get(ctx, kvKeyCert)
}

func (s *KV) RetrievePRK(ctx context.Context, id string) (string, error) {
	return s.p.RetrievePRK(ctx, id)
}

// getKeypair retrieves the certificate and private key from the bucket.
func (s *KV) getKeypair(ctx context.Context) (*rsa.PrivateKey, *x509.Certificate, error) {
	certBytes, err := s.b.Get(ctx, kvKeyCert)
	if err != nil {
		return nil, nil, fmt.Errorf("getting cert: %w", err)
	}
	cert, err := x509.ParseCertificate(certBytes)
	if err != nil {
		return nil, cert, fmt.Errorf("parsing cert: %w", err)
	}
	keyBytes, err := s.b.Get(ctx, kvKeyKey)
	if err != nil {
		return nil, cert, fmt.Errorf("getting key: %w", err)
	}
	key, err := x509.ParsePKCS1PrivateKey(keyBytes)
	if err != nil {
		return key, cert, fmt.Errorf("parsing key: %w", err)
	}
	return key, cert, nil
}

// EscrowPRK decrypts the CMS PRK and stores it.
func (s *KV) EscrowPRK(ctx context.Context, id string, cms []byte) error {
	p7, err := pkcs7.Parse(cms)
	if err != nil {
		return fmt.Errorf("parse PRK CMS: %w", err)
	}
	key, cert, err := s.getKeypair(ctx)
	if err != nil {
		return fmt.Errorf("getting keypair: %w", err)
	}
	prkBytes, err := p7.Decrypt(cert, key)
	if err != nil {
		return fmt.Errorf("decrypting PRK CMS: %w", err)
	}
	if err = s.p.StorePRK(ctx, id, string(prkBytes)); err != nil {
		return fmt.Errorf("store PRK: %w", err)
	}
	return nil
}
