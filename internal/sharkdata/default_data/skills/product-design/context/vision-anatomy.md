# The Anatomy of a Great Vision

Reference reading for `workflows/d01-vision.md`. The workflow is the procedure; this file explains *what each component is for*, *what makes it good*, and *what makes it fail*. Read it once before running the workflow. Re-read the relevant section if the user's answer to a phase feels thin.

A vision is not a strategy doc, a roadmap, a feature list, or a marketing tagline. It is a **shared, falsifiable agreement about what we're trying to make true and for whom** — written down so every downstream decision (priorities, trade-offs, scope cuts) can be checked against it.

A great vision has nine components. Each one closes a specific failure mode. If the workflow skips one, that failure mode shows up later, more expensively.

---

## 1. Target user — *who*

**Purpose:** Anchor the entire vision to a specific person whose life or work changes if this succeeds.

**What good looks like:**
- Named segment, persona, or job title: *"Mid-market HR ops managers handling onboarding for 50–500 employees."*
- One primary segment named — others may exist but are explicitly secondary.
- Specific enough that a stranger could go interview five of them tomorrow.

**Common failures:**
- *"Users."* / *"The organization."* — Means nothing; provides no boundary for design or priority calls.
- *"Everyone."* — Means no one is happy enough to switch from their current alternative.
- *"Power users and casual users and admins…"* — Three personas with conflicting needs guarantees a compromised product.

**Diagnostic question:** *"If we had budget to interview ten of these users next week, who exactly do we call?"* If the user can't answer, the segment isn't real yet.

---

## 2. Problem / unmet need — *what hurts*

**Purpose:** State the pain or unmet need precisely enough that someone can decide whether a proposed solution actually addresses it.

**What good looks like:**
- A behavior or situation, not an abstraction. *"Onboarding a new hire takes 3 days of HR ops time spread across 7 systems"* beats *"onboarding is inefficient."*
- Names the **trigger** (when does the pain show up), the **current alternative** (what they do today), and the **cost of that alternative** (time, money, error, frustration).
- Validated by something — interviews, tickets, sales loss, data — even if the validation is rough.

**Common failures:**
- **Solution dressed as problem.** *"Users need a Kanban board"* is a solution; the underlying problem might be "teams can't see who's doing what this week."
- **Pain too small to motivate change.** Users won't switch from a working alternative for a 5% improvement; the pain has to be large enough to overcome switching costs.
- **No alternative described.** If you don't know what users do today, you don't yet understand the problem.

**Diagnostic question:** *"What does the workaround cost them in time, money, or frustration?"* A user who shrugs at the cost of their current workaround does not have a problem worth your initiative.

---

## 3. Why now — *timing*

**Purpose:** Justify why this initiative belongs in *this* horizon rather than three years ago or three years from now. Vision has a shelf life; "why now" anchors it.

**What good looks like:**
- A specific change in market, technology, regulation, user behavior, or competitive landscape that opens the window.
- Or an honest admission that nothing changed — the problem has always been there, and the timing is internal (capacity, strategic shift). This is valid but flag it as a timing risk.

**Common failures:**
- **No why-now.** The team is building it because it's on the roadmap. The roadmap is on the roadmap. Etc. Risk of being deprioritized the moment something fresher appears.
- **Fake urgency.** "Competitors are doing it" without a clear loss if we're late.

**Diagnostic question:** *"What changed recently that makes this the right moment?"* If the answer is genuinely nothing, name it as such — sometimes the right call is still to do it now, but the team should know the timing is internal.

---

## 4. Desired outcome — *future state*

**Purpose:** Describe what is true if we succeed, from the user's and business's point of view.

**What good looks like:**
- Behavioral change ("HR ops finishes onboarding in 30 minutes") or business change ("first-year retention rises from 60% to 75%").
- Tense: written in the future indicative ("HR ops *finishes…*"), not the conditional ("HR ops *might be able to…*").
- Concrete enough that §11 (success criteria, in D02) can be derived from it directly.

**Common failures:**
- **Adjective soup.** "Modern, scalable, intuitive, delightful, cloud-native, AI-powered." Each adjective hides whether anything actually changed.
- **Output disguised as outcome.** "We will have shipped X." Shipping X is a milestone, not an outcome. The outcome is *what users do or what the business gains because we shipped X*.
- **Vague time horizon.** A vision with no horizon can't be evaluated.

**Diagnostic question:** *"If this works perfectly, what will be observably different on a Tuesday afternoon a year from now?"*

---

## 5. Primary value — *what kind of win*

**Purpose:** Name the dominant axis of value so trade-offs can be made consistently.

**Categories** (one of these is dominant):
- **Save time** — make a slow thing fast.
- **Save money** — reduce cost or waste.
- **Make money** — open new revenue or expand existing.
- **Reduce risk** — compliance, security, reliability, regulatory.
- **Unlock new capability** — enable something previously impossible.
- **Improve experience** — qualitative win where the others don't fit (use sparingly; usually one of the others is the real driver).

**Why it matters:** Trade-offs in design and scope require knowing which axis you're optimizing. A "save time" vision should never accept a feature that makes the experience slower, even if it's prettier.

**For whom:** value can accrue to the user, the business, or both. Be explicit. A user-only win without a business case will lose funding; a business-only win without user adoption will fail to land.

**Common failures:**
- **All values claimed equally.** Every category checked. Means no axis is being optimized — and trade-offs will get made by accident.
- **Value claimed for the user when it actually accrues to the business.** ("Streamlined onboarding for users" when the actual win is reduced support cost.) Pretending otherwise corrupts later research and design.

---

## 6. Differentiation — *why us / why this approach*

**Purpose:** Surface what makes *this* version of the solution worth doing, given that alternatives exist (competitors, the status quo, a buy option, another team's project).

**What good looks like:**
- Names 2–3 alternatives explicitly, including the **status quo** (which is always an alternative).
- States a differentiator that would survive a skeptic's pushback — a real capability, asset, distribution advantage, data advantage, or position.
- **Names what we're worse at.** Real differentiation has trade-offs. Salesforce isn't simple; Notion isn't enterprise-rigid. If a vision claims best-in-class on every axis, it's fantasy.

**Common failures:**
- **No alternatives.** Greenfield work always competes with the status quo and with "do nothing."
- **Differentiator is a feature.** "We have AI" is not a differentiator if everyone has AI. *How* you have it, and what that lets the user do that alternatives don't, is.
- **No accepted weakness.** A team unwilling to be worse at anything will end up mediocre at everything.

**Diagnostic question:** *"To win on our differentiator, what are we explicitly choosing to be worse at than the alternatives?"* If silence, the vision has a hole.

---

## 7. Scope and non-goals — *the box*

**Purpose:** Draw the edge. Vision without an edge devolves into a wishlist that no team can deliver.

**What good looks like:**
- **In scope:** 3–7 outcome bullets. Outcome-shaped (something is true at the end), not feature-shaped (we built X).
- **Non-goals:** at least one explicit "we will NOT do X, even if asked." Each non-goal carries a brief reason ("not our user," "wrong horizon," "buys vs. builds").
- The non-goals list often *defines* the vision more precisely than the in-scope list.

**Common failures:**
- **No non-goals.** Means everything is potentially in scope. Means the team will be stretched and the vision will quietly inflate.
- **Non-goals as easy throwaways.** "We will not build a mobile app" when no one suggested a mobile app. Real non-goals are things people *would* otherwise expect.

**Diagnostic question:** *"What might someone reasonably expect us to include but we're explicitly leaving out?"* A user who can't name even one of these is not yet committed to the vision.

---

## 8. Constraints — *the immovable*

**Purpose:** Surface deadlines, budgets, regulatory floors, and dependencies that bound the vision's feasible region.

**What good looks like:**
- **Time:** hard deadline (with reason — market window, contractual, regulatory) or open-ended (and honest about it).
- **Resources:** order-of-magnitude headcount/budget/vendor commitments. Precise numbers nice; rough buckets sufficient.
- **Regulatory / compliance / security:** anything legal, security, privacy, or compliance considers non-negotiable. These hide until late and then blow up scope; surface them now.
- **Dependencies:** other teams, vendors, platform shifts, data availability.

**Common failures:**
- **Hidden deadlines.** "End of year" stated casually but actually contractual. The team learns at week 9.
- **Constraint-free vision.** Every initiative has constraints. A vision with none is incomplete.
- **Compliance ignored.** Treated as a downstream problem until it forces a redesign in QA.

---

## 9. Stakeholders, assumptions, and risks — *the bets*

**Purpose:** Document who has decision power, what we're betting on, and what could kill it.

**Stakeholders** — name by role: who **approves**, who **must be informed**, who is **affected**. Steering committees obscure accountability; force named individuals.

**Assumptions** — the beliefs that, if false, invalidate the vision. Common ones:
- *Users actually want this* (vs. would say yes in an interview but never switch).
- *The technology can do this affordably* (vs. demos at toy scale).
- *The team has or can acquire the skills*.
- *Regulation will not block it*.
- *The economic case holds at projected volume*.

A vision with zero stated assumptions is a vision pretending to be a fact.

**Risks** — name 2–3 most likely failure modes, each with an early signal that would indicate the risk is materializing. *"Risk: integration with payroll vendors is harder than expected. Early signal: first vendor sandbox eval blocks past week 2."*

**Definition of failure** — surprisingly clarifying and surprisingly rare. *"We will know this didn't work if at the horizon: <observable signal>, <signal>, <signal>."* Forces the team to commit to what failure looks like, which prevents endless face-saving reinterpretation.

---

## Connective Tissue: The Vision Statement Paragraph

The first section of D01 is a single paragraph that compresses the above into something a stranger can understand in 30 seconds. The Geoffrey Moore form works well:

> *"For [target user] who [problem / unmet need], [initiative name] is a [category] that [primary value]. Unlike [main alternative], we [differentiator]."*

**Worked example:**

> *"For mid-market HR ops managers handling onboarding for 50–500 employees, who today spend 3 days per hire chasing tasks across 7 systems, **Onboard** is a unified onboarding workspace that completes a new hire's day-1 setup in under 30 minutes. Unlike full-suite HRIS platforms, we don't try to replace payroll or benefits admin — we orchestrate them, which lets us land in 4 weeks instead of 6 months."*

This paragraph names: target user, problem with cost, category, primary value (with a concrete outcome), main alternative, and differentiator (with the explicit *non-goal* of replacing payroll baked in).

---

## Anti-Patterns (Stop and Re-elicit)

If the draft has any of these, the vision isn't ready — return to elicitation rather than smoothing them over in prose:

| Anti-pattern | What it sounds like | Why it's broken |
|---|---|---|
| Adjective soup | "Modern, scalable, intuitive, delightful, AI-powered." | No observable behavior; nothing to disagree with; nothing to measure. |
| Audience = everyone | "All users / the whole company / all enterprises." | No segment to interview, design for, or position against. |
| No non-goals | In-scope list only. | No edge → unbounded scope inflation. |
| No alternatives | "There's nothing like this." | Even greenfield competes with the status quo. |
| Solution as vision | "Build a React app that does X." | Conflates output with outcome; locks in tech before need is understood. |
| Strategy doc as vision | All company strategy, no specific user/problem/outcome. | Wrong artifact; will not guide design or trade-offs. |
| Best at everything | Differentiator with no accepted weakness. | Fantasy; team will be mediocre everywhere. |
| Pain too small | User shrugs at the cost of their workaround. | Won't motivate behavior change. |
| Adjective deadline | "Soon." / "This year-ish." | Cannot be evaluated. |
| Zero assumptions | Confidence with no acknowledged bets. | Pretends fact, hides real risk. |

---

## A Sanity-Check Pass Before Saving

Read the draft as if you were:

1. **A new engineer joining in week 6.** Can you tell what we're building, for whom, and what's out of scope?
2. **A skeptical executive.** Can you spot the bet? Is there a clear outcome you'd be willing to be evaluated on?
3. **A competitor.** Could you identify our weakness and how to attack it? (If no — we have no real position.)
4. **The team six months from now, mid-execution, deciding whether to add a feature.** Can D01 alone tell you yes or no?

If any of these readers walks away confused, the vision is not done.

---

## When the User Resists

Some users resist the boundaries this format demands. Common resistance patterns and responses:

- *"I don't want to commit to a number / segment / non-goal yet."* → Mark as `TBD — owner: <name> — by: <date>`. The TBD is the commitment to commit.
- *"It's all important; I can't pick a primary value."* → Force a rank: "If we could only optimize one, which?" The unranked vision will still rank itself in execution — better to do it now consciously.
- *"The vision should be inspirational, not constrained."* → Inspiration is in §1 (the paragraph). The rest is the contract that makes inspiration deliverable. A vision with no contract is a poster.
- *"We'll figure scope out as we go."* → Sometimes true for early discovery; D01 then explicitly frames itself as a discovery vision with a tight horizon and a planned re-do. State that openly rather than pretending the vision is settled.

The skill exists because vision documents fail in predictable ways. Names matter. Numbers matter. Non-goals matter. Push for them.
