---
name: add-datasource-setting
description: Checklist for adding, renaming or removing a data source setting (jsonData or secureJsonData) end to end — TypeScript types, config editor, defaults, Go structs, provisioning, README and tests. Use for any change to what the user configures on the RabbitMQ data source.
---

# Add (or change) a data source setting

The settings shape is defined in several places that must stay in sync. Work through every step.

1. **Decide where it lives.** Secrets (passwords, private keys, tokens) → `secureJsonData`; everything else →
   `jsonData`. Pick a camelCase key; nested stream settings go under `streamOptions`.
2. **TypeScript types**: add the field to `RabbitMQDataSourceOptions` / `StreamOptions` / ... in `src/types.ts`
   (secrets in `RabbitMQSecureJsonData`).
3. **UI defaults**: add the default to `DEFAULT_OPTIONS` / `DEFAULT_STREAM_OPTIONS` in `src/components/defaults.ts`.
4. **Config editor** (`src/components/ConfigEditor.tsx`): add the input inside the right `FieldSet`, read the value
   from the defaulted `jsonData`, and write through `updateJsonData` / `updateStreamOptions` (one `onOptionsChange`
   with the complete settings — never several calls, never a mount effect). Secrets use `SecretInput` /
   `SecretTextArea` with `isConfigured={secureJsonFields.x}` plus a reset handler. Give it a `label` (tests and screen
   readers find it by role + name) and a `tooltip`.
5. **Go structs** (`pkg/rabbitmqclient/*.go`): add the field with a matching `json:"..."` tag. Secrets: read them in
   `getDatasourceSettings` (`pkg/plugin/datasource.go`) from `DecryptedSecureJSONData`, never from `JSONData`.
6. **Go defaults**: add the default to `NewRabbitMQStreamOptions()` / `NewStreamOptions()` in
   `pkg/rabbitmqclient/defaults.go` (same value as step 3) so settings saved without the field still work.
7. **Use it** in the client code; remember the units (`requestedHeartbeat` seconds, `maxAge` nanoseconds).
8. **Provisioning**: add it to `provisioning/datasources/datasources.yml`.
9. **README**: add a row to the matching settings table (field label, type, required, default, description) and
   remove anything from "Known limitations" that it resolves.
10. **Tests**:
    - Go: extend `pkg/plugin/datasource_test.go` (default applied when missing, explicit value kept — including
      explicit `0`/`false`).
    - Jest: `src/components/ConfigEditor.test.tsx` (renders the default, a change keeps all other settings).
    - E2E: if it changes runtime behaviour, cover it with `createStreamingEnv({ streamOptions: { ... } })` (see the
      `e2e-test` skill); `tests/helpers/grafana.ts#datasourceJsonData` picks up the new default automatically.
11. **CHANGELOG.md**: add an entry under `## Unreleased`.
12. Run `npm run typecheck && npm run lint && npm run test:ci && go test ./pkg/...` and the E2E suite.
