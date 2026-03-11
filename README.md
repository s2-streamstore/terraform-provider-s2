<div align="center">
  <p>
    <!-- Light mode logo -->
    <a href="https://s2.dev#gh-light-mode-only">
      <img src="https://raw.githubusercontent.com/s2-streamstore/s2-sdk-rust/main/assets/s2-black.png" height="60">
    </a>
    <!-- Dark mode logo -->
    <a href="https://s2.dev#gh-dark-mode-only">
      <img src="https://raw.githubusercontent.com/s2-streamstore/s2-sdk-rust/main/assets/s2-white.png" height="60">
    </a>
  </p>

  <h1>Terraform Provider for S2</h1>

  <p>
    <a href="https://github.com/s2-streamstore/terraform-provider-s2/actions/workflows/test.yml"><img src="https://github.com/s2-streamstore/terraform-provider-s2/actions/workflows/test.yml/badge.svg" /></a>
    <a href="https://discord.gg/vTCs7kMkAf"><img src="https://img.shields.io/discord/1209937852528599092?logo=discord" /></a>
  </p>
</div>

Terraform provider for managing [S2](https://s2.dev) basins, streams, and access
tokens.

## Getting started

1. Install Terraform (version `>= 1.5.0`).

2. Generate an S2 access token from the [S2 dashboard](https://s2.dev/dashboard).

3. Set provider credentials:
   ```bash
   export S2_ACCESS_TOKEN="<your access token>"
   # Optional:
   # export S2_ACCOUNT_ENDPOINT="aws.s2.dev"
   ```

4. Configure the provider:
   ```terraform
   terraform {
     required_providers {
       s2 = {
         source = "s2-streamstore/s2"
       }
     }
   }

   provider "s2" {}
   ```

5. Create resources:
   ```terraform
   resource "s2_basin" "example" {
     name = "my-example-basin"
   }

   resource "s2_stream" "events" {
     basin = s2_basin.example.name
     name  = "events"
   }
   ```

6. Apply:
   ```bash
   terraform init
   terraform apply
   ```

## Resources and data sources

Resources:

- `s2_basin` - [`docs/resources/basin.md`](./docs/resources/basin.md)
- `s2_stream` - [`docs/resources/stream.md`](./docs/resources/stream.md)
- `s2_access_token` - [`docs/resources/access_token.md`](./docs/resources/access_token.md)

Data sources:

- `s2_basin` - [`docs/data-sources/basin.md`](./docs/data-sources/basin.md)
- `s2_stream` - [`docs/data-sources/stream.md`](./docs/data-sources/stream.md)

## Examples

The [`examples`](./examples) directory includes ready-to-run configurations:

| Example | Description |
|---------|-------------|
| [`examples/provider/main.tf`](./examples/provider/main.tf) | Provider configuration |
| [`examples/resources/s2_basin/resource.tf`](./examples/resources/s2_basin/resource.tf) | Create a basin |
| [`examples/resources/s2_stream/resource.tf`](./examples/resources/s2_stream/resource.tf) | Create a stream |
| [`examples/resources/s2_access_token/resource.tf`](./examples/resources/s2_access_token/resource.tf) | Issue an access token |
| [`examples/data-sources/s2_basin/data-source.tf`](./examples/data-sources/s2_basin/data-source.tf) | Read basin metadata |
| [`examples/data-sources/s2_stream/data-source.tf`](./examples/data-sources/s2_stream/data-source.tf) | Read stream metadata |
| [`examples/full-workflow/main.tf`](./examples/full-workflow/main.tf) | End-to-end basin, stream, token, and data-source workflow |

## Development

Requirements:

- Go `1.24+`
- Terraform `>= 1.5.0`

Common commands:

```bash
go mod download
go build ./...
go test ./...
```

Using make targets:

```bash
make build
make install
```

Regenerate provider docs (requires `tfplugindocs`):

```bash
tfplugindocs generate --provider-name s2 --tf-version 1.14.6
```

Run acceptance tests:

```bash
export S2_ACCESS_TOKEN="<your access token>"
make testacc
```

Run acceptance tests against an already-running local `s2-lite`:

```bash
make testacc-lite
```

To start and stop `s2-lite` automatically with the `s2` CLI binary:

```bash
make testacc-lite-managed
```

You can override defaults if needed (port, URLs, timeout, binary path):

```bash
make testacc-lite-managed S2_LITE_BIN=s2 S2_LITE_PORT=18080 S2_LITE_WAIT_SECS=300
```

Note: `s2-lite` currently does not implement `/access-tokens`, so access-token
acceptance tests are skipped for Lite acceptance targets.

## Feedback

Please use [GitHub Issues](https://github.com/s2-streamstore/terraform-provider-s2/issues)
to report bugs, request features, or share feedback.

### Contributing

Pull requests are welcome. If possible, open an issue first to discuss larger
changes before submitting a PR.

## Reach out to us

Join our [Discord](https://discord.gg/vTCs7kMkAf) server or email
[hi@s2.dev](mailto:hi@s2.dev).
