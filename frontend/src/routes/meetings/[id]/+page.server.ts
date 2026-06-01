import type { PageServerLoad } from './$types';

const SERVER_URL = process.env.SERVER_URL ?? 'http://localhost:3000';

export const load: PageServerLoad = async ({ params, cookies, fetch }) => {
  const userId = cookies.get('X-User-ID');
  if (!userId) {
    return { meeting: null };
  }

  try {
    const res = await fetch(`${SERVER_URL}/api/v1/meetings/${params.id}`, {
      headers: { 'X-User-ID': userId },
    });

    if (!res.ok) {
      return { meeting: null };
    }

    const meeting = await res.json();
    return { meeting };
  } catch {
    return { meeting: null };
  }
};