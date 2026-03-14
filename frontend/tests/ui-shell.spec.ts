import { expect, test } from '@playwright/test';

const viewports = [
	{ width: 1600, height: 1200 },
	{ width: 1440, height: 1024 },
	{ width: 1280, height: 960 },
	{ width: 1024, height: 900 },
	{ width: 768, height: 1024 }
];

for (const viewport of viewports) {
	test(`workspace shell ${viewport.width}x${viewport.height}`, async ({ page }) => {
		await page.setViewportSize(viewport);
		await page.goto('/');
		await expect(page.getByText('Workspace')).toBeVisible();
		await expect(page).toHaveScreenshot(`workspace-shell-${viewport.width}.png`, {
			fullPage: true,
			animations: 'disabled'
		});
	});
}
