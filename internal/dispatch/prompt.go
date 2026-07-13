package dispatch

import (
	"fmt"
	"os"

	"github.com/jwwelbor/shark-task-manager/internal/templates"
)

const workerOwnershipPreamble = `PARENT LOOP OWNERSHIP CONTRACT:
- You are a spawned worker inside a Shark parent-run loop.
- Do NOT run Shark workflow-state commands against the entity this prompt dispatched you for.
- Never run against the dispatched entity: shark claim, shark heartbeat, shark release, shark status advance, shark status set, shark task next-status, shark task set-status, shark feature next-status, or shark epic next-status.
- Exception: if the workflow prompt below explicitly makes you an orchestration loop over OTHER entities (e.g. a sprint step iterating "shark sprint next" and dispatching each child), driving those child entities is the requested work — the prohibition above still applies to the dispatched entity itself.
- Operate in single-worker mode by default. Do NOT spawn or delegate to additional host-native subagents, agent teams, or external AI CLIs unless the workflow prompt explicitly tells you to run a multi-agent skill or recipe.
- If the bundled agent persona describes broader coordination behavior, treat that as background context only. This contract and the concrete workflow prompt override it for the current dispatched step.
- Complete the requested work, write the requested artifacts, then stop and clearly report the recommended outcome and any follow-up guidance for the parent loop.
- The parent loop owns the dispatched entity's lease and workflow transitions.`

// AssembleDispatchPrompt creates the canonical self-contained worker prompt
// shared by next, run, and team execution. It inlines the selected persona,
// renders its placeholders, and adds the parent-loop ownership contract.
func AssembleDispatchPrompt(prompt, agentType string, vars map[string]string) (string, error) {
	if vars == nil {
		vars = map[string]string{}
	}
	templates.AugmentPlaceholderAliases(vars)
	attached, err := attachAgentBody(prompt, agentType, vars)
	if err != nil {
		return "", err
	}
	return workerOwnershipPreamble + "\n\n---\n\n" + attached, nil
}

func attachAgentBody(prompt, agentType string, vars map[string]string) (string, error) {
	if agentType == "" {
		return prompt, nil
	}
	root := templates.GetOrchestratorEngine().IncludeRoot()
	body, ok := LoadAgentBodyForInline(root, agentType)
	if !ok {
		return prompt, nil
	}
	rendered, err := templates.RenderAndLintAgentBody(body, agentType, vars)
	if err != nil {
		return "", err
	}
	return rendered + "\n\n---\n\n" + prompt, nil
}

// LoadAgentBodyForInline resolves and strips the frontmatter from a bundled
// agent persona. Missing personas deliberately degrade to the workflow prompt.
func LoadAgentBodyForInline(root, agentType string) (string, bool) {
	if agentType == "" {
		return "", false
	}
	resolver := templates.NewIncludeResolverWithEmbed(root)
	resolved, err := resolver.Resolve(fmt.Sprintf("{{include: agents/%s.md}}", agentType))
	if err != nil {
		fmt.Fprintf(os.Stderr, "[shark dispatch] agent body inline skipped for %q: %v\n", agentType, err)
		return "", false
	}
	if resolved == "" {
		return "", false
	}
	return resolved, true
}
