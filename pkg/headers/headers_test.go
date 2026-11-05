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
		Nested   struct {
			NestedID uuid.UUID `header:"nested-id"`
		}
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
		"nested-id": []string{id.String()},
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

	if dst.Nested.NestedID != id {
		t.Fatalf("unexpected nested UUID: %v", dst.Nested.NestedID)
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

func TestHeaderDecoderNestedStructs(t *testing.T) {
	t.Parallel()

	type inner struct {
		ID uuid.UUID `header:"nested-id"`
	}

	type outer struct {
		Inner *inner
	}

	type recursive struct {
		Value string `header:"value"`
		Next  *recursive
	}

	id := uuid.New()
	original := &inner{}
	dst := struct {
		First   outer
		Second  *inner
		Missing *struct {
			Value string `header:"missing"`
		}
		Ignored inner `header:"-"`
		Cycle   *recursive
		Chain   *recursive
	}{Second: original, Cycle: &recursive{}, Chain: &recursive{Next: &recursive{}}}
	dst.Cycle.Next = dst.Cycle

	err := NewDecoder(nats.Header{
		"nested-id": {id.String()},
		"value":     {"decoded"},
	}).Decode(&dst)
	if err != nil {
		t.Fatal(err)
	}

	if dst.First.Inner == nil || dst.First.Inner.ID != id ||
		dst.Second != original ||
		dst.Second.ID != id {
		t.Fatalf("nested fields not decoded correctly: %+v", dst)
	}

	if dst.Missing != nil || dst.Ignored.ID != uuid.Nil {
		t.Fatalf("missing or ignored nested field changed: %+v", dst)
	}

	if dst.Cycle.Value != "decoded" || dst.Cycle.Next != dst.Cycle {
		t.Fatalf("recursive field not decoded correctly: %+v", dst.Cycle)
	}

	if dst.Chain.Value != "decoded" || dst.Chain.Next.Value != "decoded" ||
		dst.Chain.Next.Next != nil {
		t.Fatalf("recursive chain not decoded correctly: %+v", dst.Chain)
	}
}

func TestHeaderDecoderNestedError(t *testing.T) {
	t.Parallel()

	var dst struct {
		Nested *struct {
			ID uuid.UUID `header:"nested-id"`
		}
	}

	err := NewDecoder(nats.Header{"nested-id": {"invalid"}}).Decode(&dst)
	if err == nil || !strings.Contains(err.Error(), `header "nested-id"`) ||
		!strings.Contains(err.Error(), "field Nested") {
		t.Fatalf("expected nested header error, got %v", err)
	}

	if dst.Nested != nil {
		t.Fatal("allocated nested pointer after failed decode")
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
