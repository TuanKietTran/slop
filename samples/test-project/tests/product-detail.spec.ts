import { test, expect } from '@playwright/test';

test.describe('Product detail page', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/product.html?id=1');
    await page.evaluate(() => sessionStorage.clear());
  });

  test('detail page renders with title', async ({ page }) => {
    await expect(page).toHaveTitle(/ShopDemo/);
  });

  test('product name is displayed', async ({ page }) => {
    await expect(page.locator('#product-title')).toHaveText('Wireless Headphones');
  });

  test('product price is displayed', async ({ page }) => {
    await expect(page.locator('#product-price')).toContainText('79.99');
  });

  test('product description is visible', async ({ page }) => {
    await expect(page.locator('#product-description')).toBeVisible();
    await expect(page.locator('#product-description')).not.toBeEmpty();
  });

  test('breadcrumb navigation is visible', async ({ page }) => {
    await expect(page.locator('.breadcrumb')).toBeVisible();
    await expect(page.locator('.breadcrumb a').first()).toContainText('Home');
  });

  test('quantity selector defaults to 1', async ({ page }) => {
    await expect(page.locator('#qty')).toHaveValue('1');
  });

  test('increment button increases quantity', async ({ page }) => {
    await page.locator('#qty-inc').click();
    await expect(page.locator('#qty')).toHaveValue('2');
  });

  test('decrement button decreases quantity (not below 1)', async ({ page }) => {
    await page.locator('#qty-inc').click();
    await page.locator('#qty-dec').click();
    await expect(page.locator('#qty')).toHaveValue('1');
    // Try to go below 1 — should stay at 1.
    await page.locator('#qty-dec').click();
    await expect(page.locator('#qty')).toHaveValue('1');
  });

  test('Add to Cart button is present', async ({ page }) => {
    await expect(page.locator('#add-to-cart')).toBeVisible();
  });

  test('clicking Add to Cart shows confirmation message', async ({ page }) => {
    await page.goto('/product.html?id=2');
    await page.locator('#add-to-cart').click();
    await expect(page.locator('#added-msg')).toBeVisible();
  });

  test('Add to Cart updates cart badge', async ({ page }) => {
    await page.goto('/product.html?id=3');
    await page.locator('#add-to-cart').click();
    await expect(page.locator('#cart-count')).toHaveText('1');
  });

  test('different product id loads different product', async ({ page }) => {
    await page.goto('/product.html?id=2');
    await expect(page.locator('#product-title')).toHaveText('Mechanical Keyboard');
  });
});
