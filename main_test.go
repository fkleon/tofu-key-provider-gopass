package main

import (
	"encoding/json"
	"reflect"
	"testing"
)

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
