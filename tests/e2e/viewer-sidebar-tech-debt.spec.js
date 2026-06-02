// @ts-check
/**
 * Playwright E2E test: viewer sidebar, dashboard, and tech-debt visibility.
 *
 * Verifies that the dashboard renders tech-debt data, the sidebar header is
 * renamed to "Epics", the Tech Debt section is present and collapsible, and
 * the docs browser entry opens the existing folder-files API for docs/.
 */

const { test, expect } = require('@playwright/test');

const SERVER_URL = process.env.VIEWER_URL || 'http://127.0.0.1:7777';
const TIMEOUT = 15000;

test.describe('Viewer sidebar and tech debt visibility', () => {
  test('API: summary and hierarchy include tech debt', async ({ request }) => {
    const summaryResp = await request.get(`${SERVER_URL}/api/v1/viewer/summary`);
    expect(summaryResp.status()).toBe(200);

    const summary = await summaryResp.json();
    expect(summary.tech_debts).toBeTruthy();
    expect(summary.tech_debts.total).toBeGreaterThan(0);

    const hierarchyResp = await request.get(`${SERVER_URL}/api/v1/viewer/hierarchy`);
    expect(hierarchyResp.status()).toBe(200);

    const hierarchy = await hierarchyResp.json();
    expect(Array.isArray(hierarchy.tech_debts)).toBe(true);
    expect(hierarchy.tech_debts.length).toBeGreaterThan(0);
  });

  test('Browser: dashboard shows Tech Debt and sidebar docs/tech-debt controls work', async ({ page }) => {
    await page.goto(SERVER_URL, { waitUntil: 'domcontentloaded', timeout: TIMEOUT });
    await page.waitForSelector('#header-dashboard-btn', { timeout: TIMEOUT });
    await page.click('#header-dashboard-btn');

    await page.waitForSelector('.dashboard-section-title', { timeout: TIMEOUT });

    await expect(page.locator('[data-sidebar-section="epics"] .sidebar-section-title')).toHaveText('Epics');
    await expect(page.locator('[data-sidebar-section="tech_debt"]')).toBeVisible();
    await expect(page.locator('.chart-title', { hasText: 'Tech Debt' })).toHaveCount(1);
    await expect(page.locator('[data-sidebar-section]').last()).toContainText('Docs');

    const toggleAll = page.locator('#sidebar-toggle-all-btn');
    await expect(toggleAll).toHaveText('−');
    await toggleAll.click();
    await expect(toggleAll).toHaveText('+');
    await expect(page.locator('[data-sidebar-section]')).toHaveCount(7);
    await expect(page.locator('[data-sidebar-section].is-collapsed')).toHaveCount(7);
    await toggleAll.click();
    await expect(toggleAll).toHaveText('−');
    await expect(page.locator('[data-sidebar-section].is-collapsed')).toHaveCount(0);

    const techDebtSection = page.locator('[data-sidebar-section="tech_debt"]');
    await techDebtSection.locator('[data-sidebar-section-toggle="tech_debt"]').click();
    await expect(techDebtSection).toHaveClass(/is-collapsed/);

    await page.reload({ waitUntil: 'domcontentloaded', timeout: TIMEOUT });
    await page.waitForSelector('[data-sidebar-section="tech_debt"]', { timeout: TIMEOUT });
    await expect(page.locator('[data-sidebar-section="tech_debt"]')).toHaveClass(/is-collapsed/);

    await page.locator('[data-sidebar-section="docs"] [data-folder-path="docs"]').click();
    await expect(page.locator('.folder-files-header')).toHaveText(/docs/);
  });
});
