import { test } from './fixtures';

/**
 * Bugs in the current plugin that the E2E suite reproduces. Tests covering them are marked as expected
 * failures, so the suite stays green today and turns red ("unexpected pass") as soon as a bug is fixed —
 * at which point remove the corresponding `expectKnownBug(...)` call.
 */
export const KNOWN_BUGS = {
  streamNotRecreated:
    'The stream is never recreated after it is deleted in RabbitMQ (contradicts the README): Dispose() tries to ' +
    'delete the already-deleted stream, gives up before closing the stream environment, so Connect() never runs again.',
  dashboardCloseDeletesStream:
    'RunStream calls Dispose() when Grafana Live stops the stream, so simply closing the dashboard deletes the stream ' +
    '(and its stored messages), exchanges and bindings when disposal is enabled (the default).',
} as const;

/** Marks the current test as an expected failure caused by a known bug. */
export function expectKnownBug(description: string): void {
  test.fail(true, `Known bug: ${description}`);
}
