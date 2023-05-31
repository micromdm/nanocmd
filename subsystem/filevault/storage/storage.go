// Package storage defines types supporting FileVault FDE commands and responses.
package storage

import (
	"context"
)

// PRKCertRetriever retrieves the raw DER certificate bytes used for encrypting the PRK for an id.
type PRKCertRetriever interface {
	RetrievePRKCertRaw(ctx context.Context, id string) ([]byte, error)
}

// PRKRetriever retrieves the existing (already escrowed and decrypted) PRK for an enrollment.
type PRKRetriever interface {
	RetrievePRK(ctx context.Context, id string) (string, error)
}

// PRKEscrower escrows the encrypted CMS of the PRK.
type PRKEscrower interface {
	EscrowPRK(ctx context.Context, id string, cms []byte) error
}

// FVEnable is intended for enabling FileVault and escrowing PRKs.
type FVEnable interface {
	PRKCertRetriever
	PRKEscrower
}

// FVRotate is intended for rotating encrypted PRKs.
type FVRotate interface {
	FVEnable
	PRKRetriever
}

// PRKStorage retrieves and stores unencrypted PRKs.
type PRKStorage interface {
	PRKRetriever
	StorePRK(ctx context.Context, id, prk string) error
}
