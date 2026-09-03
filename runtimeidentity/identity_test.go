package runtimeidentity

import (
	"crypto/ed25519"
	"encoding/base64"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/transpara-ai/eventgraph/go/pkg/actor"
)

func TestPersistentIdentityRequiresProtectedKeyFile(t *testing.T) {
	actors := actor.NewInMemoryActorStore()
	if _, err := Resolve(actors, "Operator", "", true); err == nil || !strings.Contains(err.Error(), "WORK_SIGNING_KEY_FILE") {
		t.Fatalf("missing key error = %v", err)
	}
	path := filepath.Join(t.TempDir(), "signing-key")
	seed := make([]byte, ed25519.SeedSize)
	for index := range seed {
		seed[index] = byte(index + 1)
	}
	if err := os.WriteFile(path, []byte(base64.StdEncoding.EncodeToString(seed)), 0o600); err != nil {
		t.Fatal(err)
	}
	identity, err := Resolve(actors, "Operator", path, true)
	if err != nil {
		t.Fatal(err)
	}
	if identity.ActorID.Value() == "" || identity.Signer == nil {
		t.Fatalf("identity = %+v", identity)
	}
}

func TestPersistentIdentityRejectsBroadFilePermissions(t *testing.T) {
	path := filepath.Join(t.TempDir(), "signing-key")
	if err := os.WriteFile(path, []byte(base64.StdEncoding.EncodeToString(make([]byte, ed25519.SeedSize))), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Resolve(actor.NewInMemoryActorStore(), "Operator", path, true); err == nil || !strings.Contains(err.Error(), "only by its owner") {
		t.Fatalf("permission error = %v", err)
	}
}

func TestDevelopmentIdentityRemainsAvailableInMemory(t *testing.T) {
	first, err := Resolve(actor.NewInMemoryActorStore(), "Developer", "", false)
	if err != nil {
		t.Fatal(err)
	}
	second, err := Resolve(actor.NewInMemoryActorStore(), "Developer", "", false)
	if err != nil {
		t.Fatal(err)
	}
	if first.ActorID != second.ActorID {
		t.Fatalf("development actor ids differ: %s %s", first.ActorID, second.ActorID)
	}
}
