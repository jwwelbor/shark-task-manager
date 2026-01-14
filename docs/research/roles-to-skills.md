## SDLC skill categories

1. **Product & Domain**

* Skills: problem framing, domain knowledge, value/risk tradeoffs, roadmap, prioritization, OKRs, stakeholder mgmt
* Outputs: vision, strategy, roadmap, prioritized backlog, success metrics

2. **Discovery & Requirements**

* Skills: user research, requirements elicitation, story writing, acceptance criteria, process mapping, edge cases
* Outputs: user journeys, PRDs, epics/stories, ACs, non-functional requirements (NFRs)

3. **UX / UI / Content Design**

* Skills: interaction design, visual design, accessibility, information architecture, content design, prototyping
* Outputs: wireframes, hi-fi comps, prototypes, design system artifacts, UX specs

4. **Architecture & Technical Design**

* Skills: system design, API design, data modeling, integration patterns, scalability, reliability, tradeoff analysis
* Outputs: architecture diagrams, ADRs, interface contracts, technical designs

5. **Implementation (Build)**

* Skills: coding, code review, refactoring, debugging, performance, dependency management
* Outputs: working software, maintainable codebase, unit tests

6. **Quality Engineering**

* Skills: test strategy, test automation, exploratory testing, test data mgmt, CI quality gates
* Outputs: test plans, automated test suites, quality reports, defect triage

7. **Security, Privacy, Compliance**

* Skills: threat modeling, secure coding, identity/access, vulnerability mgmt, privacy controls, regulatory alignment
* Outputs: security requirements, threat models, risk sign-offs, audit artifacts

8. **DevOps / Platform / Release Engineering**

* Skills: CI/CD, infrastructure as code, environments, deployment strategies, rollback, release governance
* Outputs: pipelines, infra, deployment playbooks, release notes

9. **Operations & Reliability**

* Skills: observability, monitoring/alerting, incident response, SLOs/SLAs, capacity planning, runbooks
* Outputs: dashboards, alerts, on-call procedures, postmortems, reliability improvements

10. **Data & Analytics (when applicable)**

* Skills: data pipelines, BI, experimentation, telemetry, model/data governance (if ML)
* Outputs: event schemas, reports, insights, experimentation results

---

## Role-to-skill mapping (typical “primary ownership”)

| Skill category                  | Primary roles                                     | Frequent partners              |
| ------------------------------- | ------------------------------------------------- | ------------------------------ |
| Product & Domain                | Product Manager / Product Owner                   | Tech Lead, UX, Sales/CS, Execs |
| Discovery & Requirements        | Product Owner, Business Analyst                   | UX Research, Tech Lead, QA     |
| UX / UI / Content Design        | Product Designer, UX Researcher, Content Designer | PO/PM, Frontend Eng            |
| Architecture & Technical Design | Solution/Software Architect, Tech Lead            | Security, Data, SRE/Platform   |
| Implementation                  | Software Engineers (FE/BE/Mobile)                 | Tech Lead, QA, Designer        |
| Quality Engineering             | QA / SDET / QE                                    | Devs, PO, SRE                  |
| Security, Privacy, Compliance   | Security Engineer / AppSec / GRC                  | Architect, Devs, Legal/Privacy |
| DevOps / Platform / Release     | DevOps / Platform Engineer, Release Manager       | SRE, Devs, QA                  |
| Operations & Reliability        | SRE / Operations / Support Engineering            | Devs, Platform, PM             |
| Data & Analytics                | Data Engineer, Analyst, Data Scientist            | PM, Devs, Platform             |

---

## Quick “who does what” by SDLC phase (high-level)

* **Ideation/Discovery:** PM/PO + UX Research + Tech Lead/Architect
* **Definition (requirements/NFRs):** PO/BA + Architect + QA + Security
* **Design:** Product Design/UX + Frontend + PO
* **Build:** Engineers + Tech Lead + QA/QE
* **Verify:** QA/QE + Devs + Security (scans/review)
* **Release:** DevOps/Platform + Release Manager + QA + PO
* **Operate/Improve:** SRE/Ops + Devs + PM (feedback loop)

---

## One useful rule of thumb

* **PO/PM owns “what & why.”**
* **Tech Lead/Architect owns “how & tradeoffs.”**
* **Engineering owns “build it right.”**
* **QA/QE owns “prove it works.”**
* **Platform/SRE owns “ship it safely & keep it healthy.”**
* **Security owns “don’t get owned.”**

If you tell me what kind of org you’re modeling (startup vs enterprise, regulated vs not, and whether you have dedicated SRE/QA), I can tailor this into a cleaner competency matrix (levels 1–5) or a RACI you can hand to an architect/team lead.
