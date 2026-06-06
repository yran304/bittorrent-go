package main

import (
	"reflect"
	"testing"
)

func TestDecodeBencode(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    interface{}
		wantErr bool
	}{
		{
			name:  "string",
			input: "5:hello",
			want:  "hello",
		},
		{
			name:  "integer",
			input: "i52e",
			want:  52,
		},
		{
			name:  "flat list",
			input: "l5:helloi52ee",
			want:  []interface{}{"hello", 52},
		},
		{
			name:  "nested list",
			input: "ll5:helloi52eee",
			want:  []interface{}{[]interface{}{"hello", 52}},
		},
		{
			name:  "mixed nested list",
			input: "l4:spaml1:a1:bee",
			want:  []interface{}{"spam", []interface{}{"a", "b"}},
		},
		{
			name:    "trailing data",
			input:   "5:hellojunk",
			wantErr: true,
		},
		{
			name:    "unterminated list",
			input:   "l5:helloi52e",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := decodeBencode(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error for %q, got nil", tt.input)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error for %q: %v", tt.input, err)
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("decodeBencode(%q) = %#v, want %#v", tt.input, got, tt.want)
			}
		})
	}
}

func TestDecodeValueConsumedBytes(t *testing.T) {
	got, consumed, err := decodeValue("5:helloi52e")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got != "hello" {
		t.Fatalf("got value %#v, want %q", got, "hello")
	}

	if consumed != 7 {
		t.Fatalf("consumed = %d, want 7", consumed)
	}
}
