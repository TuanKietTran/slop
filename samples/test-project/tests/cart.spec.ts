import { test, expect } from '@playwright/test';

test.describe('Cart page', () => {
  test.beforeEach(async ({ page }) => {
    // Clear sessionStorage between tests.
    await page.goto('/index.html');
    await page.evaluate(() => sessionStorage.clear());
  });

  test('cart starts empty', async ({ page }) => {
    await page.goto('/cart.html');
    await expect(page.locator('#cart-empty')).toBeVisible();
  });

  test('Add to Cart from listing updates badge', async ({ page }) => {
    await page.goto('/index.html');
    await page.locator('.add-to-cart').first().click();
    await expect(page.locator('#cart-count')).toHaveText('1');
  });

  test('adding same item increments badge', async ({ page }) => {
    await page.goto('/index.html');
    await page.locator('.add-to-cart').first().click();
    await page.locator('.add-to-cart').first().click();
    await expect(page.locator('#cart-count')).toHaveText('2');
  });

  test('cart page shows added item', async ({ page }) => {
    await page.goto('/index.html');
    await page.locator('.add-to-cart').first().click();
    await page.goto('/cart.html');
    await expect(page.locator('#cart-body tr')).toHaveCount(1);
  });

  test('cart page shows correct total', async ({ page }) => {
    await page.goto('/index.html');
    // Add first item (Wireless Headphones $79.99).
    await page.locator('.add-to-cart').first().click();
    await page.goto('/cart.html');
    await expect(page.locator('#total')).toContainText('79.99');
  });

  test('removing item empties cart', async ({ page }) => {
    await page.goto('/index.html');
    await page.locator('.add-to-cart').first().click();
    await page.goto('/cart.html');
    await page.locator('.remove-btn').first().click();
    await expect(page.locator('#cart-empty')).toBeVisible();
  });

  test('Checkout button clears cart', async ({ page }) => {
    await page.goto('/index.html');
    await page.locator('.add-to-cart').first().click();
    await page.goto('/cart.html');
    // Listen for the checkout-msg to appear before clicking so we don't miss it.
    const checkoutMsgPromise = page.waitForFunction(() => {
      const el = document.getElementById('checkout-msg');
      return el && el.style.display !== 'none';
    }, { timeout: 5000 });
    await page.locator('#checkout-btn').click();
    await checkoutMsgPromise;
    await expect(page.locator('#cart-empty')).toBeVisible();
  });
});
