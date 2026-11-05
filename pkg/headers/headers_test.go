//nolint:tagliatelle, testpackage, goconst, varnamelen
package headers

import (
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
)

func TestHeaderDecoderDecode(t *testing.T) {
	t.Parallel()

	id := uuid.New()

	var dst struct {
		DeviceID uuid.UUID  `header:"device-id"`
		Name     string     `header:"name"`
		Enabled  bool       `header:"enabled"`
		Count    int8       `header:"count"`
		Size     uint16     `header:"size"`
		Ratio    float64    `header:"ratio"`
		Optional *uuid.UUID `header:"optional"`
		Missing  string     `header:"missing"`
		Ignored  string     `header:"-"`
		Untagged string
	}

	dst.Missing = "unchanged"
	dst.Ignored = "unchanged"
	dst.Untagged = "unchanged"
	headers := nats.Header{
		"device-id": {id.String()},
		"name":      {"first", "second"},
		"enabled":   {"true"},
		"count":     {"-12"},
		"size":      {"42"},
		"ratio":     {"1.5"},
		"optional":  {id.String()},
		"Missing":   {"wrong case"},
		"-":         {"ignored"},
		"Untagged":  {"ignored"},
	}

	if err := NewDecoder(headers).Decode(&dst); err != nil {
		t.Fatal(err)
	}

	if dst.DeviceID != id || dst.Optional == nil || *dst.Optional != id {
		t.Fatalf("unexpected UUIDs: %v, %v", dst.DeviceID, dst.Optional)
	}

	if dst.Name != "first" || !dst.Enabled || dst.Count != -12 ||
		dst.Size != 42 ||
		dst.Ratio != 1.5 {
		t.Fatalf("unexpected scalar values: %+v", dst)
	}

	if dst.Missing != "unchanged" || dst.Ignored != "unchanged" ||
		dst.Untagged != "unchanged" {
		t.Fatalf("unexpected changes to ignored or missing fields: %+v", dst)
	}
}

//nolint:exhaustruct_v5
func TestHeaderDecoderErrors(t *testing.T) {
	t.Parallel()

	decoder := NewDecoder(nil)

	var nilStruct *struct{}
	for _, dst := range []any{nil, struct{}{}, nilStruct, new(string)} {
		if err := decoder.Decode(dst); err == nil {
			t.Errorf("expected error for destination %T", dst)
		}
	}

	tests := []struct {
		name string
		dst  any
		val  string
	}{
		{"invalid UUID", &struct {
			Value uuid.UUID `header:"value"`
		}{}, "invalid"},
		{"invalid UUID pointer", &struct {
			Value *uuid.UUID `header:"value"`
		}{}, "invalid"},
		{"invalid integer", &struct {
			Value int8 `header:"value"`
		}{}, "128"},
		{"invalid pointer", &struct {
			Value *int `header:"value"`
		}{}, "invalid"},
		{"unsupported type", &struct {
			Value []string `header:"value"`
		}{}, "value"},
		{"unexported field", &struct {
			value string `header:"value"`
		}{}, "value"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := NewDecoder(
				nats.Header{"value": []string{tt.val}},
			).Decode(tt.dst)
			if err == nil || !strings.Contains(err.Error(), `header "value"`) {
				t.Fatalf("expected header error, got %v", err)
			}
		})
	}
}

type uppercaseHeader string

func (v *uppercaseHeader) UnmarshalText(data []byte) error {
	*v = uppercaseHeader(strings.ToUpper(string(data)))

	return nil
}

func TestHeaderDecoderUsesTextUnmarshaler(t *testing.T) {
	t.Parallel()

	var dst struct {
		Value uppercaseHeader `header:"value"`
	}
	if err := NewDecoder(nats.Header{"value": {"hello"}}).
		Decode(
			&dst,
		); err != nil {
		t.Fatal(err)
	}

	if dst.Value != "HELLO" {
		t.Fatalf("got %q, want HELLO", dst.Value)
	}
}

func TestHeaderDecoderEmptyAndMissingValues(t *testing.T) {
	t.Parallel()

	dst := struct {
		Empty   string `header:"empty"`
		Missing string `header:"missing"`
	}{Empty: "old", Missing: "old"}
	if err := NewDecoder(
		nats.Header{"empty": {""}, "missing": {}},
	).Decode(&dst); err != nil {
		t.Fatal(err)
	}

	if dst.Empty != "" || dst.Missing != "old" {
		t.Fatalf("unexpected decoded values: %+v", dst)
	}
}
