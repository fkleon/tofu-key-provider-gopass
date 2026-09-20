package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/gopasspw/gopass/pkg/gopass"
	"github.com/gopasspw/gopass/pkg/gopass/api"
)

const defaultSecretPath = "opentofu/state"

// Header is the initial greeting the key provider sends out.
type Header struct {
	// Magic must always be OpenTofu-External-Keyprovider
	Magic string `json:"magic"`
	// Version must be 1.
	Version int `json:"version"`
}

// Metadata describes both the input and the output metadata.
type Metadata struct {
	ExternalData map[string]any `json:"external_data"`
}

// Input describes the input data structure. This is nil on input if no existing
// data needs to be decrypted.
type Input *Metadata

// secretPathFor returns the configured gopass path. An explicit command-line
// path takes precedence, followed by the path stored in existing metadata, and
// finally the default path.
func secretPathFor(args []string, input Input) (string, error) {
	if len(args) > 0 && args[0] != "" {
		return args[0], nil
	}
	if input != nil {
		if path, ok := input.ExternalData["path"]; ok {
			path, ok := path.(string)
			if !ok || path == "" {
				return "", fmt.Errorf("external_data.path must be a non-empty string")
			}
			return path, nil
		}
	}
	return defaultSecretPath, nil
}

// parseInput parses the metadata sent by OpenTofu. OpenTofu sends an object
// with external_data set to null when it is encrypting new data. Treat that as
// no existing metadata, rather than as a decryption request.
func parseInput(data []byte) (Input, error) {
	var input Input
	if err := json.Unmarshal(data, &input); err != nil {
		return nil, err
	}
	if input != nil && input.ExternalData == nil {
		return nil, nil
	}
	return input, nil
}

type Keys struct {
	// EncryptionKey must always be provided.
	EncryptionKey []byte `json:"encryption_key,omitempty"`
	// DecryptionKey must be provided when the input metadata is present.
	DecryptionKey []byte `json:"decryption_key,omitempty"`
}

// Output describes the output data written to stdout.
type Output struct {
	Keys Keys `json:"keys"`
	// Meta contains the metadata to store alongside the encrypted data. You can
	// store data here you need to reconstruct the decryption key later.
	Meta Metadata `json:"meta"`
}

func lookupEncryptionSecret(secretPath string) (gopass.Secret, error) {
	ctx := context.Background()

	gp, err := api.New(ctx)
	if err != nil {
		return nil, err
	}

	sec, err := gp.Get(ctx, secretPath, "latest")
	if err != nil {
		return nil, err
	}

	return sec, nil
}

func main() {
	// Write logs to stderr
	log.Default().SetOutput(os.Stderr)

	// Write the header:
	header := Header{
		"OpenTofu-External-Key-Provider",
		1,
	}
	marshalledHeader, err := json.Marshal(header)
	if err != nil {
		log.Fatalf("%v", err)
	}
	_, _ = os.Stdout.Write(append(marshalledHeader, []byte("\n")...))

	// Read the input
	input, err := io.ReadAll(os.Stdin)
	if err != nil {
		log.Fatalf("Failed to read stdin: %v", err)
	}
	inMeta, err := parseInput(input)
	if err != nil {
		log.Fatalf("Failed to parse stdin: %v", err)
	}
	secretPath, err := secretPathFor(os.Args[1:], inMeta)
	if err != nil {
		log.Fatalf("Failed to determine secret path: %v", err)
	}

	var keys Keys

	sec, err := lookupEncryptionSecret(secretPath)
	if err != nil {
		log.Fatalf("Failed to lookup encryption key: %v", err)
	}

	key := []byte(sec.Password())
	data := make([]byte, base64.StdEncoding.EncodedLen(len(key)))
	base64.StdEncoding.Encode(data, key)
	keys.EncryptionKey = data
	if inMeta != nil {
		keys.DecryptionKey = data
	}

	output := Output{
		Keys: keys,
		Meta: Metadata{
			ExternalData: map[string]any{
				"path": secretPath,
			},
		},
	}
	outputData, err := json.Marshal(output)
	if err != nil {
		log.Fatalf("Failed to encode output: %v", err)
	}
	_, _ = os.Stdout.Write(outputData)
}
