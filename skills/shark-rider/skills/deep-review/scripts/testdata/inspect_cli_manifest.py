"""Record what an invoked reviewer can resolve, independently of its canned reply."""
import hashlib
import json
from pathlib import Path
import sys


def inspected_files(prompt):
    prefix = "FROZEN_SCOPE_REFERENCE: "
    for line in prompt.splitlines():
        if line.startswith(prefix):
            ref = json.loads(line[len(prefix):])
            scope = Path(ref["path"]).read_bytes()
            if hashlib.sha256(scope).hexdigest() != ref["sha256"]:
                raise ValueError("reviewer received a scope with the wrong digest")
            reads = []
            for name in json.loads(scope)["changed_files"]:
                path = name.encode("utf-8", "surrogateescape")
                with open(path, "rb") as member:
                    reads.append([path.hex(), member.read().hex()])
            return reads
    return []


if __name__ == "__main__":
    cli, log, reply, status, prompt = sys.argv[1:]
    with open(log, "a", encoding="utf-8") as output:
        print(json.dumps([cli, inspected_files(prompt)]), file=output)
    print(Path(reply).read_text())
    sys.exit(int(status))
