package main

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestSecretPathFor(t *testing.T) {
	tests := []struct {
		name       string
		input      Input
		secretPath string
		want       string
		wantErr    bool
	}{
		{
			name:       "configured path",
			secretPath: "team/state",
			want:       "team/state",
		},
		{
			name: "stored path takes precedence",
			input: &Metadata{ExternalData: map[string]any{
				"path": "legacy/state",
			}},
			secretPath: "new/state",
			want:       "legacy/state",
		},
		{
			name:    "missing path",
			wantErr: true,
		},
		{
			name: "invalid stored path",
			input: &Metadata{ExternalData: map[string]any{
				"path": 42,
			}},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := secretPathFor(tt.input, tt.secretPath)
			if (err != nil) != tt.wantErr {
				t.Fatalf("secretPathFor() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("secretPathFor() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestPasswordStoreDirFor(t *testing.T) {
	stored := &Metadata{ExternalData: map[string]any{
		"store": "/stored/passwords",
	}}
	got, err := passwordStoreDirFor(stored, "/configured/passwords")
	if err != nil {
		t.Fatal(err)
	}
	if got != "/stored/passwords" {
		t.Errorf("passwordStoreDirFor() = %q, want %q", got, "/stored/passwords")
	}

	got, err = passwordStoreDirFor(nil, "/configured/passwords")
	if err != nil {
		t.Fatal(err)
	}
	if got != "/configured/passwords" {
		t.Errorf("passwordStoreDirFor() fallback = %q, want %q", got, "/configured/passwords")
	}

	got, err = passwordStoreDirFor(nil, "")
	if err != nil {
		t.Fatal(err)
	}
	if got != "" {
		t.Errorf("passwordStoreDirFor() default = %q, want empty string", got)
	}
}

func TestPasswordStoreDirForRejectsInvalidStoredValue(t *testing.T) {
	_, err := passwordStoreDirFor(&Metadata{ExternalData: map[string]any{
		"store": 42,
	}}, "")
	if err == nil {
		t.Fatal("passwordStoreDirFor() error = nil, want an error")
	}
}

func TestParseInput(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantNil  bool
		wantData map[string]any
		wantErr  bool
	}{
		{
			name:    "new encryption input has null external data",
			input:   `{"external_data":null}`,
			wantNil: true,
		},
		{
			name:     "existing encryption metadata",
			input:    `{"external_data":{"path":"prod/state"}}`,
			wantData: map[string]any{"path": "prod/state"},
		},
		{
			name:     "empty existing metadata object",
			input:    `{"external_data":{}}`,
			wantData: map[string]any{},
		},
		{
			name:    "invalid JSON",
			input:   `{`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseInput([]byte(tt.input))
			if (err != nil) != tt.wantErr {
				t.Fatalf("parseInput() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			if (got == nil) != tt.wantNil {
				t.Fatalf("parseInput() nil = %v, want %v", got == nil, tt.wantNil)
			}
			if got != nil && !reflect.DeepEqual(got.ExternalData, tt.wantData) {
				t.Errorf("parseInput() external data = %#v, want %#v", got.ExternalData, tt.wantData)
			}
		})
	}
}

func TestOutputOmitsUnconfiguredStore(t *testing.T) {
	output := Output{Meta: Metadata{ExternalData: map[string]any{
		"path": "opentofu/state",
	}}}
	data, err := json.Marshal(output)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != `{"keys":{},"meta":{"external_data":{"path":"opentofu/state"}}}` {
		t.Errorf("output = %s, want no empty store metadata", data)
	}
}

func TestHeaderIsOpenTofuExternalProviderHeader(t *testing.T) {
	data, err := json.Marshal(Header{
		Magic:   "OpenTofu-External-Key-Provider",
		Version: 1,
	})
	if err != nil {
		t.Fatal(err)
	}

	const want = `{"magic":"OpenTofu-External-Key-Provider","version":1}`
	if string(data) != want {
		t.Errorf("header = %s, want %s", data, want)
	}
}
