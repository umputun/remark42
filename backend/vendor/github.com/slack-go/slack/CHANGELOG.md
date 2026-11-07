# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.35.0] - 2026-11-05

A big thank you to @jjiang-stripe for the security find and fix!

### Breaking changes (but mostly shouldn't affect anyone)

> [!WARNING]
> **`slackevents.Profile` now includes three huddle fields.** Positional struct
> literals no longer compile. Use keyed literals when constructing a profile:

```go
profile := slackevents.Profile{
  DisplayName: "Alex",
  Team:        "T123ABC456",
}
```

### Security

- `InteractionCallbackParse` no longer falls back to the URL query string for the
  `payload` field. A request's signature only covers its POST body, so the fallback
  let a validly-signed request be replayed with an attacker-controlled payload
  smuggled in via the query string.
  Callers must supply `payload` in an `application/x-www-form-urlencoded` request
  body. Form parsing errors are now returned to the caller.

### Fixed

- Prevent repeated escaping and data races when reusing a `MsgOptionText` option.
- Preserve plus signs and percent signs in slacktest message blocks and attachments.
- Encode context block elements as an array when a block is passed by value.
- Keep slacktest bot identities independent when generating RTM connection information.
- Preserve huddle state, expiration, and call ID in Events API user profiles as
  `slackevents.Profile.HuddleState`, `HuddleStateExpirationTS`, and `HuddleStateCallID`.
  These fields follow Slack's documented
  [`user_huddle_changed` payload](https://docs.slack.dev/reference/events/user_huddle_changed/).
- Fix retry jitter panics for large durations on 32-bit platforms.
- Decode JSONTime timestamps outside the 32-bit integer range on 32-bit platforms.

## [0.34.0] - 2026-11-03

Huge thank you to Michal Mazurek (@akfaew) for submitting a bunch of patches for this
release.

### Breaking changes — read before upgrading

> [!WARNING]
> **v0.34.0 includes breaking API, decoding, serialization, and event-dispatch changes.**
>
> **Your code may fail to compile. Code that still compiles may decode, serialize,
> or route events differently. Malformed payloads can cause an entire Events API
> event to be rejected.**
>
> Review every item below before upgrading, including interfaces, mocks, custom
> event decoders, and stored payloads.

#### 1. Retired user APIs have been removed

- **`SetUserAsActive` and `SetUserAsActiveContext` are gone.** Remove these calls.
  `SetUserPresence` still supports `away` and `auto`, but marking user as active is gone.
- **`GetUsersOptionPresence` is gone.** Remove it from `GetUsers`, `GetUsersContext`,
  and `GetUsersPaginated` calls. User-list requests no longer send the retired
  `presence` parameter. Use `GetUserPresence` or `GetUserPresenceContext` for
  individual presence checks.

#### 2. Team-info method signatures have changed

`GetTeamInfo`, `GetTeamInfoContext`, `GetOtherTeamInfo`, and
`GetOtherTeamInfoContext` now accept `...GetTeamInfoOption`. **Update interfaces,
mocks, and assigned function types to include the variadic parameter.** Existing
direct calls remain valid.

#### 3. Auth-provider blocks have a new concrete type and lose raw JSON

> [!NOTE]
> This is a brand new block. For now experimental as it doesn't yet appear in the slack
> docs but is already part of their official node sdk.

- **`auth_provider` blocks now decode as `*AuthProviderBlock`, replacing
  `*UnknownBlock`.** Update type assertions and type switches.

**An unchanged JSON round trip is no longer guaranteed.** Marshaling drops unknown
fields, adds zero values for absent required fields, and can omit explicit nulls
or empty optional values. Keep the original payload if you need to preserve its JSON.

#### 4. Stricter decoding can reject whole events or manifests

Values that were previously ignored or accepted can now return errors:

- **Input blocks:** malformed input elements fail decoding of the input block and
  its containing `Blocks`. Missing elements and JSON `null` input retain their
  previous behavior.
- **Auth-provider blocks:** malformed known fields, such as `"name":42`, fail
  decoding of the containing `Blocks`.
- **Messages:** malformed `display_as_bot` values, such as `"false"`, `0`, `[]`, or
  `{}`, fail message decoding. Missing and `null` values remain accepted.
  Marshaling now emits explicit `true` and `false` values that were previously
  discarded.

**A malformed message or block can cause `slackevents.ParseEvent` to reject the
entire event, preventing its delivery to your handler.** Check error handling and
any captured payloads you replay.

#### 5. Positional struct literals no longer compile

New fields in `Features`, `TeamInfo`, `Msg`, and `Tab` break positional literals for
these types. **Use keyed literals with field names.** Existing keyed literals
remain valid.

#### 6. Built-in user-connection decoders take precedence

The built-in `user_connection` and `user_connection_status_requested` decoders take
precedence over custom entries in `slack.EventMapping`. **Register custom decoders
for these events in `slackevents.EventsAPIInnerEventMapping`.**

#### 7. Socket Mode dispatch changes for envelopes without inner events

**Empty-string event handlers no longer receive these envelopes.** Register
handlers by the outer event type, for example
`HandleEvents(slackevents.AppRateLimited, handler)` for `app_rate_limited`.

A matching outer-type handler can now suppress the default handler. Check
applications that relied on the default handler receiving these envelopes.

### Added

- Add `GetCanvasContent` and `GetCanvasContentContext`, with optional Markdown or
  HTML output. [Slack API reference](https://docs.slack.dev/reference/methods/canvases.getContent/).

- Add `GetTeamInfoOptionIncludeSlackbotAvailability` and
  `TeamInfo.SlackbotAvailability` for advisory workspace availability.
  [Slack API reference](https://docs.slack.dev/reference/methods/team.info/).

- Add `RequestUserConnectionAuth`, `BulkUpdateUserConnections`, their context
  variants, and per-user bulk-update failures. Add `AuthProviderBlock`, the
  `user_connection` and `user_connection_status_requested` events, and manifest
  `features.user_connections`. The authentication APIs, status-requested event,
  and manifest configuration follow the official Node SDK schemas; their standalone
  reference pages were unavailable when added.
  `AuthProviderBlock` support is experimental: `auth_provider` is not live in Slack
  yet. This implementation uses the official Node SDK as a guide. Its schema and
  behavior may change when Slack releases the block.
  [Node SDK release](https://github.com/slackapi/node-slack-sdk/releases/tag/%40slack/web-api%408.3.0),
  [auth provider schema](https://docs.slack.dev/tools/node-slack-sdk/reference/types/interfaces/AuthProviderBlock/),
  [user connection event](https://docs.slack.dev/reference/events/user_connection/).

- Configure RTM WebSocket handshake headers with `RTMOptionDialHeaders`.
  Headers are copied and reused on reconnect; existing defaults remain unchanged.

- Decode `data.file_id` on channel tabs as `Tab.Data.FileID`, so the tab that
  shows a given canvas can be identified from `conversations.info`. Preserve
  absent `data` and `file_id` fields when marshaling tabs.

### Changed

- Document the undocumented preference endpoints and authentication limits of
  `GetUserPrefs`, `GetUserPrefsContext`, `MuteChat`, and `UnMuteChat`. The mute
  helpers use the legacy `muted_channels` preference, which may differ from the
  current client's notification settings.

### Removed

- **Breaking:** Remove `SetUserAsActive` and `SetUserAsActiveContext`, which called
  the non-functional `users.setActive` endpoint. Remove these calls from clients;
  `SetUserPresence` still supports `away` and `auto`, but neither forces a user active.
  Slack retired this endpoint in 2018; see its
  [retirement notice](https://docs.slack.dev/changelog/2018-01-presence-present-and-future/).

- **Breaking:** Remove `GetUsersOptionPresence` and stop sending the retired
  `presence` parameter on `users.list` requests. Remove that option from `GetUsers`,
  `GetUsersContext`, and `GetUsersPaginated` calls. Use `GetUserPresence` or
  `GetUserPresenceContext` when individual presence information is needed.
  Slack retired these features in 2018; see its
  [retirement notice](https://docs.slack.dev/changelog/2018-01-presence-present-and-future/).

### Fixed

- Remove the requested channel from the legacy muted-channel preference in
  `UnMuteChat`. Skip updates when the channel is already unmuted, without panicking.

- Honor `ListTeamsParameters.Limit` in `ListTeams` and `ListTeamsContext` requests.

- Encode multiple user IDs as a comma-separated list in `MigrationExchange` requests.

- Omit unset `TimestampTo` and `Limit` parameters in `ListRemoteFiles` and
  `ListRemoteFilesContext`, so Slack can apply its defaults.

- Preserve optional `display_as_bot` values in messages, including thread
  broadcasts, as `Msg.DisplayAsBot`. Use a pointer to distinguish false from an
  absent field. The remaining thread-broadcast fields already decode through
  `slackevents.MessageEvent.Message`.
  [Node SDK change](https://github.com/slackapi/node-slack-sdk/commit/e02e7feeb500bfde1f947401a639633e42ce2f91).

- Route Events API envelopes without an inner event, such as `app_rate_limited`,
  to handlers registered for the outer event type in Socket Mode.

- Reject request timestamps so far in the future that the duration difference
  overflows, instead of allowing them to pass the five-minute timestamp check.

- Decode confirmation dialogs, options, and option groups in `BlockObjects`
  using their documented object and array fields instead of silently dropping them.
  [Slack composition objects](https://docs.slack.dev/reference/block-kit/composition-objects/).

- Decode `internal_team_ids` in EKM original connected channel responses instead
  of looking for `internal_team_ids_count`.
  [Slack API reference](https://docs.slack.dev/reference/methods/admin.conversations.ekm.listOriginalConnectedChannelInfo/).

- Honor the `Types` filter in `ListFiles` and `ListFilesContext` requests.
  [Slack API reference](https://docs.slack.dev/reference/methods/files.list/).

- Send the requested Y crop coordinate in user photo uploads.

- Preserve the initial pagination state after failed conversations and stars
  requests, so a rate-limited first page is retried instead of reporting an
  empty successful result.

- Return Slack API errors from starred-item pagination rather than silently
  treating them as the end of the list.

- Synchronize RTM connection-info reads with reconnect updates.

- Reconnect RTM connections on permanent WebSocket read errors instead of
  retrying until Gorilla WebSocket panics on a truncated stream. Continue
  tolerating empty or incomplete JSON messages on healthy connections.

- Deliver warnings from user-profile and event-authorization requests.

- Avoid a nil-pointer panic when decoding a `null` conversation-kick response.

- Honor the configured API endpoint in `SetUserCustomFields` and
  `SetUserCustomFieldsContext`.

- Preserve error causes in legacy admin helpers and Events API parsing, and
  recognize wrapped rate-limit, pagination, and RTM errors.

- Accept valid non-ASCII text in `TextBlockObject.Validate` by counting Unicode
  code points for the 3000-character limit.

- Return custom option errors from `FunctionCompleteSuccess` and
  `FunctionCompleteSuccessContext` before sending a request.
- Return decoding errors for malformed input elements from
  `InputBlock.UnmarshalJSON` and `Blocks.UnmarshalJSON`. Missing elements and
  JSON `null` input retain their previous decoding behavior.

## [0.33.0] - 2026-10-30

### Added

- Arrange Block Kit content in rows and columns with `LayoutBlock`, including
  item widths and alignment, grid borders, fallback content for older clients,
  and decoding of nested child blocks. Run `go run ./examples/layout` to preview
  an example or add `-channel C_CHANNEL` to post it.
- Create, inspect, update, list, enable, disable, and delete Slackbot skills with
  all seven `slackbot.skills.*` methods. Optional fields use variadic options,
  including explicit empty values for clearing metadata and MCP declarations.
  The new methods use the existing form transport with JSON-encoded array
  fields. Existing endpoints' request formatting is unaffected.
- Manage Enterprise skill inventory, catalog publication, and deployment with
  all four `admin.slackbot.skills.*` methods. Batch failures and audience
  validation details remain available alongside returned errors. All eleven
  methods have `Context` variants.
  Admin methods use the `AdminSlackbotSkillsList`, `AdminSlackbotSkillsPublish`,
  `AdminSlackbotSkillsSetAutoEnabled`, and `AdminSlackbotSkillsUnpublish` names.
- Run individual methods or verify complete groups with
  [`examples/slackbot_skills`](examples/slackbot_skills/README.md) and
  [`examples/admin_slackbot_skills`](examples/admin_slackbot_skills/README.md).
  Group workflows create, verify, and clean up temporary skills.

## [0.32.0] - 2026-10-28

### Breaking changes

- **Update interfaces, mocks, and function values that use the old canvas creation
  signatures.** `CreateCanvas` and `CreateCanvasContext` now accept
  `options ...CreateCanvasOption`. Existing direct calls still compile, but these
  methods no longer satisfy interfaces with the old fixed signatures. Add the
  variadic parameter to interface and mock declarations; adapt stored function
  values or wrap the call in a function with the old signature.
- **Use field names when initializing `Properties`, `SlackResponse`,
  `KickUserFromConversationSlackResponse`, or `SlackErrorResponse`.** New fields
  make positional literals fail to compile. Existing keyed literals still work.

### Added

- **Show code diffs, HTML reports, Block Kit interfaces, and canvases in Slack Code
  channels.** Use `SetAgentConversationView` to publish or update a view,
  `ListAgentConversationViews` to discover attached views, and
  `RemoveAgentConversationView` to remove them. All have `Context` variants.
  Slack documents pull-request views too, but live requests have returned
  `invalid_view_type`; publish a unified diff with `AgentViewTypeDiff` if that
  happens.
- **Give a code channel useful context.** Set context-bar links, action items,
  a summary message, and external resource details with
  `SetAgentConversationProperties` or its `Context` variant. Read session status,
  title, assigned agents, origin message, context-bar ownership, summary, and
  managed views through `Properties.AgentSession` and `Properties.CodeChannel`
  in conversation responses.
- **Read and revise canvases attached to code channels.**
  `GetAgentConversationCanvas` retrieves markdown and comment threads, including
  replies and optional resolved threads. `SetAgentConversationCanvasContent`
  replaces the markdown while preserving section IDs and comment anchors on
  unchanged sections. Both have `Context` variants.
- **Create a canvas directly in a channel**, including in free workspaces that
  require a channel. Pass `CreateCanvasOptionChannelID(channelID)` to
  `CreateCanvas` or `CreateCanvasContext` (#1623).
- Try the new [`examples/slack_code`](examples/slack_code) CLI for channel
  setup, context bars, views, canvases, commands, metadata inspection, and archival.
  It includes sample HTML, Block Kit, markdown, and unified-diff files.

### Fixed

- **See which scopes are missing when an API call fails.** Slack's `needed` and
  `provided` values are now preserved in responses and `SlackErrorResponse`, even
  for methods that return only an error. Use `errors.As` to inspect `Needed` and
  `Provided`; the error text is unchanged.
- **Discover code-channel views even when Slack returns an empty legacy `views`
  array.** `ListAgentConversationViews` also preserves `session_views`, including
  view types, names, and owner IDs.
- **Receive slash commands when Slack omits `is_enterprise_install`.** JSON
  decoding now accepts absent or null values, including observed Slack Code
  context-bar action deliveries over Socket Mode.
- **Send workflow buttons without an action ID.** An empty
  `WorkflowButtonBlockElement.ActionID` is omitted from JSON so the button passes
  Slack's block validation. Populated IDs are preserved.

### Documentation and testing

- Clarify that Slack documents `raw_number` table cells for publishing. If a
  workspace rejects them with `invalid_blocks`, send the number as a
  `TableRawTextCell`. Null cells remain a received-only representation.
- Add regression coverage for code-channel lifecycle events, Block Kit view
  interactions, and agent slash commands through the existing HTTP and Socket
  Mode parsers and handlers.

## [0.31.0] - 2026-10-26

### Breaking changes

- **`ButtonBlockElement` is no longer comparable** because `VisibleToUserIDs` is a slice.
  Value comparisons and map keys using it no longer compile; interface comparisons
  containing button values can panic. Use comparable identifiers as map keys and compare
  button contents explicitly:

  ```go
  // Before
  same := a == b
  // After (import "reflect")
  same := reflect.DeepEqual(a, b)
  ```

- **If you create these structs without naming their fields, update that code.** New fields
  were added to Block Kit elements, `Attachment`, `File`, `RemoteFile` and `Features`.
  Code that supplies values by position now fails to compile. Include the field names,
  or use the existing `New...` helper functions where available:

  ```go
  // Before: values depend on field order.
  element := slack.RichTextSectionBroadcastElement{slack.RTSEBroadcast, "here"}
  // After: each value names its field.
  element := slack.RichTextSectionBroadcastElement{
      Type:  slack.RTSEBroadcast,
      Range: "here",
  }
  ```

- **Newly supported rich-text elements decode into concrete types** instead of
  `*RichTextSectionUnknownElement`. Update type assertions and switches for attachment,
  workflow, Work Object and message mentions; tags; canvas elements; citations; file and
  List record references; and Salesforce data fields.
- **Request signature validation is stricter, and mismatch errors have changed.**
  `NewSecretsVerifier` rejects signatures without the `v0=` prefix or whose decoded length
  is not 32 bytes. Update any test fixtures that relied on the previous permissive parsing.
  `SecretsVerifier.Ensure` returns `ErrInvalidSignature` on a mismatch; replace checks of
  the old error text with `errors.Is(err, slack.ErrInvalidSignature)`.
- **`UninstallApp` and `RotateTokens` now use POST**, including their `Context` variants.
  Update mocks or custom transports that expect GET requests or credentials in the URL
  to accept form-encoded POST bodies.
- **`OptionOnWarning` callbacks no longer receive the `token` parameter** in form or query
  parameter maps. Update callbacks that rely on that field.

### Security

- `SecretsVerifier.Ensure` now returns `ErrInvalidSignature` on a signature mismatch
  without including the computed signature in the error. Debug logs include only short
  signature prefixes. `NewSecretsVerifier` requires the `v0=` prefix and a 32-byte signature.
- `SlashCommand.ValidateToken` uses constant-time comparison for each candidate token.
- `UninstallApp` and `RotateTokens` (including their `Context` variants) send credentials
  in POST bodies instead of request URLs.
- `OptionOnWarning` omits the `token` parameter from form and query parameter maps passed
  to callbacks, without changing the outgoing request.
- Socket Mode debug logs redact known credential fields in incoming and outgoing payloads
  and omit credentials from connection URLs. Logs still include message content and other
  user data.

### Added

- Add `RTMOptionReadLimit` and `socketmode.OptionReadLimit` to limit incoming WebSocket
  message sizes on every connection, including reconnects. The default remains unlimited;
  exceeding a configured limit closes the connection and triggers normal reconnect handling.
  With compression enabled through a custom dialer, the limit counts compressed bytes and
  does not bound the size after decompression.

- Block Kit: Add typed [Salesforce data fields](https://docs.slack.dev/reference/block-kit/block-elements/salesforce-data-field-element/)
  with record IDs, field metadata, label visibility, and styles. These now decode into
  `RichTextSectionSalesforceDataFieldElement` instead of `RichTextSectionUnknownElement`.

  > [!WARNING]
  > `has_withheld_salesforce_field_names` is based on live Slack responses and is absent from the reference and SDK models.

- Block Kit: Add typed [workflow mentions](https://docs.slack.dev/reference/block-kit/block-elements/workflow-mention-element/)
  with workflow and trigger IDs, display text, URLs, message locations, and styles. These now decode
  into `RichTextSectionWorkflowMentionElement` instead of `RichTextSectionUnknownElement`.

- Block Kit: Add typed [Work Object mentions](https://docs.slack.dev/reference/block-kit/block-elements/work-object-mention-element/)
  with entity and app IDs, display text, URLs, icons, preview settings, product names, and styles.
  These now decode into `RichTextSectionWorkObjectMentionElement` instead of `RichTextSectionUnknownElement`.

- Block Kit: Add typed [attachment mentions](https://docs.slack.dev/reference/block-kit/block-elements/attachment-mention-element/)
  with app, entity, icon, message, preview, and product metadata. These now decode into
  `RichTextSectionAttachmentMentionElement` instead of `RichTextSectionUnknownElement`.

- Block Kit: Add typed [rich-text tags](https://docs.slack.dev/reference/block-kit/block-elements/tag-element/)
  with styles and constants for all nine colors. Tags now decode into
  `RichTextSectionTagElement` instead of `RichTextSectionUnknownElement`.

- Block Kit: Add typed [List record references](https://docs.slack.dev/reference/block-kit/block-elements/list-record-element/)
  with record and view IDs, display text, URLs, and styles. These now decode into
  `RichTextSectionListRecordElement` instead of `RichTextSectionUnknownElement`.

- Block Kit: Add typed [canvas user mentions](https://docs.slack.dev/reference/block-kit/block-elements/canvas-user-mention-element/)
  and [canvas message unfurls](https://docs.slack.dev/reference/block-kit/block-elements/canvas-message-unfurl-element/),
  including thread IDs and styles. These now decode into concrete types instead of `RichTextSectionUnknownElement`.

- Block Kit: Add typed [canvas rich-text elements](https://docs.slack.dev/reference/block-kit/block-elements/canvas-element/)
  with labels, section links, title visibility, styles, and skill metadata. These now decode
  into `RichTextSectionCanvasElement` instead of `RichTextSectionUnknownElement`.

- Block Kit: Add `RichTextSectionCitationElement` with typed file, external, web, message,
  and memory details for [rich-text citations](https://docs.slack.dev/reference/block-kit/block-elements/citation-element/).
  Preserve unknown detail variants as raw JSON. Citations now decode into this concrete type
  instead of `RichTextSectionUnknownElement`.

- Block Kit: Add typed [file](https://docs.slack.dev/reference/block-kit/block-elements/file-element/)
  and [message mention](https://docs.slack.dev/reference/block-kit/block-elements/message-mention-element/)
  rich-text elements, including styles and optional metadata. These now decode into concrete
  types instead of `RichTextSectionUnknownElement`.

- Add image alt text, GIF, PDF, 800px and inline thumbnails, audio and transcription
  metadata, preview truncation, remote file identifiers, canvas and List metadata,
  title blocks, and sharing and app metadata to
  [`File`](https://docs.slack.dev/reference/objects/file-object/).
  Preserve sharing and app metadata in `RemoteFile` responses.

- Block Kit: Add missing [element fields](https://docs.slack.dev/reference/block-kit/block-elements/)
  for button accessibility, agent prompts, visibility, conversation selection, rich-text input
  sizing, and rich-text styles and metadata, including shared emoji URLs.

- Add `IsShare`, `IsMsgUnfurl` and `Files` to `Attachment`, so the files of a forwarded
  message are decoded rather than dropped.
- Add `CreateAgentConversation` and `ArchiveAgentConversation` (plus `Context` variants)
  for [`agents.conversations.create`](https://docs.slack.dev/reference/methods/agents.conversations.create/)
  and [`agents.conversations.archive`](https://docs.slack.dev/reference/methods/agents.conversations.archive/).
- Add `SetAgentConversationCommands` (plus `Context` variant) and `Features.CodeChannels`
  for [agent slash commands](https://docs.slack.dev/ai/slack-code/agent-slash-commands/).
- Add all 12 [Slack Lists API methods](https://docs.slack.dev/reference/methods/?family=lists),
  with `Context` variants and scoped options for lists, access, items, and exports.
- Block Kit: Add `DataTableActionCell` for buttons in
  [`data_table`](https://docs.slack.dev/reference/block-kit/blocks/data-table-block/)
  blocks, with an optional `raw_text` or `raw_number` fallback (#1618).
- Add `AdminAppsApprovedList` and `AdminAppsRestrictedList` (plus `Context` variants) for the
  [`admin.apps.approved.list`](https://docs.slack.dev/reference/methods/admin.apps.approved.list)
  and [`admin.apps.restricted.list`](https://docs.slack.dev/reference/methods/admin.apps.restricted.list)
  methods, with typed app, scope and pagination responses (#1614).
- Add `AdminAppsRequestsList` (plus `Context` variant) for the
  [`admin.apps.requests.list`](https://docs.slack.dev/reference/methods/admin.apps.requests.list)
  method, with typed request and pagination responses (#1615).

- `File` now unmarshals the video fields of the
  [file object](https://docs.slack.dev/reference/objects/file-object): the video
  thumbnail (`ThumbVideo`, `ThumbVideoW`, `ThumbVideoH` and `ThumbVideoTimestamp`) and
  `DurationMs`, `HLS`, `HLSEmbed`, `HLSPreview`, `MediaDisplayType`, `MP4`, `MP4Low` and
  `VTT`.

### Changed

- Add a `Deprecated:` doc comment to `SlashCommand.ValidateToken` to reflect
  [Slack's existing deprecation of verification tokens](https://docs.slack.dev/authentication/verifying-requests-from-slack/).
  The helper remains available with the same signature. For HTTP requests, create a verifier
  with `NewSecretsVerifier`, the request headers and signing secret, write the raw request
  body to it, and call `Ensure` before handling the command. See the
  [slash command example](examples/slash/slash.go).
- Clarify that Socket Mode handlers must be registered before dispatch starts, and
  `EventMapping` and `slackevents.EventsAPIInnerEventMapping` must be configured sequentially
  before event processing. Concurrent registration or registry mutation is unsupported.

### Fixed

- RTM: Keep reader errors scoped to their connection and stop the old reader before
  reconnecting, so stale errors cannot close the next connection.
- `slackevents.ParseEvent` returns an error for an `event_callback` with a missing or null
  inner `event` instead of panicking.
- Release multipart upload writers when a request fails or returns before consuming the
  request body.

## [0.30.1] - 2026-10-04

### Security

- `GetFile`, `GetFileContext` and `UploadToURL` now send the token only to https URLs on
  `slack.com`, `slack-gov.com` and their subdomains, and to the host set with `OptionAPIURL`.
  Any other URL returns an error before a request is made. The `url_private` of a remote or
  external file (`File.IsExternal`) points outside Slack, so `GetFile(file.URLPrivate)` sent
  the token to that host
  ([GHSA-3q3v-34v2-g88f](https://github.com/slack-go/slack/security/advisories/GHSA-3q3v-34v2-g88f)).
  A test that points `GetFile` at an `httptest` server must also pass that server to
  `OptionAPIURL`.

## [0.30.0] - 2026-10-04

### Added

- `slackevents`: Add the Agent messaging events
  [`app_context_changed`](https://docs.slack.dev/reference/events/app_context_changed)
  (`AppContextChangedEvent`),
  [`agent_session_stopped`](https://docs.slack.dev/reference/events/agent_session_stopped)
  (`AgentSessionStoppedEvent`) and
  [`agent_session_title_changed`](https://docs.slack.dev/reference/events/agent_session_title_changed)
  (`AgentSessionTitleChangedEvent`). The shared `AppContext` object is also exposed as
  `AppHomeOpenedEvent.Context` and `MessageEvent.AppContext`, which Slack populates once
  the app subscribes to `app_context_changed`. An `AppContextEntity` carries a string
  `Value` for channel, canvas and list entities and a `Message` for `message_context`
  entities, whose value is an object.
- `slackevents`: `EventsAPICallbackEvent` now unmarshals `authorizations`, so the
  installations an event was delivered for can be read without a second request.
- Add `SetAgentSessionStatus` and `RenameAgentSession` (plus `Context` variants) for the
  [`agents.sessions.setStatus`](https://docs.slack.dev/reference/methods/agents.sessions.setStatus)
  and [`agents.sessions.rename`](https://docs.slack.dev/reference/methods/agents.sessions.rename)
  methods, with `AgentSessionStatus*` constants for the accepted statuses.
- `CreateManifest` now returns the new app's `AppId`, `Credentials` and `OAuthAuthorizeUrl`
  from [`apps.manifest.create`](https://docs.slack.dev/reference/methods/apps.manifest.create).
  Slack sends the credentials only in this response (#1587).
- Add `Features.AgentView` and `Features.AssistantView` for the `features.agent_view` and
  `features.assistant_view` [app manifest](https://docs.slack.dev/reference/app-manifest)
  settings, so a typed export and update keeps them (#1588, #1589).
- Add `Features.UnfurlDomains`, `Features.RichPreviews`, `Features.Search`,
  `OAuthConfig.PKCEEnabled`, `OAuthConfig.TokenManagementEnabled`, `Settings.IsMCPEnabled`,
  `Settings.TokenRotationEnabled`, `Settings.IncomingWebhooks`, `Settings.FunctionRuntime`,
  `Settings.SIWSLinks`, `EventSubscriptions.MetadataSubscriptions` and `OutgoingDomains` for the
  matching [app manifest](https://docs.slack.dev/reference/app-manifest) keys, so a typed export
  and update keeps them. `Features.Search` covers the two callback IDs, not `slackbot_metadata`
  (#1588).
- Add `Attachment.HideColor` (`hide_color`), which removes the color bar from a file unfurl
  sent with [`chat.unfurl`](https://docs.slack.dev/reference/methods/chat.unfurl) (#1590).
- Block Kit: Add `TaskCardBlock.HideTitle` (`hide_title`) and `WithHideTitle`, which hide the
  title of a [`task_card`](https://docs.slack.dev/reference/block-kit/blocks/task-card-block)
  block so `details` becomes the top element (#1590).
- Add `CreateUserGroupOptionAdditionalChannels`, `UpdateUserGroupsOptionAdditionalChannels`
  and `UpdateUserGroupsOptionIncludeCount` for the `additional_channels` and `include_count`
  arguments of [`usergroups.create`](https://docs.slack.dev/reference/methods/usergroups.create)
  and [`usergroups.update`](https://docs.slack.dev/reference/methods/usergroups.update)
  (#1598, #1599).
- Add `SetAppIcon` (plus `Context` variant) for the
  [`apps.icon.set`](https://docs.slack.dev/reference/methods/apps.icon.set) method, which sets an
  app's icon from a public URL or an uploaded image (#1596, #1597).
- Add `AdminTeamsList` for the
  [`admin.teams.list`](https://docs.slack.dev/reference/methods/admin.teams.list) method, with
  `AdminTeamsListOptionLimit` and `AdminTeamsListOptionCursor` (#1592, #1593).
- Add `AdminUsersList` for the
  [`admin.users.list`](https://docs.slack.dev/reference/methods/admin.users.list) method, with
  an `AdminUsersListOption<Field>` option for each argument (#1594, #1595).
- `UserGroup` now exposes `AutoProvision`, `ChannelCount`, `EnterpriseSubteamID`,
  `IsEditingRestricted`, `IsIDPGroup`, `IsMembershipLocked`, `IsOrgLevel`, `IsSection`,
  `IsSubteam` and `IsVisible` (#1604, #1605).
- Add `GetUsersOptionIncludeLocale` and `GetUserInfoOptionIncludeLocale` to choose whether
  [`users.list`](https://docs.slack.dev/reference/methods/users.list) and
  [`users.info`](https://docs.slack.dev/reference/methods/users.info) return the user's `locale`.
  Both methods still ask for it by default. `GetUserInfo` and `GetUserInfoContext` take the new
  option through a variadic `options ...GetUserInfoOption`, so an interface or mock that
  declares their old signature needs the new parameter (#1602, #1603).
- Add `CreateManifestRaw`, `UpdateManifestRaw`, `ExportManifestRaw` and `ValidateManifestRaw`
  (plus `Context` variants), which send and return the app manifest as raw JSON, so keys that
  `Manifest` does not model are kept (#1600, #1601).
- Add `CreateManifestOptionTeamID` to create an app in a given workspace with an org-level
  configuration token. `CreateManifest` and `CreateManifestContext` take it through a variadic
  `options ...CreateManifestOption`, so an interface or mock that declares their old signature
  needs the new parameter (#1600, #1601).

### Changed

- The minimum supported Go version is now 1.26. The library supports the two most recent Go
  releases, so the test matrix covers Go 1.26 and Go 1.27.
- `DeleteFileComment` now takes `(fileID, commentID)`, the same order as
  `DeleteFileCommentContext` (#1591). A call that still passes a comment ID (`Fc…`) first
  returns an error and sends nothing to Slack.

  > [!WARNING]
  > **Breaking change.** Swap the arguments of every `DeleteFileComment` call:
  >
  > ```go
  > // Before
  > err := api.DeleteFileComment(commentID, fileID)
  > // After
  > err := api.DeleteFileComment(fileID, commentID)
  > ```

### Fixed

- `socketmode` and RTM reconnects now wait at most 5 minutes between attempts, and
  `RetryConfig.BackoffMax` now caps the Web API retry delay. The backoff ignored its
  maximum and kept doubling, so after a long outage a Socket Mode client could stay asleep
  long after the network recovered and miss the events sent in that window (#1585, #1586).
- `socketmode`: The client now reconnects when the WebSocket stream ends in the middle of a TLS
  record. Before, it read the failed connection again until gorilla/websocket panicked with
  `repeated read on failed websocket connection`, which stopped the process. The client also
  reconnects after 10 empty or malformed frames in a row (#1608).
- `GetFile` now returns an error when the token cannot read the file. Slack then redirects the
  download to the workspace sign-in page, and `GetFile` used to write that HTML page as the
  file (#1284, #1582).

## [0.29.0] - 2026-08-15

### Fixed

- `slackevents`: `AppMentionEvent` and `MessageEvent` now expose the Data Access API action
  token through a new `ActionToken` field, which reads `action_token` from the event object
  itself. Slack sends the token there on `app_mention` and `message` events, but both types
  only modelled it nested inside `assistant_thread`, so the token was silently dropped and
  bot-token calls to `assistant.search.context` failed with `invalid_action_token`. The
  existing `AssistantThread` field is unchanged (#1577, #1580).

## [0.28.0] - 2026-08-15

### Added

- Block Kit: Add support for [`container`](https://docs.slack.dev/reference/block-kit/blocks/container-block/) block through `ContainerBlock`, with a `NewContainerBlock` constructor, fluent `With*` builders (`WithTitle`, `WithRichTextTitle`, `WithSubtitle`, `WithIcon`, `WithWidth`, `WithCollapsible`, `WithHeaderDivider`, `WithBlockID`), an `AddChildBlock` helper and a `Validate` method (#1574).
- `PostMessageWithResponse` and `PostMessageWithResponseContext` return the full `Message` object from the `chat.postMessage` response alongside the channel and timestamp, giving access to response-only fields such as `Message.ThreadTimestamp` (#1572).
- `CompleteUploadExternalParameters` and `UploadFileParameters` now take `Channels`, which shares a single uploaded file with up to 100 conversations in one `files.completeUploadExternal` request (#1579).

## [0.27.0] - 2026-06-27

### Added

- Block Kit: Add support for
  [`data_visualization`](https://docs.slack.dev/reference/block-kit/blocks/data-visualization-block/) block through `DataVisualizationBlock`.
- Interaction payloads: `Team` now preserves `enterprise_id` and `enterprise_name` when
  Slack includes Enterprise org details in interaction payload `team` objects.

## [0.26.0] - 2026-06-14

### Added

- `slackevents`: `EventsAPIEvent` now unmarshals the `is_ext_shared_channel` field, and
  `ParseEvent` propagates it for callback events (#1564 and #1565).
- Block Kit: `HeaderBlock` now supports the optional `level` field (1-4, mapping to H1-H4
  heading levels), with a `HeaderBlockOptionLevel` builder (#1563).
- Block Kit: `CardBlock` now supports the `slack_icon` field via the new `SlackIconObject`
  composition object (with a `NewSlackIconObject` constructor and `WithSlackIcon` builder)
  and the `subtext` field (with a `WithSubtext` builder). `slack_icon` is mutually
  exclusive with `icon` (#1562).

## [0.25.0] - 2026-06-05

### Changed

> [!WARNING]
> **Breaking change.**

- `TableBlock.Rows` is now `[][]TableCell` (was `[][]*RichTextBlock`), so `table`
  blocks no longer drop `raw_text`, `raw_number`, and `null` cells (#1558).

## [0.24.0] - 2026-05-24

### Added

- Block Kit: `DataTableBlock` for the [`data_table`](https://docs.slack.dev/reference/block-kit/blocks/data-table-block/)
  block, with `NewDataTableBlock`, `AddRow`, raw-text/raw-number/rich-text cell
  constructors, and `WithPageSize` / `WithRowHeaderColumnIndex` builders.

### Changed

- `NewTaskCardBlock` and `NewPlanBlock` nil-guard their variadic options,
  matching the other block constructors (#1236).

## [0.23.1] - 2026-05-10

### Fixed

- `NewSecretsVerifier` now rejects empty signing secrets to avoid accepting forged request
  signatures when applications are misconfigured.

## [0.23.0] - 2026-04-22

### Added

- **Block Kit: `CardBlock` and `CarouselBlock`** — Support for two of the new
  agent-UI blocks announced in the
  [April 16 Slack changelog](https://docs.slack.dev/changelog/2026/04/16/block-kit-new-blocks).
  `CardBlock` is constructed via `NewCardBlock` with a functional-options
  pattern and fluent `With*` builders (`WithTitle`, `WithSubtitle`, `WithBody`,
  `WithIcon`, `WithHeroImage`, `WithActions`). `CarouselBlock` is constructed
  via `NewCarouselBlock` with a variadic `*CardBlock` list plus `WithBlockID`
  and `AddCard` helpers. Both blocks wire into `Blocks.UnmarshalJSON` for
  round-trip fidelity, and reuse existing `ImageBlockElement` /
  `ButtonBlockElement` / `BlockElements` types rather than introducing new
  composition objects.
- **Block Kit: `AlertBlock`** — Support for the third of the new agent-UI
  blocks from the
  [April 16 Slack changelog](https://docs.slack.dev/changelog/2026/04/16/block-kit-new-blocks).
  `AlertBlock` is constructed via `NewAlertBlock` with a `*TextBlockObject`
  body and a functional-options pattern. Severity is set via
  `AlertBlockOptionLevel` (`AlertLevelDefault`, `AlertLevelInfo`,
  `AlertLevelWarning`, `AlertLevelError`, `AlertLevelSuccess`) and the block
  ID via `AlertBlockOptionBlockID`. Wires into `Blocks.UnmarshalJSON` for
  round-trip fidelity. Must be delivered via the streaming chunks API —
  `chat.postMessage` rejects it as an unsupported block type.
- **Streaming-message chunks API** — `chat.startStream` / `chat.appendStream` /
  `chat.stopStream` now accept a `chunks` parameter. Added `MsgOptionChunks`
  along with a `StreamChunk` interface and four chunk types:
  `MarkdownTextChunk`, `TaskUpdateChunk`, `PlanUpdateChunk`, and `BlocksChunk`
  (each with a `New*Chunk` constructor). This is the supported transport for
  streaming Block Kit content and the new agent-UI blocks in particular
  (which `chat.postMessage` rejects as `Unsupported block type`).
- **`MsgOptionTaskDisplayMode`** — New option for `chat.startStream` controlling
  whether task chunks render as a sequential timeline or a grouped plan.
  Accepts `TaskDisplayModeTimeline` or `TaskDisplayModePlan`.
- Added `Username`, `IconURL`, and `IconEmoji` fields to
  `AssistantThreadsSetStatusParameters`, forwarded by
  `SetAssistantThreadsStatusContext`, matching the new optional parameters on
  [`assistant.threads.setStatus`](https://docs.slack.dev/reference/methods/assistant.threads.setStatus)
  for customising the status-update presentation.
- Exposed `SocketmodeHandler.DispatchEvent` (previously the unexported
  `dispatcher`), enabling integration tests to exercise registered handlers
  without a live WebSocket connection. The unexported `dispatcher` is kept as
  a thin wrapper for backwards compatibility. Closes #1549.

## [0.22.0] - 2026-04-12

### Added

- Added missing parameters to `assistant.search.context` (`Sort`, `SortDir`, `Before`,
  `After`, `Highlight`, `IncludeContextMessages`, `IncludeDeletedUsers`,
  `IncludeMessageBlocks`, `IncludeArchivedChannels`, `DisableSemanticSearch`, `Modifiers`,
  `TermClauses`) and new response types (`AssistantSearchContextFile`,
  `AssistantSearchContextChannel`, `AssistantSearchContextMessageContext`) to match the
  full Real-Time Search API surface.
- Added `Underline`, `Highlight`, `ClientHighlight`, and `Unlink` fields to
  `RichTextSectionTextStyle`. Added `Style` field to `RichTextSectionUserGroupElement`.
- Added `BotOptional` and `UserOptional` fields to `OAuthScopes` for app manifests.
- Added PKCE support for OAuth: `OAuthOptionCodeVerifier` option for
  `GetOAuthV2Response`, `GenerateCodeVerifier()` and `GenerateCodeChallenge()`
  helper functions (RFC 7636). `client_secret` is now conditionally omitted when
  empty in both `GetOAuthV2ResponseContext` and `RefreshOAuthV2TokenContext`.

### Fixed

- `ChannelTypes` and `ContentTypes` now send comma-separated values instead of repeated
  form keys, matching the convention used by every other method in the library.
- In `socketmode` malformed JSON messages no longer force an unnecessary reconnect.
  Instead the error is emitted and the connection continues as normal.

## [0.21.1] - 2026-04-08

### Added

- **`slackevents.ChannelType*` constants and `MessageEvent` helpers** — Added
  `ChannelTypeChannel`, `ChannelTypeGroup`, `ChannelTypeIM`, `ChannelTypeMPIM` constants
  and `IsChannel()`, `IsGroup()`, `IsIM()`, `IsMpIM()` methods on `MessageEvent` so
  callers no longer need to compare raw strings.

### Fixed

- **Duplicate attachment/block serialization in `MsgOptionAttachments` / `MsgOptionBlocks`** —
  Attachments and blocks were serialized twice: once into typed struct fields (for the JSON
  response-URL path) and again into `url.Values` (for the form POST path). Serialization for
  the form path now happens inside `formSender.BuildRequestContext`, so each sender owns its
  own marshalling. This fixes the long-standing FIXME and eliminates redundant `json.Marshal`
  calls in the option functions. ([#1547])

  > [!NOTE]
  > `UnsafeApplyMsgOptions` returns `config.values` directly. After this change,
  > `attachments` and `blocks` keys are no longer present in those values since
  > marshalling is deferred to send time. This function is documented as unsupported.

## [0.21.0] - 2026-04-05

### Deprecated

- **`slackevents.ParseActionEvent`** — Cannot parse `block_actions` payloads (returns
  unmarshalling error). Use `slack.InteractionCallback` with `json.Unmarshal` instead,
  or `slack.InteractionCallbackParse` for HTTP requests. `InteractionCallback` handles
  all interaction types. ([#596])
- **`slackevents.MessageAction`**, **`MessageActionEntity`**, **`MessageActionResponse`** —
  Associated types that only support legacy `interactive_message` payloads.

### Removed

- **`IM` struct** — Removed the `IM` struct (and unused internal types `imChannel`,
  `imResponseFull`). The `IsUserDeleted` field has been moved to `Conversation`, where it
  is populated for IM-type conversations. Code using `IM` should switch to `Conversation`.

  > [!NOTE]
  > In practice no user should be affected — `IM` was never returned by any public API
  > method in this library, so there was no way to obtain one outside of manual construction.

- **`Info.GetBotByID`, `GetUserByID`, `GetChannelByID`, `GetGroupByID`, `GetIMByID`** —
  These methods were deprecated and returned `nil` unconditionally. They have been removed.

  > [!WARNING]
  > **Breaking change.** If you are calling any of these methods, remove those calls — they
  > were already no-ops.

### Added

- **`admin.teams.settings.*` API support** — `AdminTeamsSettingsInfo`,
  `AdminTeamsSettingsSetDefaultChannels`, `AdminTeamsSettingsSetDescription`,
  `AdminTeamsSettingsSetDiscoverability`, `AdminTeamsSettingsSetIcon`, and
  `AdminTeamsSettingsSetName`. Includes `TeamDiscoverability` enum with `Open`,
  `InviteOnly`, `Closed`, and `Unlisted` variants. ([#960])
- **`OAuthOptionAPIURL` for package-level OAuth functions** — All package-level OAuth
  functions (`GetOAuthV2Response`, `GetOpenIDConnectToken`, `RefreshOAuthV2Token`, etc.)
  now accept variadic `OAuthOption` arguments. Use `OAuthOptionAPIURL(url)` to override
  the default Slack API URL, enabling integration tests against a local HTTP server.
  Existing callers are unaffected. ([#744])
- **`GetOpenIDConnectUserInfo` / `GetOpenIDConnectUserInfoContext`** — Returns identity
  information about the user associated with the token via `openid.connect.userInfo`.
  Complements the existing `GetOpenIDConnectToken` method. ([#967])
- **HTTP response headers** — Slack API response headers (e.g. `X-OAuth-Scopes`,
  `X-Accepted-OAuth-Scopes`, `X-Ratelimit-*`) are now accessible. `AuthTestResponse`
  exposes a `Header` field directly. For all other methods, use
  `OptionOnResponseHeaders(func(method string, headers http.Header))` to register a
  callback that fires after every API call. ([#1076])
- **`DNDOptionTeamID`** — `GetDNDInfo` and `GetDNDTeamInfo` now accept optional
  `ParamOption` arguments. Use `DNDOptionTeamID("T...")` to pass `team_id`, which is
  required after workspace migration (Slack returns `missing_argument` without it).
  ([#1157])
- **`UpdateUserGroupMembersList` / `UpdateUserGroupMembersListContext`** — Convenience
  wrappers around `UpdateUserGroupMembers` that accept `[]string` instead of a
  comma-separated string, enabling clean chaining with `GetUserGroupMembers`. ([#1172])
- **`SetUserProfile` / `SetUserProfileContext`** — Set multiple user profile fields in a
  single API call by passing a `*UserProfile` struct to `users.profile.set`. Complements
  the existing single-field methods (`SetUserRealName`, `SetUserCustomStatus`, etc.).
  ([#1158])
- **API warning callbacks** — Slack API responses may include a `warnings` field with
  deprecation notices or usage hints. Use `OptionWarnings(func(warnings []string))` to
  register a callback that receives these warnings. ([#1540])
- **RTM support for `user_status_changed`, `user_huddle_changed`, `user_profile_changed`
  events** — these events are now mapped in `EventMapping` with dedicated structs
  (`UserStatusChangedEvent`, `UserHuddleChangedEvent`, `UserProfileChangedEvent`).
  Previously they triggered `UnmarshallingErrorEvent`. ([#1541])
- **RTM support for `sh_room_join`, `sh_room_leave`, `sh_room_update`, `channel_updated`
  events** — Slack Call/Huddle room events and channel property updates are now mapped with
  dedicated structs (`SHRoomJoinEvent`, `SHRoomLeaveEvent`, `SHRoomUpdateEvent`,
  `ChannelUpdatedEvent`). ([#858])
- **`CacheTS` and `EventTS` fields on `UserChangeEvent`** — these fields were sent by Slack
  but silently dropped during unmarshalling.
- **`workflows.featured` API support** — add, list, remove, and set featured workflows on
  channels via `WorkflowsFeaturedAdd`, `WorkflowsFeaturedList`, `WorkflowsFeaturedRemove`,
  and `WorkflowsFeaturedSet`
- **`IsConnectorBot` and `IsWorkflowBot` in `User`** — boolean flags for connector and
  workflow bot users
- **`GuestInvitedBy` in `UserProfile`** — user ID of whoever invited a guest user
- **`Blocks` field on `MessageEvent`** — block data from webhook payloads is now directly
  accessible via `event.Blocks` instead of only through `event.Message.Blocks`. ([#1257])
- **`Username` field on `User`** — Slack's interaction payloads (block_actions, shortcuts)
  include a `username` field in the user object that was previously dropped during
  unmarshalling. ([#1218])
- **`Blocks`, `Attachments`, `Files`, `Upload` fields on `AppMentionEvent`** — these fields
  are sent by Slack in `app_mention` event payloads but were silently dropped. ([#961])
- **`HandleShortcut`, `HandleViewSubmission`, `HandleViewClosed` in socketmode handler** —
  Level 3 handlers that dispatch `shortcut`/`message_action`, `view_submission`, and
  `view_closed` interactions by `CallbackID`, matching the pattern of
  `HandleInteractionBlockAction` and `HandleSlashCommand`. ([#1161])
- **`BlockFromJSON` / `MustBlockFromJSON`** — Create blocks from raw JSON strings, enabling
  direct use of output from Slack's [Block Kit Builder](https://app.slack.com/block-kit-builder)
  or quick adoption of new block types before the library adds typed support. The original
  JSON is preserved through marshalling. ([#1497])

### Documentation

- **`ViewSubmissionResponse` constructors** — `NewClearViewSubmissionResponse`,
  `NewUpdateViewSubmissionResponse`, `NewPushViewSubmissionResponse`, and
  `NewErrorsViewSubmissionResponse` now have doc comments explaining the HTTP response
  pattern (write JSON and return promptly) and the Socket Mode pattern (pass as Ack
  payload). `NewErrorsViewSubmissionResponse` additionally documents that map keys must
  be `BlockID`s of `InputBlock` elements. ([#726], [#1013])
- **Socket Mode examples** — `examples/socketmode/` and `examples/socketmode_handler/` now
  have doc comments explaining the two-token requirement: app-level token (`xapp-`) for the
  WebSocket connection and bot token (`xoxb-`) for API calls. ([#941])

### Fixed

- **`UnknownBlock` round-trip data loss** — Unrecognized block types (e.g. new Slack block
  types not yet supported by this library) now preserve their full JSON through
  unmarshal/marshal cycles. Previously only `type` and `block_id` were retained, silently
  discarding all other fields.

### Changed

- Adjusted some `admin` errors that started with uppercase to be lowercase per go
  conventions.

  > [!WARNING]
  > **Breaking change.** If you are matching the error content in your code, this is a
  > BREAKING CHANGE.
- **`WebhookMessage.UnfurlLinks` and `UnfurlMedia` are now `*bool`** — Previously these
  were `bool` with `omitempty`, which meant `false` was silently stripped from the JSON
  payload. Users could not explicitly disable link or media unfurling via webhooks. The
  fields are now `*bool` so that `nil` (omit), `false`, and `true` all serialize correctly.
  ([#1231])

  > [!WARNING]
  > **Breaking change.** Code that sets these fields directly must be updated:
  >
  > ```go
  > // Before
  > msg := slack.WebhookMessage{UnfurlLinks: true}
  >
  > // After — use a helper or a variable
  > t := true
  > msg := slack.WebhookMessage{UnfurlLinks: &t}
  > ```
  >
  > Leaving the fields unset (`nil`) preserves the previous default behavior — Slack's
  > server-side defaults apply (`unfurl_links=false`, `unfurl_media=true`).

- **`User.Has2FA` is now `*bool`** — When using a bot token, Slack's `users.list` API omits
  `has_2fa` entirely. With a plain `bool`, this was indistinguishable from explicitly `false`.
  Now `nil` means absent/unknown, `false` means explicitly disabled, `true` means enabled.
  ([#1121])

  > [!WARNING]
  > **Breaking change.** Code that reads `Has2FA` must handle the pointer:
  >
  > ```go
  > // Before
  > if user.Has2FA { ... }
  >
  > // After
  > if user.Has2FA != nil && *user.Has2FA { ... }
  > ```

- **`ListReactions` now uses cursor-based pagination** — `ListReactionsParameters` replaces
  `Count`/`Page` with `Cursor`/`Limit`, and `ListReactions`/`ListReactionsContext` now return
  `([]ReactedItem, string, error)` where the string is the next cursor, instead of
  `([]ReactedItem, *Paging, error)`. ([#825])

  > [!WARNING]
  > **Breaking change.** Both the parameters and return signature have changed:
  >
  > ```go
  > // Before
  > params := slack.NewListReactionsParameters()
  > params.Count = 100
  > params.Page = 2
  > items, paging, err := api.ListReactions(params)
  >
  > // After
  > params := slack.NewListReactionsParameters()
  > params.Limit = 100
  > items, nextCursor, err := api.ListReactions(params)
  > // Use nextCursor for the next page: params.Cursor = nextCursor
  > ```

- **`ListStars`/`GetStarred` now use cursor-based pagination** — `StarsParameters` replaces
  `Count`/`Page` with `Cursor`/`Limit` (and adds `TeamID`), and `ListStars`/`ListStarsContext`/
  `GetStarred`/`GetStarredContext` now return `string` (next cursor) instead of `*Paging`.
  Slack's `stars.list` API no longer returns `paging` data — only `response_metadata.next_cursor`.

  > [!WARNING]
  > **Breaking change.** Both the parameters and return signature have changed:
  >
  > ```go
  > // Before
  > params := slack.NewStarsParameters()
  > params.Count = 100
  > params.Page = 2
  > items, paging, err := api.ListStars(params)
  >
  > // After
  > params := slack.NewStarsParameters()
  > params.Limit = 100
  > items, nextCursor, err := api.ListStars(params)
  > // Use nextCursor for the next page: params.Cursor = nextCursor
  > ```

- **`GetAccessLogs` now uses cursor-based pagination** — `AccessLogParameters` replaces
  `Count`/`Page` with `Cursor`/`Limit` (and adds `Before`), and `GetAccessLogs`/
  `GetAccessLogsContext` now return `string` (next cursor) instead of `*Paging`.
  Slack's `team.accessLogs` API warns `use_cursor_pagination_instead` when using the old
  parameters.

  > [!WARNING]
  > **Breaking change.** Both the parameters and return signature have changed:
  >
  > ```go
  > // Before
  > params := slack.NewAccessLogParameters()
  > params.Count = 100
  > params.Page = 2
  > logins, paging, err := api.GetAccessLogs(params)
  >
  > // After
  > params := slack.NewAccessLogParameters()
  > params.Limit = 100
  > logins, nextCursor, err := api.GetAccessLogs(params)
  > // Use nextCursor for the next page: params.Cursor = nextCursor
  > ```

### Fixed

- **Socket Mode: large Ack payloads no longer silently fail** — Two issues caused `Ack()`
  payloads to be silently dropped by Slack. First, gorilla/websocket's default 4KB write
  buffer fragmented messages into WebSocket continuation frames that Slack does not
  reassemble. The library now uses a 32KB write buffer. Second, Slack silently drops
  Socket Mode responses at or above 20KB — `Ack()`, `Send()`, and `SendCtx()` now return
  an error when the serialized response reaches this limit. ([#1196])

  > [!WARNING]
  > **Breaking change.** `Ack()` and `Send()` now return `error`. Existing call sites that
  > don't capture the return value continue to compile without changes.

- **`MsgOptionBlocks()` with no arguments now sends `blocks=[]`** — Previously, calling
  `MsgOptionBlocks()` with no arguments or a nil spread was a silent no-op, making it
  impossible to clear blocks from a message via `chat.update`. The Slack API requires an
  explicit `blocks=[]` to reliably remove blocks. ([#1214])

  > [!WARNING]
  > **Breaking change.** `MsgOptionBlocks()` with no arguments now sends `blocks=[]` instead
  > of being a no-op. If you were relying on this to be a no-op, remove the option entirely:
  >
  > ```go
  > // Before — this was a no-op, now it sends blocks=[]
  > api.PostMessage(ch, slack.MsgOptionBlocks(), slack.MsgOptionText("text", false))
  >
  > // After — omit MsgOptionBlocks entirely to not set blocks
  > api.PostMessage(ch, slack.MsgOptionText("text", false))
  > ```

- **`WorkflowButtonBlockElement` missing from `UnmarshalJSON`** — `workflow_button` blocks
  now unmarshal correctly through `BlockElements`, `InputBlock`, and `Accessory` paths.
  Also adds missing `multi_*_select` and `file_input` cases to `BlockElements.UnmarshalJSON`,
  and fixes `toBlockElement` for `RichTextInputElement` and `WorkflowButtonElement`. ([#1539])
- **`NewBlockHeader` nil pointer dereference** — passing a nil text object no longer panics. ([#1236])
- **`ValidateUniqueBlockID` rejects empty block IDs** — multiple input blocks with no
  explicit `block_id` set (empty string) were incorrectly flagged as duplicates, causing
  `OpenView` to fail. ([#1184])

## [0.20.0] - 2026-03-21

> [!WARNING]
> `trigger_id` and `workflow_id` are NOT in any documentation or in any of the official
libraries, so exercise caution if you use these.

### Added

- **`workflow_id` and `trigger_id` in `Message`** — It seems that some types of messages,
    e.g: `bot_message`, can carry `trigger_id` and `workflow_id`.
- **`RichTextQuote.Border` field** — optional border toggle (matches the docs now)
- **`RichTextPreformatted.Language` field** — enables syntax highlighting for preformatted
  blocks

### Fixed

- **Remove embedding of `RichTextSection`** — `RichTextQuote` and `RichTextPreformatted`
  are now flattened as they should have always been. This is a breaking change for anyone
  using these structs directly.

## [0.19.0] - 2026-03-04

### Added

- **Optional HTTP retry for Web API** — Retries are off by default. Enable with `OptionRetry(n)` for 429-only retries or `OptionRetryConfig(cfg)` for full control including 5xx and connection errors with exponential backoff. ([#1532])
- **`task_card` and `plan` agent blocks** — New block types for task cards and plan agent blocks. ([#1536])

### Changed

- CI: bumped `actions/stale` from 10.1.1 to 10.2.0. ([#1534])
- Use `golangci-lint` in Makefile. ([#1533])

## [0.18.0] - 2026-02-21

### Added

- **`focus_on_load` support for remaining block elements** — Static/external/users/conversations/channels select, multi-select variants, datepicker, timepicker, plain_text_input, checkboxes, radio_buttons, and number_input. ([#1519])
- **`PlainText` and `PreviewPlainText` fields on `File`** — Email file objects now include the plain text body fields instead of silently discarding them. ([#1522])
- **Missing fields on `User`, `UserProfile`, and `EnterpriseUser`** — `who_can_share_contact_card`, `always_active`, `pronouns`, `image_1024`, `is_custom_image`, `status_text_canonical`, `huddle_state`, `huddle_state_expiration_ts`, `start_date`, and `is_primary_owner`. ([#1526])
- **Work Objects support** — Chat unfurl with Work Object metadata, entity details (flexpane), `entity_details_requested` event, and associated types (`WorkObjectMetadata`, `WorkObjectEntity`, `WorkObjectExternalRef`). ([#1529])
- **`admin.roles.*` API methods** — `admin.roles.listAssignments`, `admin.roles.addAssignments`, and `admin.roles.removeAssignments`. ([#1520])

### Fixed

- **`UserProfile.Skype` JSON tag** — Corrected typo from `"skyp"` to `"skype"`. ([#1524])
- **`assistant.threads.setSuggestedPrompts` title parameter** — Title is now sent when non-empty. ([#1528])

### Changed

- CI test matrix updated: dropped Go 1.24, added Go 1.26; bumped golangci-lint to v2.10.1. ([#1530])

## [0.18.0-rc2] - 2026-01-28

### Added

- **Audit Logs example** - New example demonstrating how to use the Audit Logs API. ([#1144])
- **Admin Conversations API support** - Comprehensive support for `admin.conversations.*`
  methods including core operations (archive, unarchive, create, delete, rename, invite,
  search, lookup, getTeams, convertToPrivate, convertToPublic, disconnectShared, setTeams),
  bulk operations (bulkArchive, bulkDelete, bulkMove), preferences, retention management,
  restrict access controls, and EKM channel info. ([#1329])

### Changed

- **BREAKING**: Removed deprecated `UploadFile`, `UploadFileContext`, and
  `FileUploadParameters`. The `files.upload` API was discontinued by Slack on November
  12, 2025. ([#1481])
- **BREAKING**: Renamed `UploadFileV2` → `UploadFile`, `UploadFileV2Context` →
  `UploadFileContext`, and `UploadFileV2Parameters` → `UploadFileParameters`. The "V2"
  suffix is no longer needed now that the old API is removed. ([#1481])

### Fixed

- **File upload error wrapping** - `UploadFile` now wraps errors with the step name
  (`GetUploadURLExternal`, `UploadToURL`, or `CompleteUploadExternal`) so callers can
  identify which of the three upload steps failed. ([#1491])
- **Audit Logs API endpoint** - Fixed `GetAuditLogs` to use the correct endpoint
  (`api.slack.com`) instead of the regular API endpoint (`slack.com/api`). The Audit
  Logs API requires a different base URL. Added `OptionAuditAPIURL` for testing. ([#1144])
- **Socket mode websocket dial debugging** - Added debug logging when a custom dialer is
  used including HTTP response status on dial failures. This helps diagnose proxy/TLS
  issues like "bad handshake" errors. ([#1360])
- **`MsgOptionPostMessageParameters` now passes `MetaData`** - Previously, metadata was
  silently dropped when using `PostMessageParameters`. ([#1343])

## [0.18.0-rc1] - 2026-01-26

### Added

- **Huddle support** - New `HuddleRoom`, `HuddleParticipantEvent`, and `HuddleRecording`
  types for handling Slack huddle events (`huddle_thread` subtype messages).
- **Call block data parsing** - `CallBlock` now includes full call data when retrieved
  from Slack messages, with new `CallBlockData`, `CallBlockDataV1`, and `CallBlockIconURLs`
  types. ([#897])
- **Chat Streaming API support** - New streaming API for real-time chat interactions
  with example usage. ([#1506])
- **Data Access API support** - Full support for Slack's Data Access API with
  example implementation. ([#1439])
- **Cursor-based pagination for `GetUsers`** - More efficient user retrieval
  with cursor pagination. ([#1465])
- **`GetAllConversations` with pagination** - Retrieve all conversations with
  automatic pagination handling, including rate limit and server error handling. ([#1463])
- **Table blocks support** - Parse and create table blocks with proper
  unmarshaling. ([#1490], [#1511])
- **Context actions block support** - New `context_actions` block type. ([#1495])
- **Workflow button block element** - Support for `workflow_button` in block
  elements. ([#1499])
- **`loading_messages` parameter for `SetAssistantThreadsStatus`** - Optional
  parameter to customize loading state messages. ([#1489])
- **Attachment image fields** - Added `ImageBytes`, `ImageHeight`, and `ImageWidth`
  fields to attachments. ([#1516])
- **`RecordChannel` to conversation properties** - New property for conversation
  metadata. ([#1513])
- **Title argument for `CreateChannelCanvas`** - Canvas creation now supports
  custom titles. ([#1483])
- **`PostEphemeral` handler for slacktest** - Audit outgoing ephemeral messages
  in test environments. ([#1517])
- **`PreviewImageName` for remote files** - Customize preview image filename
  instead of using the default `preview.jpg`.

### Fixed

- **`PublishView` no longer sends empty hash** - Prevents unnecessary payload
  when hash is empty. ([#1515])
- **`ImageBlockElement` validation** - Now properly validates that either
  `imageURL` or `SlackFile` is provided. ([#1488])
- **Rich text section channel return** - Correctly returns channel for section
  channel rich text elements. ([#1472])
- **`KickUserFromConversation` error handling** - Errors are now properly parsed
  as a map structure. ([#1471])

### Changed

- **BREAKING**: `GetReactions` now returns `ReactedItem` instead of `[]ItemReaction`.
  This aligns the response with the actual Slack API, which includes the item itself
  (message, file, or file_comment) alongside reactions. To migrate, use `resp.Reactions`
  to access the slice of reactions. ([#1480])
- **BREAKING**: `Settings` struct fields `Interactivity` and `EventSubscriptions`
  are now pointers, allowing them to be omitted when empty. ([#1461])
- Minimum Go version bumped to 1.24. ([#1504])

## [0.17.3] - 2025-07-04

Previous release. See [GitHub releases](https://github.com/slack-go/slack/releases/tag/v0.17.3)
for details.

[#897]: https://github.com/slack-go/slack/issues/897
[#1236]: https://github.com/slack-go/slack/issues/1236
[#1257]: https://github.com/slack-go/slack/issues/1257
[#1144]: https://github.com/slack-go/slack/issues/1144
[#1329]: https://github.com/slack-go/slack/issues/1329
[#1343]: https://github.com/slack-go/slack/issues/1343
[#1360]: https://github.com/slack-go/slack/issues/1360
[#1439]: https://github.com/slack-go/slack/pull/1439
[#1461]: https://github.com/slack-go/slack/pull/1461
[#1463]: https://github.com/slack-go/slack/pull/1463
[#1465]: https://github.com/slack-go/slack/pull/1465
[#1471]: https://github.com/slack-go/slack/pull/1471
[#1472]: https://github.com/slack-go/slack/pull/1472
[#1480]: https://github.com/slack-go/slack/pull/1480
[#1483]: https://github.com/slack-go/slack/pull/1483
[#1488]: https://github.com/slack-go/slack/pull/1488
[#1489]: https://github.com/slack-go/slack/pull/1489
[#1490]: https://github.com/slack-go/slack/pull/1490
[#1491]: https://github.com/slack-go/slack/issues/1491
[#1495]: https://github.com/slack-go/slack/pull/1495
[#1497]: https://github.com/slack-go/slack/pull/1497
[#1499]: https://github.com/slack-go/slack/pull/1499
[#1504]: https://github.com/slack-go/slack/pull/1504
[#1506]: https://github.com/slack-go/slack/pull/1506
[#1511]: https://github.com/slack-go/slack/pull/1511
[#1513]: https://github.com/slack-go/slack/pull/1513
[#1515]: https://github.com/slack-go/slack/pull/1515
[#1516]: https://github.com/slack-go/slack/pull/1516
[#1517]: https://github.com/slack-go/slack/pull/1517
[#1519]: https://github.com/slack-go/slack/pull/1519
[#1520]: https://github.com/slack-go/slack/pull/1520
[#1522]: https://github.com/slack-go/slack/pull/1522
[#1524]: https://github.com/slack-go/slack/pull/1524
[#1526]: https://github.com/slack-go/slack/pull/1526
[#1528]: https://github.com/slack-go/slack/pull/1528
[#1529]: https://github.com/slack-go/slack/pull/1529
[#1530]: https://github.com/slack-go/slack/pull/1530
[#1532]: https://github.com/slack-go/slack/pull/1532
[#1533]: https://github.com/slack-go/slack/pull/1533
[#1534]: https://github.com/slack-go/slack/pull/1534
[#1536]: https://github.com/slack-go/slack/pull/1536
[#596]: https://github.com/slack-go/slack/issues/596
[#1541]: https://github.com/slack-go/slack/issues/1541
[#1172]: https://github.com/slack-go/slack/issues/1172
[#1076]: https://github.com/slack-go/slack/issues/1076
[#1157]: https://github.com/slack-go/slack/issues/1157
[#1196]: https://github.com/slack-go/slack/issues/1196
[#1547]: https://github.com/slack-go/slack/pull/1547

[Unreleased]: https://github.com/slack-go/slack/compare/v0.35.0...HEAD
[0.35.0]: https://github.com/slack-go/slack/compare/v0.34.0...v0.35.0
[0.34.0]: https://github.com/slack-go/slack/compare/v0.33.0...v0.34.0
[0.33.0]: https://github.com/slack-go/slack/compare/v0.32.0...v0.33.0
[0.32.0]: https://github.com/slack-go/slack/compare/v0.31.0...v0.32.0
[0.31.0]: https://github.com/slack-go/slack/compare/v0.30.1...v0.31.0
[0.30.1]: https://github.com/slack-go/slack/compare/v0.30.0...v0.30.1
[0.30.0]: https://github.com/slack-go/slack/compare/v0.29.0...v0.30.0
[0.29.0]: https://github.com/slack-go/slack/compare/v0.28.0...v0.29.0
[0.28.0]: https://github.com/slack-go/slack/compare/v0.27.0...v0.28.0
[0.27.0]: https://github.com/slack-go/slack/compare/v0.26.0...v0.27.0
[0.26.0]: https://github.com/slack-go/slack/compare/v0.25.0...v0.26.0
[0.25.0]: https://github.com/slack-go/slack/compare/v0.24.0...v0.25.0
[0.24.0]: https://github.com/slack-go/slack/compare/v0.23.1...v0.24.0
[0.23.1]: https://github.com/slack-go/slack/compare/v0.23.0...v0.23.1
[0.23.0]: https://github.com/slack-go/slack/compare/v0.22.0...v0.23.0
[0.22.0]: https://github.com/slack-go/slack/compare/v0.21.1...v0.22.0
[0.21.1]: https://github.com/slack-go/slack/compare/v0.21.0...v0.21.1
[0.21.0]: https://github.com/slack-go/slack/compare/v0.20.0...v0.21.0
[0.20.0]: https://github.com/slack-go/slack/compare/v0.19.0...v0.20.0
[0.19.0]: https://github.com/slack-go/slack/compare/v0.18.0...v0.19.0
[0.18.0]: https://github.com/slack-go/slack/compare/v0.18.0-rc2...v0.18.0
[0.18.0-rc2]: https://github.com/slack-go/slack/releases/tag/v0.18.0-rc2
[0.18.0-rc1]: https://github.com/slack-go/slack/releases/tag/v0.18.0-rc1
[0.17.3]: https://github.com/slack-go/slack/releases/tag/v0.17.3
