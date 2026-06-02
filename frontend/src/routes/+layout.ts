import type { LayoutLoad } from './$types';

// Browser-side feature flag for the app shell.
//
// Env-var name convention (matches `specs/feature-flags.md` and the
// `FF_ENABLE_*` / `VITE_FF_ENABLE_*` dual-namespace rule in AGENTS.md):
//   - Server env:    `FF_ENABLE_APP_SHELL`         (read in hooks.server.ts via $env/dynamic/private)
//   - Browser env:   `VITE_FF_ENABLE_APP_SHELL`   (read here via import.meta.env)
//
// Vite exposes every `VITE_*` variable to the browser automatically, so the
// `PUBLIC_` prefix is NOT needed for client-side env access. We use
// `import.meta.env` directly to match the rest of the codebase (e.g.
// `+page.svelte`, `ManualMeetingImportForm.svelte`) which all read VITE_*
// browser flags this way; the SvelteKit `$env/dynamic/public` wrapper has
// an overly restrictive type that requires a `PUBLIC_` prefix, which is
// not how Vite itself exposes variables.
//
// Regression note: a previous version of this file read
// `env.PUBLIC_VITE_FF_ENABLE_APP_SHELL` (via the `$env/dynamic/public`
// wrapper), which silently never matched the `.env` value
// (`VITE_FF_ENABLE_APP_SHELL=true`) and kept `data.ffEnableAppShell`
// permanently `false` — which in turn kept the dashboard branch (and its
// innerHTML sink) from rendering. See kanban task `t_6cc7659d` / parent
// `t_793ea842`.
export const ssr = false;

export const load: LayoutLoad = () => {
  return {
    ffEnableAppShell: import.meta.env.VITE_FF_ENABLE_APP_SHELL === 'true',
  };
};
