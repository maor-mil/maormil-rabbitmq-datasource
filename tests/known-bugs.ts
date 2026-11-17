import { test } from './fixtures';

/**
 * Bugs in the current plugin that the E2E suite reproduces. Tests covering them are marked as expected
 * failures, so the suite stays green today and turns red ("unexpected pass") as soon as a bug is fixed —
 * at which point remove the corresponding `expectKnownBug(...)` call.
 */
export const KNOWN_BUGS = {
  // No open bugs are currently reproduced by the suite. Add entries as `key: 'description'`.
} as const satisfies Record<string, string>;

/** Marks the current test as an expected failure caused by a known bug. */
export function expectKnownBug(description: string): void {
  test.fail(true, `Known bug: ${description}`);
}
