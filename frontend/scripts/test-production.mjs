import { chromium } from '@playwright/test';
import { readFile } from 'node:fs/promises';
import { createServer } from 'node:http';
import { dirname, extname, resolve, sep } from 'node:path';
import { fileURLToPath } from 'node:url';

const root = resolve(dirname(fileURLToPath(import.meta.url)), '..', 'dist');
/** @type {Record<string, string>} */
const mime = {
	'.css': 'text/css',
	'.html': 'text/html',
	'.js': 'text/javascript',
	'.json': 'application/json',
	'.png': 'image/png',
	'.svg': 'image/svg+xml'
};

const server = createServer(async (request, response) => {
	const pathname = new URL(request.url ?? '/', 'http://localhost').pathname;
	const file = resolve(root, decodeURIComponent(pathname).replace(/^\/+/, '') || 'index.html');
	if (!file.startsWith(root + sep)) {
		response.writeHead(403).end();
		return;
	}
	try {
		const contents = await readFile(file);
		response.writeHead(200, { 'Content-Type': mime[extname(file)] || 'application/octet-stream' });
		response.end(contents);
	} catch {
		response.writeHead(404).end();
	}
});

await new Promise((resolveServer) => server.listen(0, '127.0.0.1', () => resolveServer(undefined)));
const address = server.address();
if (!address || typeof address === 'string') throw new Error('Unable to start the test server');
const origin = `http://127.0.0.1:${address.port}`;
let browser;
try {
	browser = await chromium.launch({ headless: true });
	for (const [width, height] of [[1440, 1024], [768, 1024]]) {
		const page = await browser.newPage({ viewport: { width, height } });
		/** @type {string[]} */
		const errors = [];
		page.on('pageerror', (error) => errors.push(error.message));
		page.on('console', (message) => {
			if (message.type() === 'error') errors.push(message.text());
		});
		page.on('response', (response) => {
			if (response.url().startsWith(`${origin}/`) && response.status() >= 400) {
				errors.push(`${response.status()} ${response.url()}`);
			}
		});
		await page.routeWebSocket(/\/ws/, (socket) => socket.close());
		await page.route('**/api/**', (route) => route.fulfill({ json: {} }));
		await page.route('https://fonts.googleapis.com/**', (route) => route.fulfill({ contentType: 'text/css', body: '' }));
		await page.goto(`${origin}/`, { waitUntil: 'domcontentloaded' });
		try {
			await page.getByText('Workspace', { exact: true }).first().waitFor({ state: 'visible', timeout: 10000 });
			await page.waitForTimeout(1000);
			if (errors.length) throw new Error(errors.join('\n'));
			console.log(`Production workspace ${width}x${height}: OK`);
		} catch (error) {
			throw new Error(`Production workspace ${width}x${height}: ${errors.join('\n')}`, { cause: error });
		} finally {
			await page.close();
		}
	}
} finally {
	if (browser) await browser.close();
	server.close();
}
