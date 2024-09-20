package getter

import (
	"fmt"
	"net/http"
	"reflect"
	"strconv"

	"github.com/ettle/strcase"
)

// IntoStructTyped returns a function that sets the fields of the given struct type to the URL values of the request via reflection.
func IntoStructTyped(t reflect.Type) (func(r *http.Request, v any) error, error) {
	if t.Kind() != reflect.Ptr {
		return nil, fmt.Errorf("expected pointer, got %v", t)
	}
	t = t.Elem()
	if t.Kind() != reflect.Struct {
		return nil, fmt.Errorf("expected pointer to struct, got %v", t)
	}
	sets := make([]func(r *http.Request) (reflect.Value, error), t.NumField())
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if field.Anonymous {
			continue
		}

		set, err := fieldSetter(field)
		if err != nil {
			return nil, fmt.Errorf("field %s: %w", field.Name, err)
		}
		sets[i] = set
	}
	return func(r *http.Request, v any) error {
		value := reflect.ValueOf(v).Elem()
		for i, set := range sets {
			if set == nil {
				continue
			}
			v, err := set(r)
			if err != nil {
				return fmt.Errorf("field %s: %w", value.Type().Field(i).Name, err)
			}
			if !v.IsValid() {
				continue
			}
			value.Field(i).Set(v)
		}
		return nil
	}, nil
}

func fieldSetter(field reflect.StructField) (func(r *http.Request) (reflect.Value, error), error) {
	name := field.Tag.Get("getter")
	if name == "" {
		name = strcase.ToKebab(field.Name)
	}
	valueParser, err := valuesParser(field.Type)
	if err != nil {
		return nil, err
	}
	return func(r *http.Request) (reflect.Value, error) {
		values := r.URL.Query()[name]
		v, err := valueParser(values)
		if err != nil {
			return reflect.Value{}, err
		}
		return v, nil
	}, nil
}

// IntoStruct uses reflection to set the fields of the given struct to the URL values of the request.
func IntoStruct(r *http.Request, v any) error {
	parse, err := IntoStructTyped(reflect.TypeOf(v))
	if err != nil {
		return err
	}
	return parse(r, v)
}

func valuesParser(t reflect.Type) (func([]string) (reflect.Value, error), error) {
	if t.Kind() == reflect.Pointer {
		parse, err := valueParser(t.Elem())
		if err != nil {
			return nil, err
		}
		return func(values []string) (reflect.Value, error) {
			if len(values) == 0 {
				return reflect.ValueOf(nil), nil
			}
			if len(values) > 1 {
				return reflect.Value{}, fmt.Errorf("expected 1 value, got %d", len(values))
			}
			parsed, err := parse(values[0])
			if err != nil {
				return reflect.Value{}, err
			}
			rValue := reflect.New(t.Elem())
			rValue.Elem().Set(parsed)
			return rValue, nil
		}, nil
	}
	if t.Kind() == reflect.Slice {
		parse, err := valueParser(t.Elem())
		if err != nil {
			return nil, err
		}
		return func(values []string) (reflect.Value, error) {
			slice := reflect.MakeSlice(t, len(values), len(values))
			for i, value := range values {
				parsed, err := parse(value)
				if err != nil {
					return reflect.Value{}, err
				}
				slice.Index(i).Set(parsed)
			}
			return slice, nil
		}, nil
	}
	parse, err := valueParser(t)
	if err != nil {
		return nil, err
	}
	return func(values []string) (reflect.Value, error) {

		if len(values) == 0 {
			return reflect.Value{}, fmt.Errorf("no value")
		}
		if len(values) > 1 {
			return reflect.Value{}, fmt.Errorf("expected 1 value, got %d", len(values))
		}
		return parse(values[0])
	}, nil
}

func valueParser(t reflect.Type) (func(string) (reflect.Value, error), error) {
	switch t.Kind() {
	case reflect.String:
		return func(value string) (reflect.Value, error) {
			return reflect.ValueOf(value), nil
		}, nil
	case reflect.Int:
		return func(value string) (reflect.Value, error) {
			intValue, err := strconv.Atoi(value)
			if err != nil {
				return reflect.Value{}, err
			}
			return reflect.ValueOf(intValue), nil
		}, nil
	case reflect.Bool:
		return func(value string) (reflect.Value, error) {
			boolValue, err := strconv.ParseBool(value)
			if err != nil {
				return reflect.Value{}, err
			}
			return reflect.ValueOf(boolValue), nil
		}, nil
	default:
		return nil, fmt.Errorf("unsupported type %s", t)
	}
}
