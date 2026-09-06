{{template "advance_preamble" .}}

Close out sprint {{.id}}: "{{.title}}".

{{include: skills/sprint-analytics/SKILL.md}}

{{template "_product_critical_path_guard" .}}

Read `shark sprint summary {{.id}} --detailed` and velocity data, synthesize the
retrospective, and review carryover. Archiving is gated on explicit user
confirmation — release outcome `pass` only after the retro is confirmed.
