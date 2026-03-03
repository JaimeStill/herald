/** Formats an ISO date string into a locale-aware short date (e.g., "Mar 3, 2026"). */
export function formatDate(iso: string): string {
  return new Date(iso).toLocaleDateString(undefined, {
    year: 'numeric',
    month: 'short',
    day: 'numeric',
  });
}
