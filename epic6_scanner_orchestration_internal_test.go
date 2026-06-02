package work

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEpic6ContainerEvidenceBlocksArtifactsWithoutTrivyScan(t *testing.T) {
	targetDir := t.TempDir()
	evidenceDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(targetDir, "dist"), 0o755); err != nil {
		t.Fatalf("create build artifact: %v", err)
	}

	item := epic6ContainerEvidence(targetDir, evidenceDir, "")
	if item.Status != SecurityGateStatusFail {
		t.Fatalf("status = %q; want fail", item.Status)
	}
	if item.EvidenceMode == "not_applicable_with_proof" || item.NotApplicableReason != "" {
		t.Fatalf("container evidence = %#v; want failing scanner-required evidence", item)
	}
	if len(item.Findings) == 0 || item.Findings[0].ID != "finding_epic6_container_artifact_trivy_missing" {
		t.Fatalf("findings = %#v; want missing trivy finding", item.Findings)
	}
}
