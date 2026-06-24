{{template "_resume_preamble" .}}RESUME assessment for feature {{.id}}: "{{.title}}".

Check feature metadata: {{template "get_json" .}}. If complexity_tier already assigned with routing decision made, execute the routing immediately (SIMPLE->ready_for_task_generation, STANDARD->ready_for_specification, COMPLEX->ready_for_research).

Otherwise, continue with full assessment per ready_for_assessment instructions.
