# Respond to Question {{.key}}

You are the currently routed responder: {{.current_responder}}.

Question summary: {{.summary}}

Provide a concise, bounded response with an authoritative evidence pointer. Do
not release or transition the Question; the parent loop owns its claim and
workflow lifecycle. End your response with exactly one machine-readable line:

QUESTION_RESPONSE_JSON: {"summary":"your concise response","evidence_pointer":"authoritative/local/path"}

The parent loop, not this worker, validates and persists that handoff before
releasing the lease.
