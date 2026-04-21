/** CSS module imports produce a CSSStyleSheet for Lit `static styles`. */
declare module "*.module.css" {
  const sheet: CSSStyleSheet;
  export default sheet;
}

declare module "*.css";
