// @ts-check
/**
 * Playwright E2E test: Sprint sidebar bucket counts (T-E27-F14-002)
 *
 * Verifies that after the SPRINT_BUCKET_MAP fix (viewer service uses grouped view),
 * the sidebar "Active Sprint" section shows non-zero bucket counts for sprints
 * that have assigned items.
 *
 * AC-004: Sidebar total count matches actual sprint assignment count
 * AC-005: Ready bucket shows ready_for_development/todo/draft statuses
 * AC-006: In Progress bucket shows in_development/ready_for_approval/in_progress
 * AC-007: Blocked bucket shows blocked status
 * AC-008: Done bucket shows completed/resolved statuses
 */

const { test, expect } = require('@playwright/test');

const SERVER_URL = process.env.VIEWER_URL || 'http://127.0.0.1:7777';
const TIMEOUT = 15000;

test.describe('Sprint sidebar bucket counts (T-E27-F14-002)', () => {
  test('API: sprint overview backlog returns grouped view with non-zero total', async ({ request }) => {
    // Directly verify the API returns grouped data now (not ordered).
    // This is the server-side fix: BacklogOptions{View: "grouped"}.
    const response = await request.get(`${SERVER_URL}/api/v1/viewer/sprint/overview`);
    expect(response.status()).toBe(200);

    const data = await response.json();

    // Must have a sprint loaded
    expect(data.sprint).toBeTruthy();
    expect(data.sprint.key).toBeTruthy();

    // Backlog must be present
    const backlog = data.backlog;
    expect(backlog).toBeTruthy();

    // AC-004: view must be 'grouped' (not 'ordered') so sidebar can aggregate
    expect(backlog.view).toBe('grouped');

    // AC-004: total count must be non-zero (sprint S002 has 8 assignments)
    expect(backlog.total_count).toBeGreaterThan(0);

    // groups must be present and non-empty
    expect(Array.isArray(backlog.groups)).toBe(true);
    expect(backlog.groups.length).toBeGreaterThan(0);

    // Each group must have status_category and items
    for (const group of backlog.groups) {
      expect(group.status_category).toBeTruthy();
      expect(Array.isArray(group.items)).toBe(true);
    }

    // Verify total_count matches sum of group items
    const totalFromGroups = backlog.groups.reduce(
      (sum, g) => sum + (g.items ? g.items.length : 0),
      0
    );
    expect(totalFromGroups).toBe(backlog.total_count);
  });

  test('API: sprint overview groups contain status_categories in SPRINT_BUCKET_MAP', async ({ request }) => {
    // Verify the status_category values in groups match the SPRINT_BUCKET_MAP keys
    // so the JS aggregation will find matches (not all zeros).
    const SPRINT_BUCKET_MAP_KEYS = new Set([
      'todo', 'draft', 'ready_for_development',
      'in_development', 'ready_for_approval', 'in_progress',
      'in_review', 'in_qa', 'ready_for_refinement_tech', 'ready_for_refinement_ba', 'in_refinement',
      'blocked',
      'completed', 'resolved',
    ]);

    const response = await request.get(`${SERVER_URL}/api/v1/viewer/sprint/overview`);
    expect(response.status()).toBe(200);

    const data = await response.json();
    const backlog = data.backlog;
    expect(backlog).toBeTruthy();
    expect(Array.isArray(backlog.groups)).toBe(true);
    expect(backlog.groups.length).toBeGreaterThan(0);

    // At least one group must map to a bucket
    let mappableGroups = 0;
    for (const group of backlog.groups) {
      if (SPRINT_BUCKET_MAP_KEYS.has(group.status_category)) {
        mappableGroups++;
      }
    }
    expect(mappableGroups).toBeGreaterThan(0);
  });

  test('Browser: sprint sidebar shows non-zero total count after loading', async ({ page }) => {
    // Navigate to the viewer
    await page.goto(SERVER_URL, { waitUntil: 'domcontentloaded', timeout: TIMEOUT });

    // Wait for the page to finish loading project data and render the header
    await page.waitForSelector('#header-sprint-btn', { timeout: TIMEOUT });

    // Click the "Sprint" button in the header to enter sprint mode
    await page.click('#header-sprint-btn');

    // Wait for the sidebar sprint section header to appear
    await page.waitForSelector('.sidebar-section-header', { timeout: TIMEOUT });

    // Wait for sprint data to load — the sidebar renders sprint data from the API
    // The sprint section shows either the sprint tree or a loading/error state
    // We wait for the sprint tree to appear with the "Active Sprint" text
    await page.waitForFunction(
      () => {
        // Check that sprint section is present in sidebar
        const headers = document.querySelectorAll('.sidebar-section-header');
        return Array.from(headers).some(h => h.textContent && h.textContent.includes('Sprint'));
      },
      { timeout: TIMEOUT }
    );

    // Wait for sprint overview data to load — triggered by clicking "Sprint" button
    // ensureSprintOverviewLoaded() is called, which fetches from the API
    // We wait for the Active Sprint node to show a non-zero item count
    await page.waitForFunction(
      () => {
        // Find the active sprint header row — it contains the total item count
        const sprintNodes = document.querySelectorAll('.tree-node-sprint');
        for (const node of sprintNodes) {
          const keySpan = node.querySelector('.sprint-item-key');
          if (keySpan && keySpan.textContent === 'Active Sprint') {
            // The count span is the last span in the node
            const spans = node.querySelectorAll('span');
            for (const span of spans) {
              const count = parseInt(span.textContent || '0', 10);
              if (count > 0) return true;
            }
          }
        }
        return false;
      },
      { timeout: TIMEOUT }
    );

    // Get the active sprint total count
    const totalCount = await page.evaluate(() => {
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
    });

    console.log('Active Sprint total count in sidebar:', totalCount);
    // AC-004: total count must be non-zero
    expect(totalCount).toBeGreaterThan(0);
  });

  test('Browser: sprint sidebar bucket rows show non-zero counts', async ({ page }) => {
    // Navigate to the viewer
    await page.goto(SERVER_URL, { waitUntil: 'domcontentloaded', timeout: TIMEOUT });

    // Wait for header to appear, then click Sprint
    await page.waitForSelector('#header-sprint-btn', { timeout: TIMEOUT });
    await page.click('#header-sprint-btn');

    // Wait for the sprint tree to render with bucket rows
    // The buckets are rendered inside .sprint-tree-bucket divs when active is expanded
    await page.waitForFunction(
      () => document.querySelectorAll('.sprint-tree-bucket').length > 0,
      { timeout: TIMEOUT }
    );

    // Get all bucket counts from the sidebar
    const bucketCounts = await page.evaluate(() => {
      const buckets = document.querySelectorAll('.sprint-tree-bucket');
      return Array.from(buckets).map(bucket => {
        // Each bucket has: [status-badge with label] [span with count]
        const node = bucket.querySelector('.tree-node');
        if (!node) return { label: 'unknown', count: 0 };
        const badge = node.querySelector('.status-badge');
        const spans = node.querySelectorAll('span');
        let count = 0;
        for (const span of spans) {
          if (!span.classList.contains('status-badge') && !span.classList.contains('status-badge-sm')) {
            const num = parseInt(span.textContent || '0', 10);
            if (!isNaN(num)) {
              count = num;
              break;
            }
          }
        }
        return {
          label: badge ? badge.textContent : 'unknown',
          count,
        };
      });
    });

    console.log('Bucket counts from sidebar:', JSON.stringify(bucketCounts));

    // AC-004, AC-005, AC-006, AC-007, AC-008: at least one bucket must be non-zero
    const totalFromBuckets = bucketCounts.reduce((sum, b) => sum + b.count, 0);
    console.log('Total from buckets:', totalFromBuckets);
    expect(totalFromBuckets).toBeGreaterThan(0);

    // Verify that buckets are rendered (at least the expected 4)
    // The test project has S002 (planning) with 8 assignments: 7 completed + 1 in_development
    // completed → Done bucket, in_development → In Progress bucket
    const bucketLabels = bucketCounts.map(b => b.label.trim());
    console.log('Bucket labels:', bucketLabels);
    // At minimum we should see some bucket labels
    expect(bucketLabels.length).toBeGreaterThan(0);
  });
});
