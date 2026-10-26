import adapter from '@sveltejs/adapter-static';
import { sveltekit } from '@sveltejs/kit/vite';
import { vitePreprocess } from '@sveltejs/vite-plugin-svelte';
import tailwindcss from '@tailwindcss/vite';
import { defineConfig } from 'vite';

export default defineConfig({
	plugins: [tailwindcss(), sveltekit({
		preprocess: vitePreprocess(),
		adapter: adapter({ pages: 'dist', assets: 'dist', fallback: undefined, precompress: false, strict: true }),
		alias: { $lib: 'src/lib', $api: 'src/api', $components: 'src/components' }
	})],
	build: {
		rolldownOptions: {
			output: {
				codeSplitting: {
					groups: [
						{ name: 'vendor-three', test: /node_modules[\\/]three[\\/]/ },
						{ name: 'vendor-zrender', test: /node_modules[\\/]zrender[\\/]/ },
						{ name: 'vendor-echarts', test: /node_modules[\\/]echarts[\\/]/ }
					]
				}
			}
		}
	},
	server: {
		proxy: {
			'/api': { target: 'http://localhost:35367', changeOrigin: true },
			'/ws': { target: 'ws://localhost:35367', ws: true }
		}
	}
});
