const { test, expect } = require('@playwright/test');
const path = require('path');

test.describe('LIFF Registration Form Visual Test', () => {
  test('should render registration form correctly', async ({ page }) => {
    // ローカルHTMLファイルを開く
    const htmlPath = path.resolve(__dirname, '../../static/liff/register.html');
    await page.goto(`file://${htmlPath}`);

    // ページタイトル確認
    await expect(page).toHaveTitle('Cupid - ユーザー登録');

    // フォーム要素の存在確認
    await expect(page.locator('h1')).toHaveText('💘 Cupid');
    await expect(page.locator('.subtitle')).toHaveText('ユーザー登録');
    await expect(page.locator('input#name')).toBeVisible();
    await expect(page.locator('input#birthday')).toBeVisible();
    await expect(page.locator('button[type="submit"]')).toBeVisible();

    // スクリーンショット（全画面）
    await page.screenshot({
      path: 'tests/visual/screenshots/register-form.png',
      fullPage: true
    });
  });

  test('should render on mobile viewport', async ({ page }) => {
    // モバイルビューポート設定
    await page.setViewportSize({ width: 375, height: 667 });

    const htmlPath = path.resolve(__dirname, '../../static/liff/register.html');
    await page.goto(`file://${htmlPath}`);

    // スクリーンショット（モバイル）
    await page.screenshot({
      path: 'tests/visual/screenshots/register-form-mobile.png',
      fullPage: true
    });
  });

  test('should show form validation', async ({ page }) => {
    const htmlPath = path.resolve(__dirname, '../../static/liff/register.html');
    await page.goto(`file://${htmlPath}`);

    // フォームに入力
    await page.fill('input#name', '山田太郎');
    await page.fill('input#birthday', '2000-01-15');

    // スクリーンショット（入力済み）
    await page.screenshot({
      path: 'tests/visual/screenshots/register-form-filled.png',
      fullPage: true
    });
  });
});
