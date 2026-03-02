import type { BunPlugin } from 'bun';

export const litCSSModulePlugin: BunPlugin = {
  name: 'lit-css-module',
  setup(build) {
    build.onResolve({ filter: /\.module\.css$/ }, (args) => {
      return {
        path: Bun.resolveSync(args.path, args.resolveDir),
        namespace: 'lit-css',
      };
    });

    build.onLoad({ filter: /\.module\.css$/, namespace: 'lit-css' }, async (args) => {
      const css = await Bun.file(args.path).text();
      const escaped = css.replace(/`/g, '\\`').replace(/\$/g, '\\$');

      return {
        contents: `
const sheet = new CSSStyleSheet();
sheet.replaceSync(\`${escaped}\`);
export default sheet;
`,
        loader: 'js',
      };
    });
  },
};
