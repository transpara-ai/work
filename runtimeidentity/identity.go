// Package runtimeidentity loads a Work event-signing identity without deriving
// production private keys from public display names.
package runtimeidentity

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/transpara-ai/eventgraph/go/pkg/actor"
	"github.com/transpara-ai/eventgraph/go/pkg/event"
	"github.com/transpara-ai/eventgraph/go/pkg/types"
)

type Identity struct {
	ActorID types.ActorID
	Signer  event.Signer
}

type signer struct{ key ed25519.PrivateKey }

func (s *signer) Sign(data []byte) (types.Signature, error) {
	return types.NewSignature(ed25519.Sign(s.key, data))
}

// Resolve requires a protected key file for persistent stores. When no
// persistent store is in use, omitting the file retains the historical
// deterministic development identity so existing local workflows still run.
func Resolve(actors actor.IActorStore, displayName, keyFile string, persistent bool) (Identity, error) {
	displayName = strings.TrimSpace(displayName)
	keyFile = strings.TrimSpace(keyFile)
	if actors == nil || displayName == "" {
		return Identity{}, errors.New("actor store and display name are required")
	}
	if keyFile == "" {
		if persistent {
			return Identity{}, errors.New("WORK_SIGNING_KEY_FILE is required for a persistent store")
		}
		return developmentIdentity(actors, displayName)
	}
	privateKey, err := loadKey(keyFile)
	if err != nil {
		return Identity{}, err
	}
	publicKey, err := types.NewPublicKey(privateKey.Public().(ed25519.PublicKey))
	if err != nil {
		return Identity{}, fmt.Errorf("signing public key: %w", err)
	}
	registered, err := actors.Register(publicKey, displayName, event.ActorTypeHuman)
	if err != nil {
		return Identity{}, fmt.Errorf("register signing identity: %w", err)
	}
	return Identity{ActorID: registered.ID(), Signer: &signer{key: privateKey}}, nil
}

func loadKey(path string) (ed25519.PrivateKey, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("stat WORK_SIGNING_KEY_FILE: %w", err)
	}
	if !info.Mode().IsRegular() || info.Mode().Perm()&0o077 != 0 {
		return nil, errors.New("WORK_SIGNING_KEY_FILE must be a regular file readable only by its owner")
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read WORK_SIGNING_KEY_FILE: %w", err)
	}
	decoded, err := base64.StdEncoding.DecodeString(strings.TrimSpace(string(raw)))
	if err != nil {
		return nil, errors.New("WORK_SIGNING_KEY_FILE must contain base64")
	}
	switch len(decoded) {
	case ed25519.SeedSize:
		return ed25519.NewKeyFromSeed(decoded), nil
	case ed25519.PrivateKeySize:
		return ed25519.PrivateKey(append([]byte(nil), decoded...)), nil
	default:
		return nil, errors.New("WORK_SIGNING_KEY_FILE must decode to an Ed25519 seed or private key")
	}
}

func developmentIdentity(actors actor.IActorStore, displayName string) (Identity, error) {
	humanSeed := sha256.Sum256([]byte("human:" + displayName))
	humanKey := ed25519.NewKeyFromSeed(humanSeed[:])
	publicKey, err := types.NewPublicKey(humanKey.Public().(ed25519.PublicKey))
	if err != nil {
		return Identity{}, err
	}
	registered, err := actors.Register(publicKey, displayName, event.ActorTypeHuman)
	if err != nil {
		return Identity{}, err
	}
	signerSeed := sha256.Sum256([]byte("signer:" + registered.ID().Value()))
	return Identity{ActorID: registered.ID(), Signer: &signer{key: ed25519.NewKeyFromSeed(signerSeed[:])}}, nil
}
