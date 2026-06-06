// @ts-check
/**
 * Playwright E2E test: docs/files opened outside shark entities can enter edit mode.
 */

const { test, expect } = require('@playwright/test');

const SERVER_URL = process.env.VIEWER_URL || 'http://127.0.0.1:7777';
const TIMEOUT = 20000;
const DOC_PATH = 'docs/plan/E30-planning-mode-verify-epic/epic.md';

test.describe('Viewer doc file edit button', () => {
  test('file opened through doc view enables Edit and shows textarea', async ({ page }) => {
    await page.goto(SERVER_URL, { waitUntil: 'domcontentloaded', timeout: TIMEOUT });
    await page.waitForFunction(
      () => typeof window.openDocumentByPath === 'function' && typeof window.apiGetFile === 'function',
      { timeout: TIMEOUT }
    );

    await page.evaluate((path) => {
      window.openDocumentByPath(path);
    }, DOC_PATH);

    await page.waitForSelector('#doc-content-pane .markdown-body', { timeout: TIMEOUT });
    await expect(page.locator('#ev-edit-btn')).toBeEnabled();

    await page.locator('#ev-edit-btn').click();
    await expect(page.locator('#edit-textarea')).toBeVisible();
    await expect(page.locator('#edit-textarea')).toContainText('E30');
  });
});
