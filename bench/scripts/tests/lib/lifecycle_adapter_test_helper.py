import re
from pathlib import Path
import sys


ADAPTER_SCRIPT = Path(__file__).resolve().parents[2] / "lifecycle-worker-adapter.sh"
ADAPTER_LIB = ADAPTER_SCRIPT.parent / "lib"


def load_adapter_module():
    sys.path.insert(0, str(ADAPTER_LIB))
    content = ADAPTER_SCRIPT.read_text(encoding="utf-8")
    match = re.search(r"<<'PYEOF'\n(.*)\nPYEOF", content, re.DOTALL)
    assert match is not None, "Could not find embedded Python in lifecycle-worker-adapter.sh"
    source = re.sub(r"\nmain\(\)\s*$", "\n", match.group(1))
    namespace = {}
    exec(source, namespace)
    return namespace
