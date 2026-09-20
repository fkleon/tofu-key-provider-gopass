package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/gopasspw/gopass/pkg/gopass"
	"github.com/gopasspw/gopass/pkg/gopass/api"
)

// Build metadata. These variables are set at build time using -ldflags.
var (
	name    string = "tofu-key-provider-gopass"
	version string = "dev"
	date    string
)

// Command-line flags.
var (
	secretPath  string
	storeDir    string
	versionFlag bool
)

// The environment variable used to configure the gopass store directory.
const passwordStoreEnv = "PASSWORD_STORE_DIR"

func init() {
	const defaultSecretPath = "opentofu/state"
	flag.StringVar(&secretPath, "path", defaultSecretPath, "secret path in gopass")
	flag.StringVar(&storeDir, "store", os.Getenv(passwordStoreEnv), "password store directory")
	flag.BoolVar(&versionFlag, "version", false, "print version and exit")
}

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

// secretPathFor returns the configured gopass path. The path stored in existing
// metadata takes precendence, followed by the command-line argument.
// If neither is set, an error is returned.
func secretPathFor(input Input, secretPath string) (string, error) {
	if input != nil {
		if path, ok := input.ExternalData["path"]; ok {
			path, ok := path.(string)
			if !ok || path == "" {
				return "", fmt.Errorf("external_data.path must be a non-empty string")
			}
			log.Printf("Using secret path: %q (from external data)", path)
			return path, nil
		}
	}
	if secretPath != "" {
		log.Printf("Using secret path: %q (from CLI)", secretPath)
		return secretPath, nil
	}
	return "", fmt.Errorf("secret path is required")
}

// passwordStoreDirFor returns the gopass store directory. The directory stored in
// existing metadata takes precedence, followed by the command-line argument or
// environment variable. If neither is set, an empty string is returned.
func passwordStoreDirFor(input Input, storeDir string) (string, error) {
	if input != nil {
		if storeDir, ok := input.ExternalData["store"]; ok {
			storeDir, ok := storeDir.(string)
			if !ok || storeDir == "" {
				return "", fmt.Errorf("external_data.store must be a non-empty string")
			}
			log.Printf("Using password store dir: %q (from external data)", storeDir)
			return storeDir, nil
		}
	}
	if storeDir != "" {
		log.Printf("Using password store dir: %q (from CLI/env)", storeDir)
		return storeDir, nil
	}
	// If no store dir is configured, gopass will use its default.
	return "", nil
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

func lookupEncryptionSecret(storeDir, secretPath string) (gopass.Secret, error) {
	ctx := context.Background()

	// If a custom store dir is configured, set the PASSWORD_STORE_DIR env
	// which is respected by gopass.
	if storeDir != "" {
		if err := os.Setenv(passwordStoreEnv, storeDir); err != nil {
			return nil, fmt.Errorf("configure password store: %w", err)
		}
	}

	gp, err := api.New(ctx)
	if err != nil {
		return nil, fmt.Errorf("initialise password store (dir=%q): %w", storeDir, err)
	}

	sec, err := gp.Get(ctx, secretPath, "latest")
	if err != nil {
		return nil, fmt.Errorf("get secret (store=%q path=%q): %w", storeDir, secretPath, err)
	}

	return sec, nil
}

func main() {
	// Write logs to stderr
	log.Default().SetOutput(os.Stderr)

	flag.Parse()

	if versionFlag {
		fmt.Printf("%s\n\n", name)
		fmt.Printf("%-14s %s\n", "GitVersion:", version)
		fmt.Printf("%-14s %s\n", "BuildDate:", date)
		os.Exit(0)
	}

	// Write the header
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

	// Lookup secret path and password store dir from input data,
	// with fallback to command-line args and env vars.
	storeDir, err := passwordStoreDirFor(inMeta, storeDir)
	if err != nil {
		log.Fatalf("Failed to determine password store: %v", err)
	}
	secretPath, err := secretPathFor(inMeta, secretPath)
	if err != nil {
		log.Fatalf("Failed to determine secret path: %v", err)
	}

	// Lookup the encryption secret from gopass.
	sec, err := lookupEncryptionSecret(storeDir, secretPath)
	if err != nil {
		log.Fatalf("Failed to lookup encryption key: %v", err)
	}

	key := []byte(sec.Password())
	data := make([]byte, base64.StdEncoding.EncodedLen(len(key)))
	base64.StdEncoding.Encode(data, key)

	var keys Keys
	keys.EncryptionKey = data
	if inMeta != nil {
		keys.DecryptionKey = data
	}

	externalData := map[string]any{
		"path": secretPath,
	}
	if storeDir != "" {
		externalData["store"] = storeDir
	}
	output := Output{
		Keys: keys,
		Meta: Metadata{ExternalData: externalData},
	}
	outputData, err := json.Marshal(output)
	if err != nil {
		log.Fatalf("Failed to encode output: %v", err)
	}
	_, _ = os.Stdout.Write(outputData)
}
