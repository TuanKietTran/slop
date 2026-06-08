import { test, expect } from '@playwright/test';

test.describe('Product listing page', () => {
  test('page loads with correct title', async ({ page }) => {
    await page.goto('/index.html');
    await expect(page).toHaveTitle(/ShopDemo/);
  });

  test('header is visible', async ({ page }) => {
    await page.goto('/index.html');
    await expect(page.locator('header h1')).toHaveText('ShopDemo');
  });

  test('product cards are visible', async ({ page }) => {
    await page.goto('/index.html');
    const cards = page.locator('.product-card');
    await expect(cards).toHaveCount(4);
  });

  test('each product card has a name, price, and Add to Cart button', async ({ page }) => {
    await page.goto('/index.html');
    const cards = page.locator('.product-card');
    const count = await cards.count();
    for (let i = 0; i < count; i++) {
      const card = cards.nth(i);
      await expect(card.locator('h3')).toBeVisible();
      await expect(card.locator('.price')).toBeVisible();
      await expect(card.locator('.add-to-cart')).toBeVisible();
    }
  });

  test('search input filters products', async ({ page }) => {
    await page.goto('/index.html');
    await page.fill('#search', 'Keyboard');
    // Only the keyboard card should be visible.
    await expect(page.locator('.product-card:visible')).toHaveCount(1);
    await expect(page.locator('.product-card:visible h3')).toContainText('Keyboard');
  });

  test('search with no match shows no-results message', async ({ page }) => {
    await page.goto('/index.html');
    await page.fill('#search', 'zzz-no-match');
    await expect(page.locator('#no-results')).toBeVisible();
  });

  test('clearing search restores all products', async ({ page }) => {
    await page.goto('/index.html');
    await page.fill('#search', 'Webcam');
    await expect(page.locator('.product-card:visible')).toHaveCount(1);
    await page.fill('#search', '');
    await expect(page.locator('.product-card:visible')).toHaveCount(4);
  });

  test('clicking View Details navigates to product detail page', async ({ page }) => {
    await page.goto('/index.html');
    await page.locator('.product-card').first().locator('a').click();
    await expect(page).toHaveURL(/product\.html/);
  });

  test('footer is present', async ({ page }) => {
    await page.goto('/index.html');
    await expect(page.locator('footer')).toBeVisible();
  });
});
