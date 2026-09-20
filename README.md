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

The external provider command is an array. Pass the gopass path as the first argument after the binary name:

```hcl
encryption {
  key_provider "external" "gopass" {
    command = ["/path/to/tofu-key-provider-gopass", "infra/opentofu"]
  }
}
```

The exact `encryption` block should match the OpenTofu version and encryption configuration you use; see the [OpenTofu encryption documentation](https://opentofu.org/docs/language/state/encryption/) for the surrounding configuration.

## Protocol behavior

- The program prints the `OpenTofu-External-Key-Provider` header before reading stdin.
- A request whose `external_data` is `null` is treated as a new-encryption request and produces only an encryption key.
- A request containing existing `external_data` produces both encryption and decryption keys.
- Errors, including a missing gopass store or secret, are written to stderr and cause a non-zero exit.

Do not use a gopass path that exposes sensitive information in logs or process-inspection tooling. Keep the gopass store and its backing keys protected.

## Tests

Run the unit tests with:

```sh
go test ./...
```
