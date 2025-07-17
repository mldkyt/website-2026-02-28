import { defineConfig } from 'astro/config';

// https://astro.build/config
export default defineConfig({
    site: "https://lunya.pet/",
    compressHTML: true,
    outDir: "../dist"
});
