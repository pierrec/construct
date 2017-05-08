package structs

import (
	"fmt"
	"reflect"
)

// Set assigns v to the value.
// If v is a string but value is not, then Set attempts to deserialize it
// using UnmarshalValue().
func Set(value reflect.Value, v interface{}, seps []rune) error {
	if !value.CanSet() {
		return errCannotSet
	}

	switch v := v.(type) {
	case nil:
		// Reset the value.
		zero := reflect.Zero(value.Type())
		value.Set(zero)
		return nil
	case string:
		return UnmarshalValue(value, v, seps)
	}

	val := reflect.ValueOf(v)
	if value.Kind() != val.Kind() {
		// The value was converted.
		v, err := convert(val, value)
		if err != nil {
			return err
		}
		val = v
	}
	value.Set(val)
	return nil
}

// convert a to b safely.
func convert(a, b reflect.Value) (_ reflect.Value, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("%v", r)
		}
	}()
	return a.Convert(b.Type()), nil
}

// setFromMap populates value, which must be a pointer to a struct,
// with values corresponding to its fields by name.
func setFromMap(value interface{}, values map[string]interface{}) error {
	fields, err := fieldsOf(value, "", "")
	if err != nil {
		return err
	}
	for _, field := range fields {
		name := field.Name()
		v, ok := values[name]
		if !ok {
			// Field not found in the map.
			continue
		}
		if err := field.Set(v); err != nil {
			return fmt.Errorf("%v: %v", name, err)
		}
	}
	return nil
}
