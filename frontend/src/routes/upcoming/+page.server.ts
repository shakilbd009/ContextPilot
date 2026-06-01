// /upcoming — list view server-side data loading
// Ref: evals/e2e/brd-05-manual-upcoming-meeting-creation.md
// Security eval reference: evals/security/brd-05-manual-upcoming-meeting-creation.md

export const load = async ({ fetch }: { fetch: typeof globalThis.fetch }) => {
  const flag = import.meta.env.VITE_FF_ENABLE_UPCOMING_MEETINGS === 'true';
  if (!flag) {
    return { meetings: [], total: 0, loading: false, error: null };
  }

  try {
    const res = await fetch('/api/upcoming?status=scheduled');
    if (!res.ok) {
      return { meetings: [], total: 0, loading: false, error: 'Failed to load upcoming meetings.' };
    }
    const data = await res.json();
    return {
      meetings: data.meetings ?? [],
      total: data.total ?? 0,
      loading: false,
      error: null,
    };
  } catch {
    return { meetings: [], total: 0, loading: false, error: 'Network error loading upcoming meetings.' };
  }
};