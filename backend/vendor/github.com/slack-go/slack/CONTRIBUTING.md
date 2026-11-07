# Contributing

Welcome! We are glad that you want to contribute to our project! 💖

## Pull requests

- Discuss new features in an issue first. For a bug, describe how to reproduce it.
- Branch from `master` and keep each PR focused. Keep unrelated formatting and refactors separate.
- Add tests and update documentation for changed behavior. Add a short `[Unreleased]` changelog
  entry for library-visible changes; refine an existing entry when the feature is still unreleased.
- Use conventional commit subjects and PR titles: `fix(block): preserve table cells`.
  Subjects are imperative, lowercase, and have no period. Use `!` for breaking changes.
  If conventional commit subjects makes it for a worse read, then don't. Above conventional commits is communicating intent well.
- Do not force-push or rewrite reviewed history. Add follow-up commits; PRs are squash-merged.
  Each commit should build and pass checks so regressions remain easy to bisect.

## Checks

Before submitting a code or example change, run:

```sh
make pr-prep
```

This formats code and runs lint checks, the full race suite,
and module and whitespace checks. Use `make lint` to run the lint checks alone.
The tidy check reports required changes without modifying `go.mod` or `go.sum`.

Supported Go releases are listed in the CI matrix. Tool versions live in `mise.toml`,
`go.mod`, `.github/workflows/test.yml`, and the `Makefile`; keep relevant pins aligned when
updating them. Compare exact versions when a local check passes but CI fails.

## Code conventions

Keep Web API methods in the root `slack` package, grouped by method family. Copy the closest
current sibling rather than introducing another API style.

- Provide `Foo(...)` and `FooContext(ctx, ...)`; the former calls the latter with
  `context.Background()`.
- New methods take required positional arguments and method-scoped functional options for
  optional parameters. Ignore nil options. Keep existing parameter structs on shipped methods;
  extend other existing methods with scoped variadic options rather than `FooWithOptions` pairs.
  Do not add new uses of the generic `ParamOption`.
- Reuse the client's request helpers and `SlackResponse` handling. Match Slack's wire format,
  use cursor pagination, and preserve explicit false or zero values when omission differs.
- Block constructors follow `New<Name>Block(required..., options ...<Name>BlockOption)` with
  nil-guarded options. Register new types in every applicable decoder: blocks, elements,
  inputs, accessories, and rich-text containers. Add real-payload round-trip coverage.
- Register Events API types in `EventsAPIInnerEventMapping` and test them through `ParseEvent`.
  A decoding error can drop the whole event, so handle union-shaped fields deliberately.

Exported declarations need [Go doc comments](https://go.dev/doc/comment). Link Web API methods
to the current Slack reference; do not repeat the reference page in the comment:

```go
// GetUserInfo gets information about a user.
//
// Slack API docs: https://docs.slack.dev/reference/methods/users.info
```

Other comments should explain non-obvious reasons, not narrate the code.

## Tests and Slack behavior

- For a bug fix, first write a regression test and confirm it fails for the reported reason.
  Keep tests beside the code and small captured payloads as inline JSON; do not add `testdata/`.
- Round-trip tests must preserve the actual payload. Fix the model instead of removing fields
  from the fixture. Synchronize concurrent tests without sleeps.
- Check the current [Slack reference](https://docs.slack.dev/reference/), then official SDKs
  (`@slack/types`, `@slack/web-api`, Python, and Java). Use observed live payloads to resolve
  disagreements. Cite the source of new fields; note undocumented fields in the changelog.
- When live behavior matters, provide a runnable example with exact commands, scopes, app setup,
  and trigger actions. Distinguish API acceptance, returned data, Go decoding, and manual UI
  review. An accepted request does not prove a preview works; an omitted field remains unverified.

Follow [examples/README.md](examples/README.md): credentials come from environment variables;
IDs and modes come from flags. Prefer Socket Mode for interaction examples unless HTTP behavior
is the subject. Redact credentials from captured JSON and print Slack's
`ResponseMetadata.Messages` when debugging errors. Decoder or preservation failures must fail
the example. Keep throwaway verification programs out of the PR unless they are useful to users.

## Compatibility

Preserve shipped identifiers, signatures, field types, JSON tags, and wire output. Prefer
additive changes and explain any break. Adding slices or maps can make a struct non-comparable;
new concrete decoder types can break type assertions. Variadic options preserve direct calls
but change method types, interfaces, and mocks. Do not replace `omitempty` with `omitzero` as cleanup.

Breaking changes use a changelog warning and migration example where needed. New features and
breaking changes bump MINOR; fix-only releases bump PATCH (`v0.MINOR.PATCH`). The maintainer
chooses the release. Tests, examples, CI, and documentation-only changes need no library changelog entry.
