import { env } from '$env/dynamic/public';
import type { LayoutLoad } from './$types';

export const ssr = false;

export const load: LayoutLoad = () => {
  return {
    ffEnableAppShell: env.PUBLIC_VITE_FF_ENABLE_APP_SHELL === 'true',
  };
};