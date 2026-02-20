import { defineConfig } from 'vite';
import { merge } from './vite.client';
import scalarConfig from './scalar/client.config.ts';

export default defineConfig(merge([
  scalarConfig
]))
