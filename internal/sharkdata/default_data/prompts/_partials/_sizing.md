{{/* ===== Sizing Guidance =====
     Reusable size-specification block for create-instruction templates.
     Fibonacci values: 1|2|3|5|8|13  (t-shirt: XS|S|M|L|XL|XXL).
     Pass --size to shark create on EVERY new task/feature/epic.
*/}}
{{define "_sizing_scale"}}SIZE SCALE (Fibonacci or t-shirt — both accepted by --size):
- 1 / XS  — trivial; <1 hour of focused work; one-line change, doc tweak
- 2 / S   — small; a few hours; single file or small set of files
- 3 / M   — medium; ~1 day; cohesive change across a few files, normal complexity
- 5 / L   — large; 2-3 days; multiple components, moderate unknowns. Consider splitting.
- 8 / XL  — very large; ~1 week; many components or significant unknowns. SHOULD be split.
- 13 / XXL — too large; MUST be split into smaller items before work begins.{{end}}


{{define "_sizing_task"}}REQUIRED: Every task must carry a size. Pass --size=<1|2|3|5|8|13> (or XS|S|M|L|XL|XXL) on {{template "create_task" .}}. Aim for 1/XS – 3/M per task; if you reach 5/L or higher, decompose before creating.

{{template "_sizing_scale"}}{{end}}

{{define "_sizing_feature"}}REQUIRED: Every feature must carry a size. Pass --size=<1|2|3|5|8|13> (or XS|S|M|L|XL|XXL) on {{template "create_feature" .}}. Features sized 8/XL or 13/XXL must be split before decomposition completes.

{{template "_sizing_scale"}}{{end}}
