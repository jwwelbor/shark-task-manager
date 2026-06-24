{{/* ===================================================================== */}}
{{/* _client.tmpl — System-specific client execution commands               */}}
{{/*                                                                         */}}
{{/* Customize this file to match your AI execution environment.             */}}
{{/* Swap out "client_exec" to change how agents are spawned system-wide.    */}}
{{/*                                                                         */}}
{{/* Examples:                                                               */}}
{{/*   /claude-exec  (default — Claude Code CLI)                             */}}
{{/*   /copilot-exec (GitHub Copilot workspace agents)                       */}}
{{/*   /agent-exec   (custom wrapper)                                        */}}
{{/* ===================================================================== */}}

{{/* Base execution command — override this one define to change everything */}}
{{define "client_exec"}}/claude-exec{{end}}

{{/* ===== Red-team / UAT client invocation ===== */}}

{{/* Spawn a fresh UAT red-team session for the given entity */}}
{{define "uat_client"}}{{template "client_exec" .}} --agent uat-agent{{end}}

{{/* Inline hint shown in templates so readers know which command to run */}}
{{define "uat_spawn_hint"}}
To launch the red-team UAT reviewer, run:
  {{template "uat_client" .}}
{{end}}
