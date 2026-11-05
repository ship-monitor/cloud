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
// non-nil pointer to a struct. Untagged nested structs are traversed using the
// same header names; other untagged fields and fields tagged "-" are ignored.
// Nil pointers to nested structs are allocated only when a header is decoded.
// Fields may be strings, booleans, numbers, pointers to supported types, or
// types implementing encoding.TextUnmarshaler.
func (d *Decoder) Decode(dst any) error {
	reflValue := reflect.ValueOf(dst)
	if !reflValue.IsValid() || reflValue.Kind() != reflect.Pointer ||
		reflValue.IsNil() ||
		reflValue.Elem().Kind() != reflect.Struct {
		return fmt.Errorf("kind %v: %w", reflValue.Kind(), ErrInvalidDest)
	}

	_, err := d.decodeStruct(reflValue.Elem(), &decodePath{
		values: make(map[reflect.Value]bool),
		types:  make(map[reflect.Type]int),
	})

	return err
}

type decodePath struct {
	values map[reflect.Value]bool
	types  map[reflect.Type]int
}

func (d *Decoder) decodeStruct(
	value reflect.Value,
	active *decodePath,
) (bool, error) {
	t := value.Type()

	ptr := value.Addr()
	if active.values[ptr] {
		return false, nil
	}

	active.values[ptr] = true

	active.types[t]++
	defer func() {
		delete(active.values, ptr)
		active.types[t]--
	}()

	decoded := false

	for fieldIndex := range value.NumField() {
		fieldType := t.Field(fieldIndex)

		name := fieldType.Tag.Get("header")
		if name == "-" {
			continue
		}

		field := value.Field(fieldIndex)
		if name == "" {
			if !field.CanSet() {
				continue
			}

			nested := field
			if nested.Kind() == reflect.Pointer &&
				nested.Type().Elem().Kind() == reflect.Struct {
				if nested.IsNil() {
					// Avoid infinitely expanding a recursive type through nil
					// pointers.
					if active.types[nested.Type().Elem()] > 0 {
						continue
					}

					nested = reflect.New(nested.Type().Elem())
				}

				nested = nested.Elem()
			}

			if nested.Kind() != reflect.Struct {
				continue
			}

			if _, ok := reflect.TypeAssert[encoding.TextUnmarshaler](
				nested.Addr(),
			); ok {
				continue
			}

			found, err := d.decodeStruct(nested, active)
			if err != nil {
				return false, fmt.Errorf("field %s: %w", fieldType.Name, err)
			}

			if found && field.Kind() == reflect.Pointer && field.IsNil() {
				field.Set(nested.Addr())
			}

			decoded = decoded || found

			continue
		}

		if !field.CanSet() {
			return false, fmt.Errorf(
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
			return false, fmt.Errorf("header %q: %w", name, err)
		}

		decoded = true
	}

	return decoded, nil
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
