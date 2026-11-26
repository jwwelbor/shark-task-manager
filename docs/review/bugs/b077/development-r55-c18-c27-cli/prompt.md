Execute exactly this candidate-bound B077 scenario and report its exit code and JSON evidence. Do not modify the checkout or alter the scenario.

From this checkout, build the current Shark candidate to `/tmp/b077-c18-c27-proof/shark` with `GOCACHE=/tmp/b077-c18-c27-proof/go-cache GOTMPDIR=/tmp/b077-c18-c27-proof go build -trimpath -buildvcs=false -o /tmp/b077-c18-c27-proof/shark ./cmd/shark`. Then execute `B077_SHARK=/tmp/b077-c18-c27-proof/shark python3 /tmp/b077-c18-c27-proof/scenario.py`. The private Go cache and temporary files must stay under `/tmp/b077-c18-c27-proof` because the shared cache is read-only in this execution sandbox.

The scenario initializes an isolated project, exercises real `shark question` and `shark claim` CLI commands for C18/C27, and independently checks rejected-write state through read-only SQLite queries. Do not run any other commands. The fixture and question data are disposable under `/tmp`.
