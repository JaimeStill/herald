import type { RouteConfig } from './types';

/** Route table mapping URL path patterns to view components. */
export const routes: Record<string, RouteConfig> = {
  '': { component: 'hd-documents-view', title: 'Documents' },
  'prompts': { component: 'hd-prompts-view', title: 'Prompts' },
  'review/:documentId': { component: 'hd-review-view', title: 'Review' },
  '*': { component: 'hd-not-found-view', title: 'Not Found' },
};
