/** Maps a route pattern to the component that handles it. */
export interface RouteConfig {
  component: string;
  title: string;
}

/** Result of matching a URL path against the route table. */
export interface RouteMatch {
  config: RouteConfig;
  params: Record<string, string>;
  query: Record<string, string>;
}
