// Package contracts exercises the public seams shared by E38-F04 consumers.
package contracts

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/sharkdata"
	"github.com/jwwelbor/shark-task-manager/internal/templates"
)

// TestTC001_I04InboxMessageProtocol preserves the canonical I-04 message
// shape and acknowledgement lifecycle as a distributable file protocol.
func TestTC001_I04InboxMessageProtocol(t *testing.T) {
	content, err := sharkdata.ReadEmbedded("skills/shark-attack/context/message-schema.md")
	if err != nil {
		t.Fatalf("ReadEmbedded(message-schema.md) error = %v", err)
	}
	for _, want := range []string{
		"message_id", "sender_role", "recipient_role", "root_key", "child_key",
		"subject", "requested_action", "question", "urgency", "evidence", "created_at",
		"After acting, write the resulting durable artifact", "acknowledge or remove",
		"Do not remove the only durable copy",
	} {
		if !strings.Contains(string(content), want) {
			t.Errorf("message protocol omits I-04 field or lifecycle rule %q", want)
		}
	}
}

// TestTC002_I04ArtifactProtocol ensures F04 publishes a bounded artifact
// schema rather than a runtime persistence API.
func TestTC002_I04ArtifactProtocol(t *testing.T) {
	content, err := sharkdata.ReadEmbedded("skills/shark-attack/context/message-schema.md")
	if err != nil {
		t.Fatalf("ReadEmbedded(message-schema.md) error = %v", err)
	}
	for _, want := range []string{
		"decision", "handoff", "escalation", "resolution", "artifact_id", "type", "status",
		"roles", "root_key", "evidence", "updated_at", "next_action", "trigger", "route",
		"byte-equivalent content", "conflict, not an update", "credentials", "rendered prompts",
		"unrestricted worker output", "relative paths", "symlink escapes",
	} {
		if !strings.Contains(string(content), want) {
			t.Errorf("artifact protocol omits bounded-contract rule %q", want)
		}
	}
}

// TestTC003_X03RolePullContractUsesWorkflowAndClaimAuthorities verifies the
// distributed procedure consumes, rather than reimplements, E19's sprint and
// claim authorities. Service-level ordering and lease behavior remain covered
// at their owning production seams.
func TestTC003_X03RolePullContractUsesWorkflowAndClaimAuthorities(t *testing.T) {
	pull, err := sharkdata.ReadEmbedded("skills/shark-attack/workflows/pull-by-role.md")
	if err != nil {
		t.Fatalf("ReadEmbedded(pull-by-role.md) error = %v", err)
	}
	content := string(pull)
	normalizedContent := strings.Join(strings.Fields(content), " ")
	requiredClauses := []struct {
		name string
		text string
	}{
		// TC-F06-006: selection is read-only and bounded before a separate claim.
		{"workflow role is the sole pull input", "The workflow-resolved `agent_type` is the only role input to a pull."},
		{"selection service", "`SprintService.GetNextTask(ctx, agentType)`"},
		{"CLI selection adapter", "`shark sprint next --agent=<type>`"},
		{"selection preserves workflow ordering", "priority/dependency order"},
		{"selection is read-only", "This is read-only selection, not a claim."},
		{"selection retains live claims", "does not exclude live claims"},
		{"selected child includes canonical prompt metadata", "canonical prompt metadata"},
		{"selection cannot authorize a role", "it is not a claim, a role authorization, a free-form role assignment, or a second workflow engine."},
		{"claim authority", "`ClaimService.Claim`"},
		{"claim service owns lease lifecycle", "ClaimService owns session generation, expiry reclamation, claim conflict reporting, heartbeat, and session-scoped release."},
		{"worker returns bounded evidence", "context/worker-ownership.md"},

		// TC-F06-007: workflow owns authorization; roster and model metadata do not.
		{"legacy assignment is non-authoritative", "legacy `agent` assignment"},
		{"model tier is non-authoritative", "`model_tier`"},
		{"roster cannot grant authority", "it does not grant claim or status authority"},
		{"roster model preference cannot select", "A roster's model preference cannot select work or override workflow metadata."},
		{"direct local claim is only a lease", "A direct local `shark claim` is a lease operation, not role authorization."},

		// TC-F06-006: every bounded parent-owned procedure outcome.
		{"no-role outcome", "**No role:** Return the missing workflow-role outcome."},
		{"no-role does not infer authority", "Do not infer a role from the roster, legacy `agent` assignment, actor identity, or `model_tier`."},
		{"no-item outcome", "**No item:** Return the no-item outcome. Do not claim another item."},
		{"claim-conflict outcome", "**Claim conflict:** Return the conflict. Do not force-claim, retry with another role, or steal the live lease."},
		{"never force-claim rule", "Never force-claim"},
		{"claim cannot derive from roster metadata", "or construct a detached claim from roster data."},
		{"workflow pause or gate outcome", "**Workflow pause/gate:** For a workflow pause/gate, return the pause or gate outcome. Do not transition workflow state or release the dispatched parent lease."},
		{"parent owns outcomes and state", "The parent Rider loop owns these outcomes, the dispatched parent lease, and workflow transitions."},

		// The procedure also bounds the remaining unsupported-capability exit.
		{"missing product gate outcome", "For missing product gates, recommend bootstrap or escalation; do not guess product decisions."},
		{"unavailable capability outcome", "Otherwise, stop with an actionable capability gap."},
		{"only routing and lease authorities", "The workflow engine, sprint service, and claim service remain the only routing and lease authorities."},
	}
	for _, clause := range requiredClauses {
		if !strings.Contains(normalizedContent, clause.text) {
			t.Errorf("pull-by-role procedure omits required %s clause %q", clause.name, clause.text)
		}
	}
	prohibitedClauses := []struct {
		name string
		text string
	}{
		{"selection excludes live claims", "excludes ineligible, blocked, or already-claimed work"},
		{"roster role selects work", "roster role can select work"},
		{"legacy assignment selects work", "legacy `agent` assignment can select work"},
		{"model tier selects work", "`model_tier` can select work"},
		{"direct local lease authorizes a role", "A direct local `shark claim` grants role authorization."},
		{"worker owns workflow transition", "worker may transition workflow state"},
	}
	for _, prohibited := range prohibitedClauses {
		if strings.Contains(normalizedContent, prohibited.text) {
			t.Errorf("pull-by-role procedure grants prohibited %s clause %q", prohibited.name, prohibited.text)
		}
	}
}

// TestTC004_X05EmbeddedSkillOverrideIsReplaceOnly proves the E32 resolution
// path installs shark-attack, replaces only its selected entrypoint, and keeps
// an unrelated embedded skill resolvable.
func TestTC004_X05EmbeddedSkillOverrideIsReplaceOnly(t *testing.T) {
	projectRoot := t.TempDir()
	if _, err := sharkdata.Init(projectRoot); err != nil {
		t.Fatalf("Init() error = %v", err)
	}
	dataRoot := filepath.Join(projectRoot, sharkdata.SharkDataDirName)
	resolver := templates.NewIncludeResolver(dataRoot)

	defaultSkill, err := resolver.Resolve("{{include: skills/shark-attack/SKILL.md}}")
	if err != nil {
		t.Fatalf("resolve embedded shark-attack skill error = %v", err)
	}
	if !strings.Contains(defaultSkill, "chair-led council") {
		t.Fatalf("embedded shark-attack skill missing protocol content: %q", defaultSkill)
	}

	overridePath := filepath.Join(dataRoot, "overrides", "skills", "shark-attack", "SKILL.md")
	if err := os.MkdirAll(filepath.Dir(overridePath), 0o755); err != nil {
		t.Fatalf("create shark-attack override directory: %v", err)
	}
	const override = "# Private shark-attack override\n"
	if err := os.WriteFile(overridePath, []byte(override), 0o644); err != nil {
		t.Fatalf("write shark-attack override: %v", err)
	}

	overriddenSkill, err := resolver.Resolve("{{include: skills/shark-attack/SKILL.md}}")
	if err != nil {
		t.Fatalf("resolve shark-attack override error = %v", err)
	}
	if overriddenSkill != override {
		t.Fatalf("resolved shark-attack skill = %q, want only its override %q", overriddenSkill, override)
	}
	unrelatedSkill, err := resolver.Resolve("{{include: skills/quality/SKILL.md}}")
	if err != nil {
		t.Fatalf("resolve unrelated embedded skill error = %v", err)
	}
	if unrelatedSkill == override || !strings.Contains(unrelatedSkill, "Quality") {
		t.Fatalf("shark-attack override shadowed unrelated embedded skill: %q", unrelatedSkill)
	}
}
