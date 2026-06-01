// /upcoming — list view server-side data loading
// Ref: evals/e2e/brd-05-manual-upcoming-meeting-creation.md
// Security eval reference: evals/security/brd-05-manual-upcoming-meeting-creation.md
//
// Data source: the in-memory SvelteKit endpoint at src/routes/api/upcoming/+server.ts
// owns the demo data store (globalThis.__upcomingStore). We read the store directly
// here instead of going through fetch('/api/upcoming?...').
//
// Why not fetch: in dev mode, SvelteKit server-side fetch from a load function
// gets routed to the Vite proxy in vite.config.ts (which forwards /api to the
// Go backend at host.docker.internal:3000), NOT to the in-app +server.ts handler.
// The Go backend has no /api/upcoming route (only /api/v1/upcoming with a missing
// repository middleware), so the proxy path returns 4xx/5xx and the page renders
// the error branch. Calling the in-memory store directly avoids the network hop
// and matches the in-process nature of the demo data layer. For production, this
// would be replaced with a real repository call (e.g. lib/server/upcoming.ts).

declare global {
  // eslint-disable-next-line no-var
  var __upcomingStore: Map<string, {
    id: string;
    title: string;
    scheduledStart: string;
    description: string;
    clientOrOrganization: string;
    status: 'scheduled' | 'cancelled';
    createdBy: string;
    createdAt: string;
    updatedAt: string;
    participants: Array<{
      id: string;
      displayName: string;
      email?: string;
      organization?: string;
      createdAt: string;
      updatedAt: string;
    }>;
  }>;
}

function getStore(): Map<string, any> {
  if (!globalThis.__upcomingStore) {
    globalThis.__upcomingStore = new Map();
  }
  return globalThis.__upcomingStore;
}

const DEMO_ID = '11111111-1111-1111-1111-111111111111';

function ensureDemoSeed(store: Map<string, any>) {
  if (store.size > 0) return;
  const demoScheduledStart = new Date(Date.now() + 2 * 24 * 60 * 60 * 1000).toISOString();
  store.set(DEMO_ID, {
    id: DEMO_ID,
    title: 'Q3 Planning Review',
    scheduledStart: demoScheduledStart,
    description: 'Review Q3 roadmap priorities and confirm resource allocation plan',
    clientOrOrganization: 'Acme Corp',
    status: 'scheduled',
    createdBy: 'user-1',
    createdAt: new Date().toISOString(),
    updatedAt: new Date().toISOString(),
    participants: [
      { id: 'p-1', displayName: 'Alice Smith', email: 'alice@example.com', organization: 'Acme Corp', createdAt: new Date().toISOString(), updatedAt: new Date().toISOString() },
      { id: 'p-2', displayName: 'Bob Jones', email: 'bob@example.com', organization: 'Beta LLC', createdAt: new Date().toISOString(), updatedAt: new Date().toISOString() },
    ],
  });
}

export const load = async ({ url }: { url: URL }) => {
  const flag = import.meta.env.VITE_FF_ENABLE_UPCOMING_MEETINGS === 'true';
  if (!flag) {
    return { meetings: [], total: 0, loading: false, error: null };
  }

  try {
    const store = getStore();
    ensureDemoSeed(store);

    const status = url.searchParams.get('status') ?? 'scheduled';
    const meetings = Array.from(store.values())
      .filter((m: any) => (status === 'all' ? true : m.status === status))
      .sort((a: any, b: any) => new Date(a.scheduledStart).getTime() - new Date(b.scheduledStart).getTime());

    return {
      meetings,
      total: meetings.length,
      loading: false,
      error: null,
    };
  } catch (err) {
    return {
      meetings: [],
      total: 0,
      loading: false,
      error: 'Failed to load upcoming meetings.',
    };
  }
};
