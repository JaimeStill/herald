import { litCSSModulePlugin } from '../plugins/css-modules';

const result = await Bun.build({
  entrypoints: ['client/app.ts'],
  outdir: 'dist',
  naming: 'app.[ext]',
  plugins: [litCSSModulePlugin],
  minify: false,
});

if (!result.success) {
  console.error('Build failed:');
  for (const log of result.logs) {
    console.error(log);
  }
  process.exit(1);
}

console.log('Build complete: dist/app.js, dist/app.css');
