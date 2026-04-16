# Contributing to terraform-provider-clickstack

Thank you for your interest in contributing!

## Development Setup

1. Clone the repository:
   ```bash
   git clone https://github.com/teamlapse/terraform-provider-clickstack.git
   cd terraform-provider-clickstack
   ```

2. Install dependencies:
   ```bash
   go mod download
   ```

3. Build the provider:
   ```bash
   go build -o terraform-provider-clickstack .
   ```

## Running Tests

### Unit Tests

```bash
go test ./... -v
```

### Acceptance Tests

Acceptance tests run against a live ClickStack instance. Set the required environment variables:

```bash
export CLICKSTACK_ORGANIZATION_ID="your-org-id"
export CLICKSTACK_SERVICE_ID="your-service-id"
export CLICKSTACK_API_KEY_ID="your-key-id"
export CLICKSTACK_API_KEY_SECRET="your-key-secret"
```

Then run:

```bash
TF_ACC=1 go test ./... -v -timeout 30m
```

## Generating Documentation

Documentation is generated from provider schema and example files using [tfplugindocs](https://github.com/hashicorp/terraform-plugin-docs):

```bash
go generate ./...
```

## Submitting Changes

1. Fork the repository and create your branch from `main`.
2. Add or update tests for your changes.
3. Run `go vet ./...` and ensure tests pass.
4. Run `go generate ./...` if you changed any schema or examples.
5. Submit a pull request with a clear description of the change.

## Code Style

- Follow standard Go conventions (`gofmt`, `go vet`).
- All source files must include the MPL-2.0 license header:
  ```go
  // Copyright (c) Lapse Technologies, Inc.
  // SPDX-License-Identifier: MPL-2.0
  ```

## Reporting Issues

Please open an issue on GitHub with:
- Terraform version (`terraform version`)
- Provider version
- Relevant Terraform configuration (redact sensitive values)
- Expected vs actual behavior
- Debug logs if applicable (`TF_LOG=DEBUG terraform plan`)
