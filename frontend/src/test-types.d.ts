// Global ambient types for test matchers used across the project.
// This file is picked up automatically by svelte-check (via `include: ../src/**/*.ts` in .svelte-kit/tsconfig.json)
// and lets `expect(...)` know about @testing-library/jest-dom matchers like `toBeInTheDocument`, `toBeDisabled`, etc.
//
// The matchers are registered at runtime by src/test-setup.ts (imported by vitest setupFiles).
// This file just makes the types available to the type checker so test files compile.

import '@testing-library/jest-dom';
