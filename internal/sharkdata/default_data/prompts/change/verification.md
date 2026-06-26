{{template "_resume_preamble" .}}Verify change {{.id}}: "{{.title}}".

Check for existing verification report. If report exists with PASS verdict, advance immediately.

---

VERIFY:
- [ ] Run: make fmt && make lint && make test
- [ ] Change matches description
- [ ] No regressions

DECISION:
- ALL PASS → shark status advance {{.id}}
- ANY FAIL → shark status set {{.id}} development --reason "<findings>"
