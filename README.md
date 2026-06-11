# Open Location Hub CLI

`olh` is a Go CLI for the Open Location Hub REST and WebSocket APIs.

Current scope:
- CRUD for `zones`, `trackables`, `providers`, and `fences`
- collection `summary` and `delete-all` operations for core resources
- location and proximity ingest
- transient-state inspection for locations, motions, sensors, fences, and providers
- zone transform and derived-fence helpers
- JSON-RPC discovery and invocation
- WebSocket subscribe and publish
- `login` flow that validates hub and OAuth settings and saves them
- human-readable output by default
- machine-readable JSON with `--json`
- automatic OAuth token fetch from `~/.openlocationhub.env` or `OLH_*` environment variables

## Install

Install `olh` via Homebrew:

```bash
brew tap jillesvangurp/tap
brew install jillesvangurp/tap/open-location-hub-cli
```

Bash and Zsh completions are installed automatically.

## Build

```bash
just build
just build-all
```

Artifacts are written to `dist/`.

## Release

Build and package a release locally:

```bash
just clean
VERSION=0.1.0 just package-release
```

Packaged archives and `checksums.txt` are written to `release/`.

Publish the GitHub release by pushing a matching tag:

```bash
git tag 0.1.0
git push origin 0.1.0
```

The release workflow uploads the packaged files and uses GitHub-generated release notes.

## Auth

The CLI resolves auth configuration in this order:

1. explicit command-line flags
2. process environment variables
3. `~/.openlocationhub.env`

All commands accept the global flags for these values. Common aliases:

- `--hub-endpoint` or `--base-url`
- `--token-endpoint` or `--oauth-token-url`
- `--client-id` or `--oauth-client-id`
- `--client-secret` or `--oauth-client-secret`

Supported variables:

```bash
OLH_BASE_URL=http://localhost:8080
OLH_TOKEN=
OLH_OAUTH_TOKEN_URL=http://localhost:5556/dex/token
OLH_OAUTH_CLIENT_ID=open-location-cli
OLH_OAUTH_CLIENT_SECRET=cli-secret
OLH_OAUTH_USERNAME=admin@example.com
OLH_OAUTH_PASSWORD=testpass123
OLH_OAUTH_SCOPE=openid email profile
OLH_OAUTH_GRANT_TYPE=password
OLH_OAUTH_AUDIENCE=
```

If `OLH_TOKEN` is empty and OAuth settings are present, `olh` fetches an access token automatically before REST and WebSocket operations.

Save validated settings:

```bash
olh login \
  --hub-endpoint http://localhost:8080 \
  --token-endpoint http://localhost:5556/dex/token \
  --client-id open-location-cli \
  --client-secret cli-secret \
  --oauth-username admin@example.com \
  --oauth-password testpass123
```

Fetch just the token:

```bash
olh auth token
```

## Examples

```bash
olh zones list
olh zones summary
olh zones transform 0190c9d2-6f54-7ccf-8f55-f34eb0bf01f1 -f point.json
olh zones create-fence 0190c9d2-6f54-7ccf-8f55-f34eb0bf01f1
olh zones get 0190c9d2-6f54-7ccf-8f55-f34eb0bf01f1
olh trackables create -f trackable.json
olh trackables motions
olh trackables location 0190c9d2-6f54-7ccf-8f55-f34eb0bf01f1
olh providers sensors uwb-sim-demo
olh providers update-location uwb-sim-demo -f location.json
olh locations list
olh locations replace -f locations.json
olh locations post -f locations.json
olh proximities replace -f proximities.json
olh locations stream > location_updates.ndjson
olh locations stream --create-trackables > location_updates.ndjson
olh trackables stream > trackable_motions.ndjson
olh fences stream > fence_events.ndjson
olh collisions stream > collision_events.ndjson
./scripts/replay_simulated_tag_1.sh
olh rpc available
olh rpc call -f request.json
olh ws subscribe --topic location_updates --param provider_id=uwb-sim-demo
olh ws publish --topic location_updates -f locations.json
```

Replay the `simulated-tag-1` verification sequence. The script first posts an
outside point to reset fence state, then replays the three inside updates with
3-second spacing:

```bash
./scripts/replay_simulated_tag_1.sh
```

Optional overrides:

```bash
SLEEP_SECONDS=1 PROVIDER_ID=simulated-tag-1 SOURCE_ID=simulated-tag-1 ./scripts/replay_simulated_tag_1.sh
```

## Notes

- `create` and `update` commands accept JSON or YAML payloads.
- `locations stream`, `trackables stream`, `fences stream`, and `collisions stream` emit NDJSON records shaped as `{received_at, topic, message}`.
- `zones`, `trackables`, and `fences` resource IDs are validated as UUIDs before requests are sent.
- The generated REST client is derived from `api/omlox-hub.v0.yaml`.
- `oapi-codegen` currently warns about OpenAPI 3.1 support. The CLI still builds and uses the generated client.
