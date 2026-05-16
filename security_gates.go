package work

import (
	"fmt"
	"slices"
	"sort"
	"time"
)

const SaaSTemplateV1SecurityGateVersion = "dark-factory-v3.9-d2-security-gates"

type SecurityGateID string

const (
	GateSecretScan                  SecurityGateID = "secret_scan"
	GateDependencyVulnerabilityScan SecurityGateID = "dependency_vulnerability_scan"
	GateDependencyLicenseScan       SecurityGateID = "dependency_license_scan"
	GateSAST                        SecurityGateID = "sast"
	GateAuthFlowSecurityCheck       SecurityGateID = "auth_flow_security_check"
	GateConfigurationSecurityCheck  SecurityGateID = "configuration_security_check"
	GateContainerOrArtifactScan     SecurityGateID = "container_or_build_artifact_scan"
)

type SecurityFindingSeverity string

const (
	FindingSeverityLow      SecurityFindingSeverity = "low"
	FindingSeverityMedium   SecurityFindingSeverity = "medium"
	FindingSeverityHigh     SecurityFindingSeverity = "high"
	FindingSeverityCritical SecurityFindingSeverity = "critical"
)

type SecurityFindingStatus string

const (
	FindingStatusOpen     SecurityFindingStatus = "open"
	FindingStatusResolved SecurityFindingStatus = "resolved"
	FindingStatusWaived   SecurityFindingStatus = "waived"
)

type SecurityGateStatus string

const (
	SecurityGateStatusPass          SecurityGateStatus = "pass"
	SecurityGateStatusFail          SecurityGateStatus = "fail"
	SecurityGateStatusNotApplicable SecurityGateStatus = "not_applicable"
)

type SecurityScanner struct {
	Gate    SecurityGateID `json:"gate"`
	Tool    string         `json:"tool"`
	Version string         `json:"version"`
}

type FactoryRuntimeBOM struct {
	TemplateID          string            `json:"template_id"`
	SecurityGateVersion string            `json:"security_gate_version"`
	SecurityScanners    []SecurityScanner `json:"security_scanners"`
}

type SecurityFinding struct {
	ID        string
	Gate      SecurityGateID
	Severity  SecurityFindingSeverity
	Status    SecurityFindingStatus
	WaiverID  string
	Summary   string
	SecretHit bool
}

type SecurityWaiver struct {
	ID                   string
	FindingID            string
	ApproverRole         string
	ExpiresAt            time.Time
	Reason               string
	CompensatingControls string
	NotValidFor          []string
}

type SecurityGateEvidence struct {
	Gate           SecurityGateID
	Status         SecurityGateStatus
	ScannerTool    string
	ScannerVersion string
	Findings       []SecurityFinding
}

type SecurityGateCertificationResult struct {
	Blocked          bool
	MissingEvidence  []SecurityGateID
	BlockingFindings []SecurityFinding
	BlockingReasons  []string
}

func SaaSTemplateV1SecurityScanners() []SecurityScanner {
	return []SecurityScanner{
		{Gate: GateSecretScan, Tool: "gitleaks", Version: "8.18.4"},
		{Gate: GateDependencyVulnerabilityScan, Tool: "osv-scanner", Version: "1.9.1"},
		{Gate: GateDependencyLicenseScan, Tool: "license-policy", Version: "dark-factory-local-1"},
		{Gate: GateSAST, Tool: "semgrep", Version: "1.96.0"},
		{Gate: GateAuthFlowSecurityCheck, Tool: "auth-flow-check", Version: "dark-factory-local-1"},
		{Gate: GateConfigurationSecurityCheck, Tool: "config-security-check", Version: "dark-factory-local-1"},
		{Gate: GateContainerOrArtifactScan, Tool: "trivy", Version: "0.57.1"},
	}
}

func RequiredSaaSTemplateV1SecurityGates() []SecurityGateID {
	return []SecurityGateID{
		GateSecretScan,
		GateDependencyVulnerabilityScan,
		GateDependencyLicenseScan,
		GateSAST,
		GateAuthFlowSecurityCheck,
		GateConfigurationSecurityCheck,
		GateContainerOrArtifactScan,
	}
}

func SaaSTemplateV1FactoryRuntimeBOM() FactoryRuntimeBOM {
	return FactoryRuntimeBOM{
		TemplateID:          SaaSTemplateV1ID,
		SecurityGateVersion: SaaSTemplateV1SecurityGateVersion,
		SecurityScanners:    SaaSTemplateV1SecurityScanners(),
	}
}

func EvaluateSecurityGateCertification(evidence []SecurityGateEvidence, waivers []SecurityWaiver, asOf time.Time) SecurityGateCertificationResult {
	result := SecurityGateCertificationResult{}
	byGate := map[SecurityGateID]SecurityGateEvidence{}
	for _, gateEvidence := range evidence {
		byGate[gateEvidence.Gate] = gateEvidence
		if gateEvidence.ScannerTool == "" || gateEvidence.ScannerVersion == "" {
			result.BlockingReasons = append(result.BlockingReasons, fmt.Sprintf("%s scanner evidence is missing", gateEvidence.Gate))
		}
		if gateEvidence.Status == SecurityGateStatusFail {
			result.BlockingReasons = append(result.BlockingReasons, fmt.Sprintf("%s gate status is fail", gateEvidence.Gate))
		}
		for _, finding := range gateEvidence.Findings {
			if finding.SecretHit {
				result.BlockingFindings = append(result.BlockingFindings, finding)
				result.BlockingReasons = append(result.BlockingReasons, fmt.Sprintf("%s exposes a committed secret", finding.ID))
				continue
			}
			if !findingOpen(finding) {
				continue
			}
			if finding.Severity == FindingSeverityCritical {
				result.BlockingFindings = append(result.BlockingFindings, finding)
				result.BlockingReasons = append(result.BlockingReasons, fmt.Sprintf("%s is an open critical finding", finding.ID))
				continue
			}
			if finding.Severity == FindingSeverityHigh && !hasValidWaiver(finding, waivers, asOf) {
				result.BlockingFindings = append(result.BlockingFindings, finding)
				result.BlockingReasons = append(result.BlockingReasons, fmt.Sprintf("%s is an open high finding without valid waiver", finding.ID))
			}
		}
	}

	for _, required := range RequiredSaaSTemplateV1SecurityGates() {
		if _, ok := byGate[required]; !ok {
			result.MissingEvidence = append(result.MissingEvidence, required)
			result.BlockingReasons = append(result.BlockingReasons, fmt.Sprintf("%s evidence is missing", required))
		}
	}
	sort.Slice(result.MissingEvidence, func(i, j int) bool { return result.MissingEvidence[i] < result.MissingEvidence[j] })
	result.Blocked = len(result.MissingEvidence) > 0 || len(result.BlockingFindings) > 0 || len(result.BlockingReasons) > 0
	return result
}

func findingOpen(finding SecurityFinding) bool {
	return finding.Status == "" || finding.Status == FindingStatusOpen
}

func hasValidWaiver(finding SecurityFinding, waivers []SecurityWaiver, asOf time.Time) bool {
	for _, waiver := range waivers {
		if waiver.ID == "" || waiver.ID != finding.WaiverID || waiver.FindingID != finding.ID {
			continue
		}
		if waiver.ApproverRole == "" || waiver.Reason == "" || waiver.CompensatingControls == "" {
			continue
		}
		if !waiver.ExpiresAt.After(asOf) {
			continue
		}
		if slices.Contains(waiver.NotValidFor, "certification") || slices.Contains(waiver.NotValidFor, "production_release") {
			continue
		}
		return true
	}
	return false
}
