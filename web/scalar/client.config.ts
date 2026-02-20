import { resolve } from 'path'
import type { ClientConfig } from '../vite.client';

const config: ClientConfig = {
  name: 'scalar',
  input: resolve(__dirname, 'app.ts'),
  output: {
    entryFileNames: 'scalar/scalar.js',
    assetFileNames: 'scalar/scalar.css',
  },
}

export default config
