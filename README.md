# OpenTofu gopass external key provider

This program implements OpenTofu's [external key provider protocol](https://opentofu.org/docs/language/state/encryption/#external-key-provider) and reads a key from [gopass](https://www.gopass.pw/).

It writes the required provider header to stdout, reads the JSON request from stdin, and returns the gopass secret as a base64-encoded encryption key.

The configured secret path and, when specified, password store are stored in `external_data`, so OpenTofu can pass the metadata back on a later run.

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

The Makefile builds a stripped binary and embeds the current Git version and build date.

## Configure OpenTofu

The basic configuration uses the default secret path, `opentofu/state`.

This external key provider is meant to be chained together with the `pbkdf2` provider as in this example:

```hcl
encryption {
  key_provider "external" "gopass" {
    command = ["/path/to/tofu-key-provider-gopass"]
  }

  key_provider "pbkdf2" "this" {
    chain = key_provider.external.gopass
  }
}
```

For decryption, the provider reuses the path saved in `external_data.path`.

The exact `encryption` block should match the OpenTofu version and encryption configuration you use; see the [OpenTofu encryption documentation](https://opentofu.org/docs/language/state/encryption/) for the surrounding configuration.

### Custom secret path

Use `-path` to select a secret other than the default:

```hcl
command = ["/path/to/tofu-key-provider-gopass", "-path", "infra/opentofu"]
```

For decryption, a path saved in `external_data.path` takes precedence over `-path`.

### Custom gopass store

Use `-store` or `PASSWORD_STORE_DIR` to select a non-default password store:

```hcl
command = ["/path/to/tofu-key-provider-gopass", "-store", "/path/to/password-store"]
```

The selected store is saved in `external_data.store` for later decryption requests. A stored value takes precedence over `-store` and `PASSWORD_STORE_DIR` when decrypting.

## Behaviour

- A request with `external_data: null` produces a new encryption key.
- A request with existing metadata produces encryption and decryption keys.
- `-version` prints build information and exits.
- Errors, including a missing gopass store or secret, are written to stderr and cause a non-zero exit.

## Tests

Run the unit tests with:

```sh
go test ./...
```
