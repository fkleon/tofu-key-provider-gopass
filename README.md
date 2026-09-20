# OpenTofu gopass external key provider

This program implements OpenTofu's [external key provider protocol](https://opentofu.org/docs/language/state/encryption/#external-key-provider) and reads a key from [gopass](https://www.gopass.pw/).

It writes the required provider header to stdout, reads the JSON request from stdin, and returns the gopass secret as a base64-encoded encryption key. The secret path is also returned as `external_data`, so OpenTofu can pass the metadata back on a later run.

## Requirements

- OpenTofu configured to use an external key provider.
- A working gopass store. Initialize one with `gopass setup` if needed.
- Go 1.27.1 or a prebuilt `tofu-key-provider-gopass` binary.

## Build

```sh
git clone https://github.com/fkleon/tofu-key-provider-gopass.git
cd tofu-key-provider-gopass
go build -o tofu-key-provider-gopass .
```

The Makefile provides the same build as `make` and produces a statically linked, stripped binary when supported by the environment.

## Configure OpenTofu

The basic configuration uses a default secret path, `opentofu/state`:

```hcl
encryption {
  key_provider "external" "gopass" {
    command = ["/path/to/tofu-key-provider-gopass"]
  }
}
```

For decryption, the provider reuses the path saved in `external_data.path`.

The exact `encryption` block should match the OpenTofu version and encryption configuration you use; see the [OpenTofu encryption documentation](https://opentofu.org/docs/language/state/encryption/) for the surrounding configuration.

### Custom secret path

Pass a secret path as the first argument when you want to use a path other than the default:

```hcl
command = ["/path/to/tofu-key-provider-gopass", "infra/opentofu"]
```

An explicit path takes precedence over the path saved in `external_data.path`.

### Custom gopass store

Select a non-default password store by setting the `PASSWORD_STORE_DIR` environment variable:

```sh
PASSWORD_STORE_DIR=/path/to/password-store tofu-key-provider-gopass
```

## Behavior

- A request with `external_data: null` produces a new encryption key.
- A request with existing metadata produces encryption and decryption keys.
- Errors, including a missing gopass store or secret, are written to stderr and cause a non-zero exit.

## Tests

Run the unit tests with:

```sh
go test ./...
```
