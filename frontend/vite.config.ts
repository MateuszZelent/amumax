import { sveltekit } from '@sveltejs/kit/vite';
import { defineConfig } from 'vite';

export default defineConfig({
	plugins: [sveltekit()],
	build: {
		rollupOptions: {
			output: {
				manualChunks(id) {
					if (!id.includes('node_modules')) {
						return;
					}

					if (id.includes('/three/') || id.includes('/three/examples/')) {
						return 'vendor-three';
					}

					if (id.includes('/zrender/')) {
						return 'vendor-zrender';
					}

					if (id.includes('/echarts/')) {
						return 'vendor-echarts';
					}

					if (id.includes('/prismjs/')) {
						return 'vendor-prism';
					}

					if (id.includes('/@sveltejs/') || id.includes('/svelte/')) {
						return 'vendor-svelte';
					}
				}
			}
		}
	},
	server: {
		proxy: {
			'/api': {
				target: 'http://localhost:35367',
				changeOrigin: true
			},
			'/ws': {
				target: 'ws://localhost:35367',
				ws: true
			},
		}
	}
});
