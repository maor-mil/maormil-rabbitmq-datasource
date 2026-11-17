import { test } from './fixtures';

/**
 * Bugs in the current plugin that the E2E suite reproduces. Tests covering them are marked as expected
 * failures, so the suite stays green today and turns red ("unexpected pass") as soon as a bug is fixed —
 * at which point remove the corresponding `expectKnownBug(...)` call.
 */
export const KNOWN_BUGS = {
  streamNotRecreated:
    'The stream is never recreated after it is deleted in RabbitMQ (contradicts the README): Reconnect() retries ' +
    'forever because closeConsumer() keeps the already-closed consumer when Close() fails (addressed by PR #4).',
  dashboardCloseDeletesStream:
    'RunStream calls Dispose() when Grafana Live stops the stream, so simply closing the dashboard deletes the stream ' +
    '(and its stored messages), exchanges and bindings when disposal is enabled (the default).',
  exchangeBindingNotRemoved:
    'DisposeBinding calls ExchangeBind instead of ExchangeUnbind for exchange-to-exchange bindings ' +
    '(pkg/rabbitmqclient/binding.go), so those bindings are never removed.',
  typeChangeDropsValues:
    'Framer.AddValue silently drops every value whose JSON type differs from the first value seen for that field ' +
    '(addressed by PR #4).',
  nullCrashesPlugin:
    'Framer.AddNil calls Set(0, nil) on an emptied field: "panic: index out of range" kills the plugin process (all ' +
    'RabbitMQ data sources), and the message is replayed from the first offset after every restart (addressed by PR #4).',
} as const;

/** Marks the current test as an expected failure caused by a known bug. */
export function expectKnownBug(description: string): void {
  test.fail(true, `Known bug: ${description}`);
}
