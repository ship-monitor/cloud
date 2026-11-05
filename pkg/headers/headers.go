package headers

import (
	"encoding"
	"errors"
	"fmt"
	"reflect"
	"strconv"
)

var (
	ErrUnsupportedFieldType = errors.New("unsupported field type")
	ErrInvalidDest          = errors.New(
		"header destination must be a non-nil pointer to a struct",
	)
	ErrCantSet = errors.New("field can't be set")
)

type Header interface {
	Get(key string) string
	Values(key string) []string
}

// Decoder decodes NATS headers into fields tagged with header.
// Header names are case-sensitive, as in nats.Header. Missing headers leave
// their destination fields unchanged; validation of required fields is
// separate.
type Decoder struct {
	header Header
}

func NewDecoder(header Header) *Decoder {
	return &Decoder{header: header}
}

// Decode reads the first value of each tagged header into dst, which must be a
// non-nil pointer to a struct. Untagged fields and fields tagged "-" are
// ignored.
// Fields may be strings, booleans, numbers, pointers to supported types, or
// types implementing encoding.TextUnmarshaler.
func (d *Decoder) Decode(dst any) error {
	reflValue := reflect.ValueOf(dst)
	if !reflValue.IsValid() || reflValue.Kind() != reflect.Pointer ||
		reflValue.IsNil() ||
		reflValue.Elem().Kind() != reflect.Struct {
		return fmt.Errorf("kind %v: %w", reflValue.Kind(), ErrInvalidDest)
	}

	reflValue = reflValue.Elem()

	t := reflValue.Type()
	for fieldIndex := range reflValue.NumField() {
		fieldType := t.Field(fieldIndex)

		name := fieldType.Tag.Get("header")
		if name == "" || name == "-" {
			continue
		}

		field := reflValue.Field(fieldIndex)
		if !field.CanSet() {
			return fmt.Errorf(
				"header %q: field %s: %w",
				name,
				fieldType.Name,
				ErrCantSet,
			)
		}

		values := d.header.Values(name)
		if len(values) == 0 {
			continue
		}

		if err := decodeHeaderValue(field, values[0]); err != nil {
			return fmt.Errorf("header %q: %w", name, err)
		}
	}

	return nil
}

func decodeHeaderValue(field reflect.Value, value string) error {
	if field.Kind() == reflect.Pointer {
		decoded := reflect.New(field.Type().Elem())
		if err := decodeHeaderValue(decoded.Elem(), value); err != nil {
			return fmt.Errorf("decode value: %w", err)
		}

		field.Set(decoded)

		return nil
	}

	if field.CanAddr() {
		if unmarshaler, ok := reflect.TypeAssert[encoding.TextUnmarshaler](
			field.Addr(),
		); ok {
			if err := unmarshaler.UnmarshalText([]byte(value)); err != nil {
				return fmt.Errorf("unmarshall text: %w", err)
			}

			return nil
		}
	}

	//nolint:exhaustive
	switch field.Kind() {
	case reflect.String:
		field.SetString(value)
	case reflect.Bool:
		v, err := strconv.ParseBool(value)
		if err != nil {
			return fmt.Errorf("parse bool: %w", err)
		}

		field.SetBool(v)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		v, err := strconv.ParseInt(value, 10, field.Type().Bits())
		if err != nil {
			return fmt.Errorf("parse int: %w", err)
		}

		field.SetInt(v)
	case reflect.Uint,
		reflect.Uint8,
		reflect.Uint16,
		reflect.Uint32,
		reflect.Uint64:
		v, err := strconv.ParseUint(value, 10, field.Type().Bits())
		if err != nil {
			return fmt.Errorf("parse uint: %w", err)
		}

		field.SetUint(v)
	case reflect.Float32, reflect.Float64:
		v, err := strconv.ParseFloat(value, field.Type().Bits())
		if err != nil {
			return fmt.Errorf("parse float64: %w", err)
		}

		field.SetFloat(v)
	default:
		return fmt.Errorf(
			"field type %s: %w",
			field.Type(),
			ErrUnsupportedFieldType,
		)
	}

	return nil
}
