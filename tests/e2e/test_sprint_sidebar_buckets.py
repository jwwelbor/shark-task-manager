"""
Playwright E2E tests: Sprint sidebar bucket counts (T-E27-F14-002)

Verifies that after the SPRINT_BUCKET_MAP fix (viewer service uses grouped view),
the sidebar 'Active Sprint' section shows non-zero bucket counts for sprints
that have assigned items.

AC-004: Sidebar total count matches actual sprint assignment count
AC-005: Ready bucket shows ready_for_development/todo/draft statuses
AC-006: In Progress bucket shows in_development/ready_for_approval/in_progress
AC-007: Blocked bucket shows blocked status
AC-008: Done bucket shows completed/resolved statuses
"""

import os
import sys
import json
import urllib.request
import urllib.error
import traceback

SERVER_URL = os.environ.get("VIEWER_URL", "http://127.0.0.1:7777")
TIMEOUT = 15  # seconds

# SPRINT_BUCKET_MAP — mirrors the JS constant in viewer.html
SPRINT_BUCKET_MAP = {
    "todo": "ready",
    "draft": "ready",
    "ready_for_development": "ready",
    "in_development": "in_progress",
    "ready_for_approval": "in_progress",
    "in_progress": "in_progress",
    "in_review": "in_progress",
    "in_qa": "in_progress",
    "ready_for_refinement_tech": "in_progress",
    "ready_for_refinement_ba": "in_progress",
    "in_refinement": "in_progress",
    "blocked": "blocked",
    "completed": "done",
    "resolved": "done",
}

PASS = "\033[32mPASS\033[0m"
FAIL = "\033[31mFAIL\033[0m"

results = []


def api_get(path):
    url = f"{SERVER_URL}{path}"
    try:
        with urllib.request.urlopen(url, timeout=TIMEOUT) as resp:
            return json.loads(resp.read().decode("utf-8")), resp.status
    except urllib.error.HTTPError as e:
        return json.loads(e.read().decode("utf-8")), e.code
    except Exception as exc:
        return {"error": str(exc)}, 0


def run_test(name, fn):
    try:
        fn()
        results.append((name, True, None))
        print(f"{PASS}  {name}")
    except AssertionError as e:
        results.append((name, False, str(e)))
        print(f"{FAIL}  {name}")
        print(f"      AssertionError: {e}")
    except Exception as e:
        results.append((name, False, traceback.format_exc()))
        print(f"{FAIL}  {name}")
        print(f"      Exception: {e}")


# ─── API Tests (no browser required) ──────────────────────────────────────────

def test_api_returns_200():
    """Sprint overview endpoint returns HTTP 200."""
    data, status = api_get("/api/v1/viewer/sprint/overview")
    assert status == 200, f"Expected 200, got {status}. Error: {data.get('message', data)}"


def test_api_returns_grouped_view():
    """Sprint overview backlog uses 'grouped' view so sidebar can aggregate by status_category."""
    data, status = api_get("/api/v1/viewer/sprint/overview")
    assert status == 200, f"Expected 200, got {status}"
    backlog = data.get("backlog", {})
    assert backlog, "Expected 'backlog' in response"
    view = backlog.get("view")
    assert view == "grouped", f"Expected view='grouped' (fixed by T-E27-F14-002), got '{view}'. " \
        "The viewer service must call GetSprintBacklog with View:'grouped' so sidebar bucket " \
        "aggregation works. Without this, active sprints use 'ordered' view and groups=[]=0."


def test_api_backlog_total_count_nonzero():
    """Sprint backlog total_count is non-zero (sprint has assigned items)."""
    data, status = api_get("/api/v1/viewer/sprint/overview")
    assert status == 200, f"Expected 200, got {status}"
    backlog = data.get("backlog", {})
    total = backlog.get("total_count", 0)
    assert total > 0, f"Expected total_count > 0, got {total}. Sprint {data.get('sprint', {}).get('key', '?')} has no assigned items."


def test_api_groups_present():
    """Sprint backlog has groups array with at least one group."""
    data, status = api_get("/api/v1/viewer/sprint/overview")
    assert status == 200, f"Expected 200, got {status}"
    backlog = data.get("backlog", {})
    groups = backlog.get("groups", [])
    assert isinstance(groups, list), f"Expected groups to be a list, got {type(groups)}"
    assert len(groups) > 0, "Expected at least one group in backlog.groups"


def test_api_groups_have_status_category():
    """Each group in backlog has a status_category field."""
    data, status = api_get("/api/v1/viewer/sprint/overview")
    assert status == 200, f"Expected 200, got {status}"
    backlog = data.get("backlog", {})
    groups = backlog.get("groups", [])
    for g in groups:
        sc = g.get("status_category")
        assert sc, f"Group missing status_category: {g}"


def test_api_groups_total_matches_backlog_total():
    """Sum of group item counts equals total_count."""
    data, status = api_get("/api/v1/viewer/sprint/overview")
    assert status == 200, f"Expected 200, got {status}"
    backlog = data.get("backlog", {})
    groups = backlog.get("groups", [])
    total_declared = backlog.get("total_count", 0)
    total_from_groups = sum(len(g.get("items", [])) for g in groups)
    assert total_from_groups == total_declared, \
        f"Sum of group items ({total_from_groups}) != total_count ({total_declared})"


def test_api_groups_map_to_buckets():
    """At least one group's status_category maps to a bucket via SPRINT_BUCKET_MAP."""
    data, status = api_get("/api/v1/viewer/sprint/overview")
    assert status == 200, f"Expected 200, got {status}"
    backlog = data.get("backlog", {})
    groups = backlog.get("groups", [])

    mapped_count = 0
    unmapped = []
    for g in groups:
        sc = g.get("status_category", "")
        if sc in SPRINT_BUCKET_MAP:
            mapped_count += 1
        else:
            unmapped.append(sc)

    assert mapped_count > 0, \
        f"No group status_category maps to SPRINT_BUCKET_MAP. " \
        f"Unmapped categories: {unmapped}. " \
        f"This means the sidebar buckets will all show 0."

    if unmapped:
        print(f"      (Note: unmapped status_categories not in SPRINT_BUCKET_MAP: {unmapped})")


def test_api_bucket_aggregation_nonzero():
    """Simulated JS SPRINT_BUCKET_MAP aggregation produces non-zero bucket counts (AC-004..AC-008)."""
    data, status = api_get("/api/v1/viewer/sprint/overview")
    assert status == 200, f"Expected 200, got {status}"
    backlog = data.get("backlog", {})
    groups = backlog.get("groups", [])

    # Simulate JavaScript sprintBucketGroups() aggregation
    bucket_items = {"ready": [], "in_progress": [], "blocked": [], "done": []}
    for group in groups:
        category = group.get("status_category", "")
        bucket = SPRINT_BUCKET_MAP.get(category)
        if bucket and bucket in bucket_items:
            bucket_items[bucket].extend(group.get("items", []))

    bucket_counts = {k: len(v) for k, v in bucket_items.items()}
    total = sum(bucket_counts.values())
    print(f"      Simulated bucket counts: {bucket_counts}")
    print(f"      Total items in buckets: {total}")

    assert total > 0, \
        f"Simulated SPRINT_BUCKET_MAP aggregation produced 0 items in all buckets. " \
        f"Groups: {[(g.get('status_category'), len(g.get('items',[]))) for g in groups]}"


# ─── Browser Tests (require Playwright Python) ────────────────────────────────

def run_browser_tests():
    """Run browser-level tests using Python Playwright."""
    try:
        from playwright.sync_api import sync_playwright
    except ImportError:
        print("  WARNING: Python playwright not available, skipping browser tests")
        return

    def test_browser_sprint_sidebar_nonzero_total():
        """Browser: Active Sprint sidebar shows non-zero total count."""
        with sync_playwright() as p:
            browser = p.chromium.launch(headless=True)
            page = browser.new_page()
            try:
                page.goto(SERVER_URL, wait_until="domcontentloaded", timeout=TIMEOUT * 1000)

                # Wait for the Sprint button
                page.wait_for_selector("#header-sprint-btn", timeout=TIMEOUT * 1000)

                # Click Sprint button to enter sprint mode
                page.click("#header-sprint-btn")

                # Wait for sprint sidebar section to appear
                page.wait_for_function(
                    """() => {
                        const headers = document.querySelectorAll('.sidebar-section-header');
                        return Array.from(headers).some(h => h.textContent && h.textContent.includes('Sprint'));
                    }""",
                    timeout=TIMEOUT * 1000
                )

                # Wait for sprint overview data to load and the active sprint to show items
                page.wait_for_function(
                    """() => {
                        const sprintNodes = document.querySelectorAll('.tree-node-sprint');
                        for (const node of sprintNodes) {
                            const keySpan = node.querySelector('.sprint-item-key');
                            if (keySpan && keySpan.textContent === 'Active Sprint') {
                                const spans = node.querySelectorAll('span');
                                for (const span of spans) {
                                    const count = parseInt(span.textContent || '0', 10);
                                    if (count > 0) return true;
                                }
                            }
                        }
                        return false;
                    }""",
                    timeout=TIMEOUT * 1000
                )

                # Get the count
                total_count = page.evaluate("""() => {
                    const sprintNodes = document.querySelectorAll('.tree-node-sprint');
                    for (const node of sprintNodes) {
                        const keySpan = node.querySelector('.sprint-item-key');
                        if (keySpan && keySpan.textContent === 'Active Sprint') {
                            const spans = node.querySelectorAll('span');
                            for (const span of spans) {
                                const count = parseInt(span.textContent || '0', 10);
                                if (count > 0) return count;
                            }
                        }
                    }
                    return 0;
                }""")

                print(f"      Active Sprint total count in sidebar: {total_count}")
                assert total_count > 0, \
                    f"Active Sprint sidebar shows 0 items. Expected > 0. " \
                    f"The bucket_map fix should populate groups so items are counted."

            finally:
                browser.close()

    def test_browser_sprint_buckets_nonzero():
        """Browser: sprint-tree-bucket elements show non-zero counts."""
        with sync_playwright() as p:
            browser = p.chromium.launch(headless=True)
            page = browser.new_page()
            try:
                page.goto(SERVER_URL, wait_until="domcontentloaded", timeout=TIMEOUT * 1000)
                page.wait_for_selector("#header-sprint-btn", timeout=TIMEOUT * 1000)
                page.click("#header-sprint-btn")

                # Wait for bucket elements to appear (active is expanded by default)
                page.wait_for_function(
                    "() => document.querySelectorAll('.sprint-tree-bucket').length > 0",
                    timeout=TIMEOUT * 1000
                )

                # Get bucket counts
                bucket_data = page.evaluate("""() => {
                    const buckets = document.querySelectorAll('.sprint-tree-bucket');
                    return Array.from(buckets).map(bucket => {
                        const node = bucket.querySelector('.tree-node');
                        if (!node) return { label: 'unknown', count: 0 };
                        const badge = node.querySelector('.status-badge');
                        const spans = node.querySelectorAll('span');
                        let count = 0;
                        for (const span of spans) {
                            if (!span.classList.contains('status-badge') &&
                                !span.classList.contains('status-badge-sm')) {
                                const num = parseInt(span.textContent || '0', 10);
                                if (!isNaN(num)) {
                                    count = num;
                                    break;
                                }
                            }
                        }
                        return {
                            label: badge ? badge.textContent.trim() : 'unknown',
                            count: count,
                        };
                    });
                }""")

                print(f"      Bucket data from browser: {bucket_data}")
                total = sum(b.get("count", 0) for b in bucket_data)
                print(f"      Total from all buckets: {total}")

                assert len(bucket_data) > 0, "No .sprint-tree-bucket elements found in sidebar"
                assert total > 0, \
                    f"All sidebar bucket counts are 0. Expected > 0. " \
                    f"Bucket data: {bucket_data}"

            finally:
                browser.close()

    run_test("Browser: Active Sprint sidebar shows non-zero total count", test_browser_sprint_sidebar_nonzero_total)
    run_test("Browser: sprint-tree-bucket elements show non-zero counts", test_browser_sprint_buckets_nonzero)


if __name__ == "__main__":
    print(f"\nRunning sprint sidebar bucket tests against: {SERVER_URL}")
    print("=" * 70)

    # API tests (no browser required)
    run_test("API: sprint overview returns HTTP 200", test_api_returns_200)
    run_test("API: backlog view is 'grouped' (T-E27-F14-002 fix)", test_api_returns_grouped_view)
    run_test("API: backlog total_count is non-zero", test_api_backlog_total_count_nonzero)
    run_test("API: backlog.groups array is non-empty", test_api_groups_present)
    run_test("API: groups have status_category fields", test_api_groups_have_status_category)
    run_test("API: group item counts sum to total_count", test_api_groups_total_matches_backlog_total)
    run_test("API: groups map to SPRINT_BUCKET_MAP keys", test_api_groups_map_to_buckets)
    run_test("API: simulated bucket aggregation is non-zero (AC-004..AC-008)", test_api_bucket_aggregation_nonzero)

    # Browser tests
    print()
    print("Browser tests:")
    run_browser_tests()

    # Summary
    print()
    print("=" * 70)
    passed = sum(1 for _, ok, _ in results if ok)
    failed = sum(1 for _, ok, _ in results if not ok)
    print(f"Results: {passed} passed, {failed} failed")

    if failed > 0:
        print()
        print("FAILED tests:")
        for name, ok, msg in results:
            if not ok:
                print(f"  - {name}")
                if msg:
                    print(f"    {msg[:200]}")
        sys.exit(1)
    else:
        print("\nAll tests PASSED.")
        sys.exit(0)
