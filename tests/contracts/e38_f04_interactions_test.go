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
	normalized := strings.Join(strings.Fields(string(content)), " ")
	for _, want := range []string{
		"message_id", "sender_role", "recipient_role", "root_key", "child_key",
		"subject", "requested_action", "question", "urgency", "evidence", "created_at",
		"After acting, write the resulting durable artifact", "acknowledge or remove",
		"Do not remove the only durable copy",
	} {
		if !strings.Contains(normalized, want) {
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
	normalized := strings.Join(strings.Fields(string(content)), " ")
	for _, want := range []string{
		"decision", "handoff", "escalation", "resolution", "artifact_id", "type", "status",
		"roles", "root_key", "evidence", "updated_at", "next_action", "trigger", "route",
		"byte-equivalent content", "conflict, not an update", "credentials", "rendered prompts",
		"unrestricted worker output", "relative paths", "absolute paths", "`..` traversal", "symlink escapes",
	} {
		if !strings.Contains(normalized, want) {
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
	normalized := strings.Join(strings.Fields(content), " ")
	for _, want := range []string{
		"workflow-resolved `agent_type`",
		"SprintService.GetNextTask(ctx, agentType)",
		"shark sprint next --agent=<type>",
		"priority/dependency order",
		"ClaimService.Claim",
		"`/shark-rider run <selected-key>`",
		"`response.entity_key`",
		"claims or executes the returned `BacklogItemView` directly",
		"legacy `agent` assignment",
		"`model_tier`",
		"does not grant claim or status authority",
	} {
		if !strings.Contains(normalized, want) {
			t.Errorf("pull-by-role procedure omits required X-03 contract %q", want)
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
