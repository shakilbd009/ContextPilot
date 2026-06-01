// +page.server.ts — expose server feature flag for client-side mismatch detection
// Uses $env/dynamic/private so this runs server-side only and has runtime access to env vars.
import { env } from '$env/dynamic/private';
import type { PageServerLoad } from './$types';

export const load: PageServerLoad = async () => {
  return {
    serverPreCallBriefingEnabled: env.FF_ENABLE_PRE_CALL_BRIEFING === 'true',
  };
};