/**
 * Root layout load function — unit tests
 *
 * Background: kanban task `t_6cc7659d` / parent `t_793ea842`.
 *
 * The previous version of `+layout.ts` read `env.PUBLIC_VITE_FF_ENABLE_APP_SHELL`
 * (via `$env/dynamic/public`), while `.env` set `VITE_FF_ENABLE_APP_SHELL=true`
 * (no `PUBLIC_` prefix). The mismatch was silent — no log, no error, no test —
 * which kept `data.ffEnableAppShell` permanently `false` and hid the dashboard
 * branch (including the F1 innerHTML sink).
 *
 * The fix switches the layout to read `import.meta.env.VITE_FF_ENABLE_APP_SHELL`
 * directly (matching the rest of the codebase). These tests pin the corrected
 * env-var name so the mismatch cannot regress without breaking the suite.
 *
 * Coverage:
 *  - `VITE_FF_ENABLE_APP_SHELL=true`  → `ffEnableAppShell === true`
 *  - `VITE_FF_ENABLE_APP_SHELL=false` → `ffEnableAppShell === false`
 *  - `VITE_FF_ENABLE_APP_SHELL` unset → `ffEnableAppShell === false` (defensive default)
 *  - Legacy wrong name `PUBLIC_VITE_FF_ENABLE_APP_SHELL=true` is NOT picked up
 *    (regression guard — proves the original bug cannot return)
 *  - Load returns only the documented `ffEnableAppShell` key
 */
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';

// Why we explicitly stub every test (including the "unset" case):
//   `import.meta.env` is hydrated from `.env` at vite init, so the real
//   `VITE_FF_ENABLE_APP_SHELL` value is `'true'` (set in `.env`). `vi.unstubAllEnvs`
//   restores the stubbed value back to the process-env baseline, which is
//   still `'true'`. To test the "unset" branch, every test must explicitly
//   stub the env var to the value under test.
const stubAppShellFlag = (value: string | undefined) => {
  vi.stubEnv('VITE_FF_ENABLE_APP_SHELL', value);
  vi.stubEnv('PUBLIC_VITE_FF_ENABLE_APP_SHELL', value);
};

beforeEach(() => {
  vi.unstubAllEnvs();
});

afterEach(() => {
  vi.unstubAllEnvs();
});

// Import after `vi.stubEnv` setup is in place so the layout module sees the
// current `import.meta.env` snapshot.
import { load } from './+layout';

// The shape this layout's `load` returns at runtime. Kept local because the
// generated SvelteKit `LayoutData` union is wider than the values this test
// asserts against, and we want the test to be self-documenting.
type LayoutLoadResult = { ffEnableAppShell: boolean };

// The layout's `load` does not consume its `LoadEvent` argument, so a minimal
// `{}` is sufficient. The return type is `void | Partial<LayoutData>` in the
// generated types (the framework lets a load return nothing); at runtime this
// load always returns the documented object, so we narrow here.
const callLoad = async (): Promise<LayoutLoadResult> => {
  const data = (await load({} as Parameters<typeof load>[0])) as LayoutLoadResult;
  return data;
};

describe('+layout load — env-var name', () => {
  it('returns ffEnableAppShell=true when VITE_FF_ENABLE_APP_SHELL="true"', async () => {
    stubAppShellFlag('true');
    const data = await callLoad();
    expect(data.ffEnableAppShell).toBe(true);
  });

  it('returns ffEnableAppShell=false when VITE_FF_ENABLE_APP_SHELL="false"', async () => {
    stubAppShellFlag('false');
    const data = await callLoad();
    expect(data.ffEnableAppShell).toBe(false);
  });

  it('returns ffEnableAppShell=false when VITE_FF_ENABLE_APP_SHELL is unset', async () => {
    stubAppShellFlag(undefined);
    const data = await callLoad();
    expect(data.ffEnableAppShell).toBe(false);
  });

  it('returns ffEnableAppShell=false for any non-"true" value (defensive)', async () => {
    stubAppShellFlag('TRUE'); // case-sensitive
    expect((await callLoad()).ffEnableAppShell).toBe(false);

    stubAppShellFlag('1'); // numeric truthy string
    expect((await callLoad()).ffEnableAppShell).toBe(false);

    stubAppShellFlag(''); // empty
    expect((await callLoad()).ffEnableAppShell).toBe(false);
  });

  // ── Regression guard for the original bug (t_6cc7659d / t_793ea842) ──
  it('ignores the legacy wrong name PUBLIC_VITE_FF_ENABLE_APP_SHELL (regression guard)', async () => {
    // Set the OLD (wrong) name to "true" — the layout must still report false,
    // proving the PUBLIC_-prefixed property is no longer being read.
    vi.stubEnv('PUBLIC_VITE_FF_ENABLE_APP_SHELL', 'true');
    // Leave the correct name unset
    vi.stubEnv('VITE_FF_ENABLE_APP_SHELL', undefined);
    const data = await callLoad();
    expect(data.ffEnableAppShell).toBe(false);
  });

  it('correct name wins when both names are set', async () => {
    vi.stubEnv('VITE_FF_ENABLE_APP_SHELL', 'true');
    vi.stubEnv('PUBLIC_VITE_FF_ENABLE_APP_SHELL', 'false');
    const data = await callLoad();
    expect(data.ffEnableAppShell).toBe(true);
  });
});

describe('+layout load — output shape', () => {
  it('returns an object with only the documented ffEnableAppShell key', async () => {
    stubAppShellFlag('true');
    const data = await callLoad();
    expect(Object.keys(data).sort()).toEqual(['ffEnableAppShell']);
  });
});
