# Replay Interaction Preamble

This session is a REPLAYED run. `AskUserQuestion`, `WebSearch`, and
`WebFetch` are unavailable to you in this session. Every human elicitation,
approval, or decision, and every market or technical research lookup, MUST
be resolved through the replay resolver instead.

Whenever the current step would normally call for one of those:

    "$REPLAY_ANSWER_SCRIPT" --bundle "$REPLAY_BUNDLE_PATH" --stage <D0X> --kind <human_question|research_query> --topic <topic_key>

- `<D0X>` is the current artifact stage (`D01`-`D05`).
- `<human_question>` is the kind for elicitation, approvals, and human
  decisions; `research_query` is the kind for market and technical research.
- `<topic_key>` is a short label for what is being asked (for example
  `target_user` or `success_metric`).

The command prints the authorized response on stdout and exits `0` on a
match. A non-zero exit means the request could not be resolved from the
authorized response set — stop the current step immediately, report the
resolver's error output verbatim, and do not guess, paraphrase, invent, or
proceed without a resolved response.

This preamble routes interaction only. It adds no product-design
methodology, evidence, or content of its own.
