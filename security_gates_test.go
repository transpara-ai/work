package work_test

import (
	"slices"
	"testing"
	"time"

	"github.com/transpara-ai/work"
)

func TestSaaSTemplateV1SecurityScannersCoverD2Gates(t *testing.T) {
	bom := work.SaaSTemplateV1FactoryRuntimeBOM()
	if bom.TemplateID != work.SaaSTemplateV1ID {
		t.Fatalf("TemplateID = %q; want %q", bom.TemplateID, work.SaaSTemplateV1ID)
	}
	if bom.SecurityGateVersion != work.SaaSTemplateV1SecurityGateVersion {
		t.Fatalf("SecurityGateVersion = %q; want %q", bom.SecurityGateVersion, work.SaaSTemplateV1SecurityGateVersion)
	}

	byGate := map[work.SecurityGateID]work.SecurityScanner{}
	for _, scanner := range bom.SecurityScanners {
		if scanner.Tool == "" || scanner.Version == "" {
			t.Fatalf("scanner evidence missing tool/version: %#v", scanner)
		}
		byGate[scanner.Gate] = scanner
	}
	for _, gate := range work.RequiredSaaSTemplateV1SecurityGates() {
		if _, ok := byGate[gate]; !ok {
			t.Fatalf("missing scanner for gate %s", gate)
		}
	}
}

func TestEvaluateSecurityGateCertificationPassesWithCompleteCleanEvidence(t *testing.T) {
	result := work.EvaluateSecurityGateCertification(cleanEvidence(), nil, time.Date(2026, 5, 15, 0, 0, 0, 0, time.UTC))
	if result.Blocked {
		t.Fatalf("clean complete evidence blocked certification: %#v", result)
	}
}

func TestEvaluateSecurityGateCertificationBlocksMissingEvidence(t *testing.T) {
	evidence := cleanEvidence()
	evidence = slices.DeleteFunc(evidence, func(item work.SecurityGateEvidence) bool {
		return item.Gate == work.GateSAST
	})
	result := work.EvaluateSecurityGateCertification(evidence, nil, time.Date(2026, 5, 15, 0, 0, 0, 0, time.UTC))
	if !result.Blocked || !slices.Contains(result.MissingEvidence, work.GateSAST) {
		t.Fatalf("missing SAST evidence did not block certification: %#v", result)
	}
}

func TestEvaluateSecurityGateCertificationBlocksOpenCriticalEvenWithWaiver(t *testing.T) {
	asOf := time.Date(2026, 5, 15, 0, 0, 0, 0, time.UTC)
	evidence := cleanEvidence()
	evidence[0].Findings = []work.SecurityFinding{{
		ID:       "finding-critical",
		Gate:     work.GateDependencyVulnerabilityScan,
		Severity: work.FindingSeverityCritical,
		Status:   work.FindingStatusOpen,
		WaiverID: "waiver-critical",
	}}
	waivers := []work.SecurityWaiver{validWaiver("waiver-critical", "finding-critical", asOf)}

	result := work.EvaluateSecurityGateCertification(evidence, waivers, asOf)
	if !result.Blocked || len(result.BlockingFindings) != 1 {
		t.Fatalf("open critical finding did not block certification: %#v", result)
	}
}

func TestEvaluateSecurityGateCertificationBlocksOpenHighWithoutValidWaiver(t *testing.T) {
	asOf := time.Date(2026, 5, 15, 0, 0, 0, 0, time.UTC)
	evidence := cleanEvidence()
	evidence[0].Findings = []work.SecurityFinding{{
		ID:       "finding-high",
		Gate:     work.GateSAST,
		Severity: work.FindingSeverityHigh,
		Status:   work.FindingStatusOpen,
		WaiverID: "waiver-high",
	}}

	result := work.EvaluateSecurityGateCertification(evidence, nil, asOf)
	if !result.Blocked || len(result.BlockingFindings) != 1 {
		t.Fatalf("open high finding without waiver did not block certification: %#v", result)
	}

	result = work.EvaluateSecurityGateCertification(evidence, []work.SecurityWaiver{validWaiver("waiver-high", "finding-high", asOf)}, asOf)
	if result.Blocked {
		t.Fatalf("valid waiver did not unblock high finding: %#v", result)
	}

	result = work.EvaluateSecurityGateCertification(evidence, []work.SecurityWaiver{expiredWaiver("waiver-high", "finding-high", asOf)}, asOf)
	if !result.Blocked || len(result.BlockingFindings) != 1 {
		t.Fatalf("expired waiver unblocked high finding: %#v", result)
	}

	incomplete := validWaiver("waiver-high", "finding-high", asOf)
	incomplete.CompensatingControls = ""
	result = work.EvaluateSecurityGateCertification(evidence, []work.SecurityWaiver{incomplete}, asOf)
	if !result.Blocked || len(result.BlockingFindings) != 1 {
		t.Fatalf("incomplete waiver unblocked high finding: %#v", result)
	}

	wrongScope := validWaiver("waiver-high", "finding-high", asOf)
	wrongScope.NotValidFor = []string{"certification"}
	result = work.EvaluateSecurityGateCertification(evidence, []work.SecurityWaiver{wrongScope}, asOf)
	if !result.Blocked || len(result.BlockingFindings) != 1 {
		t.Fatalf("certification-scoped-out waiver unblocked high finding: %#v", result)
	}
}

func TestEvaluateSecurityGateCertificationBlocksCommittedSecretEvenWhenFindingStatusWaived(t *testing.T) {
	evidence := cleanEvidence()
	evidence[0].Findings = []work.SecurityFinding{{
		ID:        "finding-secret",
		Gate:      work.GateSecretScan,
		Severity:  work.FindingSeverityHigh,
		Status:    work.FindingStatusWaived,
		SecretHit: true,
	}}

	result := work.EvaluateSecurityGateCertification(evidence, []work.SecurityWaiver{validWaiver("waiver-secret", "finding-secret", time.Now())}, time.Now())
	if !result.Blocked || len(result.BlockingFindings) != 1 {
		t.Fatalf("committed secret finding did not block certification: %#v", result)
	}
}

func TestEvaluateSecurityGateCertificationBlocksMissingScannerVersion(t *testing.T) {
	evidence := cleanEvidence()
	evidence[0].ScannerVersion = ""

	result := work.EvaluateSecurityGateCertification(evidence, nil, time.Date(2026, 5, 15, 0, 0, 0, 0, time.UTC))
	if !result.Blocked || len(result.BlockingReasons) == 0 {
		t.Fatalf("missing scanner version did not block certification: %#v", result)
	}
}

func TestEvaluateSecurityGateCertificationIgnoresResolvedAndWaivedNonSecretFindings(t *testing.T) {
	evidence := cleanEvidence()
	evidence[0].Findings = []work.SecurityFinding{
		{
			ID:       "finding-resolved",
			Gate:     work.GateSAST,
			Severity: work.FindingSeverityHigh,
			Status:   work.FindingStatusResolved,
		},
		{
			ID:       "finding-waived",
			Gate:     work.GateSAST,
			Severity: work.FindingSeverityHigh,
			Status:   work.FindingStatusWaived,
		},
	}

	result := work.EvaluateSecurityGateCertification(evidence, nil, time.Date(2026, 5, 15, 0, 0, 0, 0, time.UTC))
	if result.Blocked {
		t.Fatalf("resolved or waived non-secret finding blocked certification: %#v", result)
	}
}

func cleanEvidence() []work.SecurityGateEvidence {
	scanners := work.SaaSTemplateV1SecurityScanners()
	out := make([]work.SecurityGateEvidence, 0, len(scanners))
	for _, scanner := range scanners {
		status := work.SecurityGateStatusPass
		if scanner.Gate == work.GateContainerOrArtifactScan {
			status = work.SecurityGateStatusNotApplicable
		}
		out = append(out, work.SecurityGateEvidence{
			Gate:           scanner.Gate,
			Status:         status,
			ScannerTool:    scanner.Tool,
			ScannerVersion: scanner.Version,
		})
	}
	return out
}

func validWaiver(id, findingID string, asOf time.Time) work.SecurityWaiver {
	return work.SecurityWaiver{
		ID:                   id,
		FindingID:            findingID,
		ApproverRole:         "security",
		ExpiresAt:            asOf.Add(24 * time.Hour),
		Reason:               "accepted temporarily for local template verification",
		CompensatingControls: "restricted to local generated-template verification",
		NotValidFor:          []string{"production_deploy"},
	}
}

func expiredWaiver(id, findingID string, asOf time.Time) work.SecurityWaiver {
	waiver := validWaiver(id, findingID, asOf)
	waiver.ExpiresAt = asOf.Add(-time.Hour)
	return waiver
}
