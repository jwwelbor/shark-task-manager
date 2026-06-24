# How to Measure: Frameworks, Schemas, and Traps

Reference reading for `workflows/d02-success-criteria.md`. The workflow is the procedure; this file explains *what makes a measurement system trustworthy*, *which framework to choose when*, and *what fails repeatedly in the wild*.

A great measurement system has three properties:
1. **Falsifiable** — every claim could, in principle, be proven wrong with observable evidence.
2. **Connected** — every metric traces back to a vision element, and the metrics together cover the vision (no orphan metrics, no orphan vision claims).
3. **Actionable** — when a number moves, a human knows what to do about it. A metric no one acts on is decoration.

Most teams fail not at picking metrics but at one of three sins below: vanity metrics, output-as-outcome, or unbalanced horizons. The frameworks here exist mainly to prevent those sins.

---

## The Per-Metric Schema (the unit of work)

Every metric in D02 — primary or guardrail — gets the same eight-field treatment. This is the contract. Skipping fields is the most common quality failure.

| Field | Purpose | Failure if omitted |
|---|---|---|
| **Name** | Short label for dashboards and conversation. | Long name → no one says it → no one tracks it. |
| **Definition** | One-sentence operational definition someone could compute identically. | Different people compute different numbers; arguments instead of decisions. |
| **Why this metric (traceability)** | Which line of D01 this metric evaluates. | Orphan metric → optimization with no link to vision. |
| **Type (leading vs lagging)** | Tells the team whether the metric is an early signal or a final outcome. | All-leading: we declare victory while losing. All-lagging: we find out at the horizon, too late. |
| **Measurement method** | Where the number comes from — query, instrumentation, survey. | If it can't be measured today, the team is silently committing to instrumentation work. |
| **Data source** | Authoritative system or dataset. | "The dashboard says 42, the BI report says 47" — until you name the source, this never resolves. |
| **Baseline** | Current value (or `TBD — owner — by date`). | A target with no baseline is meaningless. *"Increase X"* from what? |
| **Target (with thresholds)** | Success / warning / failure values, dated. | Single point targets miss gradient. Three thresholds describe shape. |
| **Owner** | Named human accountable for this number. | Team-owned metric is no-one-owned. |

A metric that has all eight fields is a metric a team can actually use. A metric missing any of them will degrade into a debate at review time.

---

## Leading vs Lagging — and why mix matters

**Lagging indicators** measure outcomes — what we *actually wanted*. They're slow (months/quarters) and definitive. Examples: revenue, retention, NPS, support ticket volume, churn, gross margin.

**Leading indicators** measure precursors — things that move first and predict the lagging metrics. They're fast (days/weeks) and noisy. Examples: activation rate, time-to-first-value, weekly active users, search-to-result ratio.

**Why mix:**
- **Leading-only** = false confidence. Sign-ups are up but no one comes back. The team celebrates and ships more sign-up funnels.
- **Lagging-only** = no steering. By the time the quarterly retention number drops, the team has spent months in the wrong direction.
- **Mixed** = course correction is possible. The leading metric tells you whether you're trending toward the lagging target with time to react.

**Rule of thumb:** at least one of each. For a 12-month vision, pick a 1–2-week leading metric and a 3–6-month lagging metric. The leading one is what the team looks at on Monday; the lagging one is what the executive cares about at the QBR.

---

## Outcome vs Output (the most common sin)

| Output | Outcome |
|---|---|
| Feature shipped | Users do <thing> users couldn't do before |
| Page deployed | Bounce rate from <ref page> drops X% |
| API live with 99.9% uptime | <Consumer team> processes Y events/day they couldn't before |
| Documentation written | Support tickets in category Z drop W% |

**Output metrics measure the team. Outcome metrics measure the product.**

A measurement system dominated by outputs gives the team a way to "succeed" without the user or business getting better. Reframe ruthlessly: for every output candidate, ask *"What changes for the user or business because we delivered this?"* That's the outcome.

The only legitimate output metrics in D02 are **delivery guardrails** — "ship by date X" — and even these should be paired with an outcome ("…and Y becomes true").

---

## Vanity vs Actionable

A **vanity metric** is one that goes up reliably with effort but doesn't connect to value. The diagnostic: *"If this number went up but the business didn't change, would I be happy?"* If yes, it's vanity.

| Common vanity | The actionable replacement |
|---|---|
| Page views | Page-to-action conversion |
| Sign-ups | Activated users (completed core action within N days) |
| Downloads | Weekly retained users at week 4 |
| GitHub stars | Weekly active developers building on the project |
| "Engagement score" composites | One specific behavior tied to value |
| Total users | Weekly/monthly active users with a defined "active" |

Vanity metrics are seductive because they trend up almost no matter what. That's exactly why they're useless for steering.

---

## Frameworks (choose one, then forget it)

The framework is a presentation choice. The *underlying metric quality* (the eight-field schema) is what matters. Pick the framework the team will actually recognize and use, then move on.

### SMART (default fallback)

**Specific. Measurable. Achievable. Relevant. Time-bound.**

A criterion checklist, not a structure. Apply it to any metric to test quality.

- **Specific** — names the segment, behavior, or system precisely.
- **Measurable** — has a unit and an instrument.
- **Achievable** — order of magnitude is reasonable given resources and horizon. Stretch is fine; fantasy is not.
- **Relevant** — traces to a vision element.
- **Time-bound** — has a date.

Use as the default when no other framework fits. Most internal-tool, infra, and one-off initiatives are well-served by a plain SMART list.

### OKRs (Objectives and Key Results)

One **Objective** (qualitative, aspirational, ≤1 sentence) supported by 3–5 **Key Results** (quantitative, ambitious, dated).

```
Objective: Make first-day onboarding feel effortless for HR ops.
KR1: Median time-to-complete-day-1-setup drops from 3 days → 30 minutes by Q3 end.
KR2: ≥80% of new hires self-attest "easy" or "very easy" by Q3 end.
KR3: Onboarding-related support tickets drop ≥40% YoY.
```

**Use OKRs when:**
- The initiative rolls up to existing company/team OKRs.
- The team has a quarterly review rhythm and OKR literacy.

**Don't use OKRs when:**
- The team is small and OKR ceremony adds more overhead than alignment.
- The horizon is shorter than a quarter (the framework wastes effort).

**Common OKR failures:** sandbagging KRs to guarantee 100% achievement (defeats the purpose); using KRs as a task list ("ship feature X" — that's an output, not a key result); too many KRs (3–5 max — more dilutes focus).

### North Star + Inputs

One **North Star metric** (the one number that, if it moves, the business is winning) supported by 3–5 **input metrics** that the team can directly influence.

```
North Star: Weekly active completed-onboardings (a "completed onboarding" = new hire reaches "ready for day 2" within 24 hrs of start).
Inputs:
  - Median minutes from start → ready-for-day-2.
  - Activation rate (new hires who finish onboarding without HR touch).
  - Reopen rate (onboardings re-touched by HR after marked complete).
```

**Use North Star when:**
- The product is consumer or growth-focused and a single user behavior captures most of the value.
- You want the whole org to align on one number.

**Don't use North Star when:**
- The vision has multiple distinct user behaviors with no single dominant one (you'll force-fit and create a vanity composite).
- B2B / multi-stakeholder products where "the user" is several different people whose behaviors don't combine into one number.

### HEART (Google's UX framework)

**H**appiness, **E**ngagement, **A**doption, **R**etention, **T**ask success.

Five categories; pick metrics within each. Excellent for user-experience initiatives because it forces coverage of the qualitative dimension (Happiness) and the behavioral one (Task success) — both often missed in business-only frameworks.

| Category | Example metric |
|---|---|
| Happiness | NPS, CSAT, in-product survey |
| Engagement | Session frequency, depth, time-in-task |
| Adoption | New users completing core action in week 1 |
| Retention | Cohort retention at day 7 / 30 / 90 |
| Task success | Completion rate, error rate, time-on-task |

**Use HEART when:**
- The vision is user-experience or user-product oriented.
- Coverage matters more than focus.

**Don't use HEART when:**
- The vision is business-outcome (revenue, cost, risk) rather than user-experience — HEART under-weights business metrics.

### AARRR / Pirate Metrics

**A**cquisition, **A**ctivation, **R**etention, **R**eferral, **R**evenue. The funnel framework.

```
Acquisition: weekly new sign-ups by channel.
Activation: % of new users who reach "aha moment" within 7 days.
Retention: % of activated users still active at week 4.
Referral: invites sent per active user per month.
Revenue: ARPU in cohort.
```

**Use AARRR when:**
- The initiative is growth, funnel, or self-serve oriented.
- The customer journey is roughly linear.

**Don't use AARRR when:**
- The vision is internal tooling, infra, or platform — the funnel doesn't map.
- B2B with long sales cycles where the funnel lives in a CRM, not the product.

---

## Categories of Metrics (cover the right ground)

A balanced D02 typically draws from several categories. If the document leans heavily on only one, it's likely missing a dimension.

| Category | Examples | Watch out for |
|---|---|---|
| **Business** | Revenue, cost, gross margin, ROI, payback | Often delegated entirely to lagging — pair with a leading proxy. |
| **Product** | Activation, engagement, retention, NPS | Easy to fall into vanity. Apply the "would I be happy if only this moved?" test. |
| **User** | CSAT, qualitative interviews, support tickets, churn reasons | Qualitative inputs are valid metrics if defined and sampled consistently. |
| **Operational** | Latency, uptime, error rate, support volume | Often the most natural guardrail category. |
| **Quality** | Defect rate, regression rate, test coverage of contract | Useful guardrails; rarely useful primaries. |
| **Security / Compliance** | Vuln SLA, audit findings, incident rate | Almost always a guardrail or halt-condition, not a primary. |

For a typical product initiative: 1 business outcome (lagging) + 1–2 product behaviors (leading) + 1 user-quality measure + 1–2 operational guardrails ≈ a complete D02.

---

## Counter-Metrics and Guardrails (what we won't break)

Every primary metric can be gamed by ignoring something else. The guardrail is the *something else* the team commits not to ignore.

**Examples of paired primary + guardrail:**

| Primary | Likely game / side-effect | Guardrail |
|---|---|---|
| Sign-ups | Spam / low-quality cohorts | Activation rate within 7 days ≥ baseline |
| Onboarding speed | Skipping necessary steps | Onboarding-error rate ≤ baseline |
| Conversion rate | Aggressive UX dark patterns | Refund / unsubscribe rate ≤ baseline |
| API latency reduction | Cutting features that slow it | Functional contract test pass rate = 100% |
| Cost per user | Service quality dropped | p95 latency, support response time |
| Throughput | Burning the team out | On-call pages/week ≤ baseline |

**Halt conditions** are guardrails with teeth: a value at which we *stop* the initiative regardless of the wins on the primaries. Reserve for cases where the side effect would be unacceptable (security incident, fairness violation, runaway cost).

A vision with no guardrails is dangerous. If the user can't think of one, suggest a generic safety net (latency, error rate, support load) and confirm.

---

## Targets: Three Thresholds, Not One

A single target ("hit X by Y") flattens the gradient of how the initiative is actually doing. Three thresholds capture the shape:

- **Success** — what we're committing to.
- **Warning** — below success, above failure. "We're off track but not in trouble." Triggers a review.
- **Failure** — below this, we say the initiative did not work. Triggers a stop or a vision revisit.

```
Median time-to-complete-day-1-setup (currently 3 days):
  Success:  ≤ 30 minutes by 2026-12-31
  Warning:  31–60 minutes
  Failure:  > 90 minutes
```

This format also makes the distance from where you are now to where you need to be visible — and that distance is itself information about whether the target is plausible.

**Setting target values:** start with order of magnitude (5%, 20%, 50%, 2×, 10×) before precision. The user almost always knows the order; precision can come from a baseline measurement task.

---

## Common Traps (and how to spot them)

| Trap | Symptom | Fix |
|---|---|---|
| **Vanity** | Number that goes up no matter what | Apply the "would I be happy" test; replace with action-tied metric |
| **Output as outcome** | "Ship X" | Ask "what changes because we shipped X?" |
| **All-leading** | Every metric is a 1-week signal | Add at least one lagging outcome |
| **All-lagging** | Every metric is quarter-end | Add a weekly/biweekly leading proxy |
| **No baseline** | "Increase by X%" with no current value | Mark `Baseline: TBD — owner — by date`; commit to measure first |
| **No owner** | "The team owns this" | Force a named human |
| **No date** | "Eventually" / "this year-ish" | Force a date |
| **Composite no one trusts** | "Engagement score = 0.4·dau + …" | Replace with 2–3 plain metrics |
| **Goodhart's Law victim** | Metric became a target → got gamed | Add a counter-metric / guardrail |
| **No guardrails** | Only primaries listed | Add at least one — operational, quality, or security |
| **Targets miscalibrated** | All hit at 100%, every quarter | The targets aren't ambitious; the team is sandbagging |
| **Targets fantasy** | 10× improvement with no plausible path | Distance from baseline is unbridgeable in the horizon |
| **Metric no one looks at** | Saved in D02, never on a dashboard | Define cadence + reviewer at write time |

---

## Two Worked Examples

### Example A — User-experience initiative (HEART-shaped)

Vision excerpt (D01 §3): *"HR ops finishes day-1 setup for a new hire in under 30 minutes from a single workspace."*

| # | Metric | Type | Baseline | Target | Owner |
|---|---|---|---|---|---|
| 1 | Median minutes to complete day-1 setup | Lagging (primary outcome) | 3 days (≈1440 min) | ≤30 min by 2026-12-31; warning 31–60; failure >90 | <PM> |
| 2 | % of onboardings with zero HR-ops touch after kickoff | Leading | 12% | ≥60% by 2026-09-30 | <PM> |
| 3 | New-hire CSAT for day-1 (1–5 in-product survey) | Lagging | 3.1 | ≥4.3 by 2026-12-31; warning 3.8–4.2; failure <3.5 | <UX> |
| 4 | Onboarding-related support tickets / week | Lagging | 42 | ≤25 by 2026-12-31 | <Support lead> |
| **G1** | Onboarding-error rate (steps marked complete that need rework) | Guardrail | 4% | ≤4% (must not exceed); halt at >10% | <PM> |
| **G2** | p95 latency on workspace primary actions | Guardrail | 800ms | ≤1000ms | <Eng lead> |

This D02 has: a clear lagging outcome (#1), a leading behavioral signal (#2), a user-quality dimension (#3), a business-impact lagging metric (#4), an integrity guardrail (G1), and an operational guardrail (G2). Six metrics, all eight fields populated, balanced horizons.

### Example B — Internal infrastructure initiative (plain SMART list)

Vision excerpt (D01 §3): *"Backend services emit traces and structured logs in a consistent format, queryable in one tool."*

| # | Metric | Type | Baseline | Target | Owner |
|---|---|---|---|---|---|
| 1 | % of services emitting standardized traces | Leading | 0% | 100% by 2026-09-30 | <Platform lead> |
| 2 | Median time-to-root-cause for incidents (post-rollout) | Lagging | 47 min | ≤15 min by 2026-12-31 | <SRE lead> |
| 3 | Number of distinct logging tools dev teams must use | Output proxy (acceptable here — infra) | 4 | 1 by 2026-12-31 | <Platform lead> |
| **G1** | p99 service latency regression vs pre-instrumentation | Guardrail | n/a | ≤5% regression | <Eng lead> |
| **G2** | Per-service log volume cost | Guardrail | $X/mo | ≤1.3·X/mo | <Platform lead> |

For internal infra, OKR/HEART/AARRR all add ceremony without insight. SMART list works.

---

## A Sanity-Check Pass Before Saving

Read D02 as if you were:

1. **The team six months in, reviewing the dashboard at standup.** Can you tell whether the initiative is on track from the leading metrics alone?
2. **The executive at QBR.** Can you tell whether the initiative succeeded or failed from the lagging outcomes alone?
3. **A skeptical engineer.** Could you game any primary without tripping a guardrail? If yes, add a guardrail.
4. **The business owner if all primaries are green but the business hasn't moved.** Does the connection from D01 vision to D02 metrics survive scrutiny? If not, the metrics are vanity.

If any of these readers walks away unsure, the measurement system isn't done.

---

## When the User Resists Numbers

Common patterns and responses:

- *"I don't want to commit to a target — what if we miss?"* → That's why there are three thresholds. Missing success but staying above failure is a normal outcome of ambitious targets and should be expected, not punished. The point is to know where you are, not to guarantee victory.
- *"The metric is hard to measure today."* → Then the first commitment is the instrumentation. Mark `Baseline: TBD — owner — by <date>` and treat baseline measurement as the first piece of work.
- *"It depends on the user / context / season."* → Segment the metric. *"Median onboarding time among teams ≥50 employees"* beats *"average onboarding time."* If you can't even segment, the metric is too vague to commit to.
- *"Numbers feel cold for a vision about user delight."* → Operationalize delight: NPS ≥X, retention ≥Y, qualitative interviews showing Z themes. Quantitative + qualitative is fine; un-measurable is not.
- *"This is too many metrics."* → 3–7 is the band. Cut to the most load-bearing. The unsaid metrics aren't gone, they just aren't goal-bearing.

The skill exists because measurement systems fail in predictable ways. Numbers, dates, owners, and guardrails are non-negotiable. Push for them.
