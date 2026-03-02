import { watch } from 'fs';
import { join } from 'path';

const CLIENT_DIR = join(import.meta.dir, '..', 'client');
const BUILD_SCRIPT = join(import.meta.dir, 'build.ts');

let timeout: ReturnType<typeof setTimeout> | null = null;

async function rebuild() {
  console.log('Rebuilding...');
  const proc = Bun.spawn(['bun', BUILD_SCRIPT], {
    cwd: join(import.meta.dir, '..'),
    stdout: 'inherit',
    stderr: 'inherit',
  });
  await proc.exited;
}

function debounceRebuild() {
  if (timeout) clearTimeout(timeout);
  timeout = setTimeout(rebuild, 150);
}

await rebuild();

console.log('Watching client/ for changes...');
watch(CLIENT_DIR, { recursive: true }, (_, filename) => {
  if (filename && (filename.endsWith('.ts') || filename.endsWith('.css'))) {
    debounceRebuild();
  }
});
