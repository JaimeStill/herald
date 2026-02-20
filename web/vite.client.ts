import { resolve } from 'path'
import type { PreRenderedAsset, PreRenderedChunk, RollupOptions } from 'rollup'
import type { UserConfig } from 'vite'

export interface ClientConfig {
  name: string
  input?: string
  output?: {
    entryFileNames?: string | ((chunk: PreRenderedChunk) => string)
    assetFileNames?: string | ((asset: PreRenderedAsset) => string)
  }
  aliases?: Record<string, string>
}

const root = __dirname

export function merge(clients: ClientConfig[]): UserConfig {
  return {
    build: {
      outDir: '.',
      emptyOutDir: false,
      rollupOptions: mergeRollup(clients),
    },
    resolve: mergeResolve(clients),
  }
}

function defaultInput(name: string) {
  return resolve(root, `${name}/client/app.ts`)
}

function defaultEntry(name: string) {
  return `${name}/dist/app.js`
}

function defaultAssets(name: string) {
  return `${name}/dist/[name][extname]`
}

function mergeRollup(clients: ClientConfig[]): RollupOptions {
  return {
    input: Object.fromEntries(
      clients.map(c => [c.name, c.input ?? defaultInput(c.name)])
    ),
    output: {
      entryFileNames: (chunk: PreRenderedChunk): string => {
        const client = clients.find(c => c.name === chunk.name)
        const custom = client?.output?.entryFileNames
        if (custom) return typeof custom === 'function' ? custom(chunk) : custom
        return defaultEntry(chunk.name)
      },
      assetFileNames: (asset: PreRenderedAsset): string => {
        const originalPath = asset.originalFileNames?.[0] ?? ''
        const client = clients.find(c => originalPath.startsWith(`${c.name}/`))
        if (client?.output?.assetFileNames) {
          const custom = client.output.assetFileNames
          return typeof custom === 'function' ? custom(asset) : custom
        }
        return client ? defaultAssets(client.name) : 'app/dist/[name][extname]'
      },
    },
  }
}

function mergeResolve(clients: ClientConfig[]): UserConfig['resolve'] {
  return {
    alias: Object.assign({}, ...clients.map(c => c.aliases ?? {})),
  }
}
