export interface RouteConfig {
  component: string;
  title: string;
}

export interface RouteMatch {
  config: RouteConfig;
  params: Record<string, string>;
  query: Record<string, string>;
}
