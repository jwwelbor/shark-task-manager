# Prompt: Extract workflow material from a ~/.claude/skills skill into a layered structure

Use this prompt to migrate **a source skill at a time** from `~/.claude/skills` into a layered architecture with clear separation of concerns.

---

You are refactoring a source skill from a monolithic skill architecture into a layered architecture with clear separation between:

1. **workflow controls**
2. **step execution prompts**
3. **domain methodology**
4. **supporting references and reusable content**

## Goal

Extract all workflow-driving material out of the source skill so the result follows this target model:

- **workflow definitions** own the flow
- **prompt files** own step execution adapters
- **skillfiles** own reusable domain knowledge only

Assume the reader has **no prior knowledge of any specific framework, repository architecture, lifecycle model, or orchestration systems**. Do not rely on product-specific vocabulary unless it appears in the source material and is necessary to explain the extraction.

## Inputs

Source skill path:
`<SOURCE_SKILL_DIR>`

Target repo root:
`<TARGET_REPO_ROOT>`

Optional existing counterpart skill:
`<TARGET_SKILL_DIR_OR_NONE>`

## Required separation rules

### 1. Workflow layer owns

Anything that answers **"what happens next?"** belongs in the workflow layer, including:

- ordered steps or phases
- branching logic
- pass/fail routes
- retry / loop-back behavior
- skip conditions
- gating conditions
- terminal states
- paused / blocked / cancelled states
- metadata captured for later routing

### 2. Prompt layer owns

Anything that answers **"how should this step be executed right now?"** belongs in the prompt layer, including:

- mission / objective for the current step
- resume checks
- what to read
- what to produce
- exit criteria
- what result key or outcome to emit
- instructions for interacting with external systems or humans
- composition of reusable methodology via include/augment patterns

### 3. Skill layer owns

Anything that answers **"how is this kind of work done well?"** belongs in the skill layer, including:

- domain procedures
- heuristics
- scoring systems
- frameworks
- lenses
- checklists
- templates
- standards
- worked examples
- reusable evaluation criteria

### 4. Keep these OUT of the skill layer

Do **not** leave these in the refactored skill unless they are purely explanatory:

- direct flow sequencing
- explicit step-to-step progression logic
- route decisions such as "if X then go to Y"
- direct state-machine control
- backend-specific workflow transitions
- orchestration-engine semantics
- status-setting commands used only to drive flow

## Task

Analyze the source skill and design a layered extraction.

### Step 1: Inventory the source skill

Read all relevant files in the source skill and identify:

1. reusable domain methodology
2. embedded workflow sequencing
3. branching or routing logic
4. resume or re-entry logic
5. artifact existence checks
6. output contracts
7. external system interactions
8. human checkpoint / approval points
9. reusable references, templates, and examples

### Step 2: Build a classification matrix

For each major section, classify it as one of:

- **KEEP IN SKILL**
- **MOVE TO PROMPT**
- **MOVE TO WORKFLOW**
- **MOVE TO ORCHESTRATION / INTEGRATION LAYER**
- **SPLIT ACROSS LAYERS**
- **DROP OR MERGE**

Use this table:

| Source File | Section / Lines | Summary | Target Layer | Why |
|---|---|---|---|---|

### Step 3: Design the target structure

Design the extracted structure in four parts:

#### A. Workflow definition

Define:

- ordered steps
- branch points
- skip conditions
- gate conditions
- outputs or records
- retry / remediation loops
- terminal and interrupt states

Use generic names where possible unless the source requires domain-specific terminology.

#### B. Prompt files

For each execution step, define a prompt file with:

- file path
- purpose
- required variables
- reusable includes / augment sources
- expected outputs
- possible outcomes

#### C. Refactored skill

Define what remains in the skill:

- core methodology
- reusable frameworks
- references
- examples
- standards

#### D. Supporting references

Identify what should become:

- `context/`
- `references/`
- `assets/`
- `workflows/` inside the skill, if those files are still domain-only and reusable

### Step 4: Draft the target artifacts

Produce:

#### 1. Workflow definition draft

Write a skeleton workflow file that shows:

- step order
- routes
- conditions
- skip logic
- records
- terminal outcomes

#### 2. Prompt file list

For each prompt file, provide:

- path
- purpose
- variables
- includes / augment dependencies
- outputs produced
- route or outcome keys emitted

#### 3. Refactored skill outline

Write an outline for the new `SKILL.md` that contains only reusable domain methodology and references.

### Step 5: Identify migration hazards

Call out:

- sections tightly coupled to a specific backend or workflow engine
- sections that mix domain guidance and flow control
- duplicated logic that should be centralized
- content that may need a new cross-cutting skill instead of staying local
- unresolved design choices that need human review

## Output format

Return these sections exactly:

1. **Summary**
2. **Source-to-target classification matrix**
3. **Proposed workflow definition**
4. **Proposed prompt files**
5. **Refactored skill outline**
6. **Migration hazards**
7. **Validation checklist**

## Validation checklist

Before finishing, verify:

- the refactored skill no longer owns flow control
- ordered progression lives in the workflow definition
- step-specific execution guidance lives in prompt files
- reusable methodology stays centralized in the skill
- backend- or engine-specific routing logic has been removed or isolated
- prompts are adapters, not giant copies of reusable domain content
- the result would be understandable even to someone with no prior knowledge of the target architecture

## Important constraints

- Do **not** just rename backend commands
- Do **not** preserve state-machine behavior inside the skill
- Do **not** duplicate methodology across multiple prompts if it can remain reusable
- Prefer extraction and centralization over copy/paste
- Preserve the source skill's domain intent while changing ownership boundaries

---

Suggested placeholders:

- `<SOURCE_SKILL_DIR>` = `~/.claude/skills/<skill-name>`
- `<TARGET_REPO_ROOT>` = the root of whatever repository will receive the extracted structure
- `<TARGET_SKILL_DIR_OR_NONE>` = matching target skill path or `none`
