// @ts-check
/**
 * Playwright E2E test: Mermaid diagram pan/zoom controls in the viewer.
 */

const { test, expect } = require('@playwright/test');

const SERVER_URL = process.env.VIEWER_URL || 'http://127.0.0.1:7777';
const TIMEOUT = 20000;
const SAMPLE_MERMAID = [
  '```mermaid',
  'flowchart LR',
  '  A[Draft] --> B[Review]',
  '  B --> C[Done]',
  '  C --> D[Archive]',
  '```',
].join('\n');

async function renderSampleDiagram(page) {
  await page.goto(SERVER_URL, { waitUntil: 'domcontentloaded', timeout: TIMEOUT });
  await page.waitForFunction(
    () => typeof window.renderMarkdownFromString === 'function' &&
      typeof window.mermaid !== 'undefined' &&
      typeof window.svgPanZoom !== 'undefined',
    { timeout: TIMEOUT }
  );
  await page.evaluate((markdown) => {
    const target = document.querySelector('#content');
    window.renderMarkdownFromString(markdown, target);
  }, SAMPLE_MERMAID);
  await page.waitForSelector('.mermaid-viewer svg', { timeout: TIMEOUT });
  await page.waitForFunction(() => {
    const viewer = document.querySelector('.mermaid-viewer');
    return viewer && viewer.dataset.panZoomReady === 'true';
  }, { timeout: TIMEOUT });
}

async function mermaidState(page) {
  return page.evaluate(() => {
    const viewer = document.querySelector('.mermaid-viewer');
    const viewport = viewer && viewer.querySelector('.svg-pan-zoom_viewport');
    return {
      zoom: Number(viewer && viewer.dataset.zoom || '0'),
      panX: Number(viewer && viewer.dataset.panX || '0'),
      panY: Number(viewer && viewer.dataset.panY || '0'),
      transform: viewport ? viewport.getAttribute('transform') || '' : '',
    };
  });
}

test.describe('Viewer Mermaid pan/zoom controls', () => {
  test('toolbar, wheel, drag, maximize, and collapse update the rendered SVG', async ({ page }) => {
    await renderSampleDiagram(page);

    const viewer = page.locator('.mermaid-viewer').first();
    await expect(viewer.locator('.mermaid-viewport')).toBeVisible();

    const initial = await mermaidState(page);
    await viewer.locator('[data-mermaid-action="zoom-in"]').click();
    const zoomedIn = await mermaidState(page);
    expect(zoomedIn.zoom).toBeGreaterThan(initial.zoom);

    await viewer.locator('[data-mermaid-action="zoom-out"]').click();
    const zoomedOut = await mermaidState(page);
    expect(zoomedOut.zoom).toBeLessThan(zoomedIn.zoom);

    await viewer.locator('[data-mermaid-action="reset"]').click();
    const reset = await mermaidState(page);
    expect(reset.zoom).toBeCloseTo(initial.zoom, 1);

    const box = await viewer.locator('.mermaid-viewport').boundingBox();
    expect(box).toBeTruthy();
    await page.mouse.move(box.x + box.width / 2, box.y + box.height / 2);
    await page.mouse.wheel(0, -500);
    await page.waitForTimeout(100);
    const wheelZoomed = await mermaidState(page);
    expect(wheelZoomed.zoom).not.toBeCloseTo(reset.zoom, 2);

    await page.mouse.down();
    await page.mouse.move(box.x + box.width / 2 + 80, box.y + box.height / 2 + 30);
    await page.mouse.up();
    const panned = await mermaidState(page);
    expect(Math.abs(panned.panX - wheelZoomed.panX) + Math.abs(panned.panY - wheelZoomed.panY)).toBeGreaterThan(5);

    await viewer.locator('[data-mermaid-action="maximize"]').click();
    await expect(page.locator('.mermaid-overlay')).toBeVisible();
    await expect(page.locator('.mermaid-overlay .mermaid-viewer svg')).toBeVisible();
    await page.locator('.mermaid-overlay-close').click();
    await expect(page.locator('.mermaid-overlay')).toHaveCount(0);
    await expect(viewer.locator('svg')).toBeVisible();

    await viewer.locator('[data-mermaid-action="collapse"]').click();
    await expect(viewer.locator('.mermaid-viewport')).toBeHidden();
    await expect(viewer.locator('[data-mermaid-action="collapse"]')).toHaveAttribute('aria-expanded', 'false');
    await viewer.locator('[data-mermaid-action="collapse"]').click();
    await expect(viewer.locator('.mermaid-viewport')).toBeVisible();
  });
});
