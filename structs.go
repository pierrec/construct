package iniconfig

import (
	"encoding"
	"encoding/csv"
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/kr/pretty"
)

var errNoStruct = fmt.Errorf("not a struct")
var errNoPointer = fmt.Errorf("not a pointer")
var errCannotUnmarshal = fmt.Errorf("cannot unmarshal value")

func newStructs(s interface{}) (*structs, error) {
	v := reflect.ValueOf(s)
	if v.Kind() != reflect.Ptr {
		return nil, errNoPointer
	}
	if v.Elem().Kind() != reflect.Struct {
		return nil, errNoStruct
	}

	return &structs{
		raw:   s,
		value: v,
		data:  fieldsOf(s),
	}, nil
}

type structfield struct {
	field    *reflect.StructField
	value    reflect.Value
	embedded *structs
}

// Set assigns the given value to the field.
// If the value is a string but the field is not,
// then its value is deserialized using encoding.Unmarshaler
// or in a best effort way.
func (f *structfield) Set(v interface{}) error {
	s, ok := v.(string)
	if !ok || f.value.Kind() == reflect.String {
		value := reflect.ValueOf(v)
		f.value.Set(value)
		return nil
	}

	// v is a string but the field is not one:
	// unmarshaling required
	return unmarshalValue(f.value, s)
}

func unmarshalValue(value reflect.Value, s string) error {
	if dec, ok := ptrValue(value).(encoding.TextUnmarshaler); ok {
		return dec.UnmarshalText([]byte(s))
	}

	switch value.Kind() {
	default:
		return errCannotUnmarshal
	case reflect.Bool:
		v, err := strconv.ParseBool(s)
		if err != nil {
			return err
		}
		value.SetBool(v)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		v, err := strconv.ParseInt(s, 0, 64)
		if err != nil {
			return err
		}
		value.SetInt(v)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		v, err := strconv.ParseUint(s, 0, 64)
		if err != nil {
			return err
		}
		value.SetUint(v)
	case reflect.Float32, reflect.Float64:
		v, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return err
		}
		value.SetFloat(v)
	case reflect.String:
		value.SetString(s)
	case reflect.Array:
		values, err := getStringSlice(s)
		if err != nil {
			return err
		}
		for i, s := range values {
			v := value.Index(i).Elem()
			if err := unmarshalValue(v, s); err != nil {
				return err
			}
		}
	//TODO case reflect.Map:
	case reflect.Slice:
		values, err := getStringSlice(s)
		if err != nil {
			return err
		}
		elem := value.Type().Elem()
		sliceValues := reflect.MakeSlice(value.Type(), 0, len(values))
		for _, s := range values {
			v := reflect.New(elem).Elem()
			if err := unmarshalValue(v, s); err != nil {
				return err
			}
			sliceValues = reflect.Append(sliceValues, v)
		}
		value.Set(sliceValues)
	}
	return nil
}

func (f *structfield) Value() interface{} {
	return f.value.Interface()
}

type structs struct {
	raw   interface{}
	value reflect.Value
	data  []*structfield
}

func (s *structs) GoString() string {
	return pretty.Sprint(s)
}

// Get returns the struct for the corresponding path.
func (s *structs) Get(path ...string) *structfield {
	name := path[0]
	if len(path) == 1 {
		for _, item := range s.data {
			if item.embedded == nil && item.field.Name == name {
				return item
			}
		}
		return nil
	}
	for _, item := range s.data {
		if item.embedded != nil && item.field.Name == name {
			return item.embedded.Get(path[1:]...)
		}
	}
	return nil
}

func (s *structs) Fields() []*structfield {
	return s.data
}

// CallFirst recursively calls the given method on its structs and stops
// at the first one satisfying the stop condition.
func (s *structs) CallFirst(m string, args []interface{}, stop func([]interface{}) bool) ([]interface{}, bool) {
	res, ok := s.Call(m, args)
	if ok && stop(res) {
		return res, true
	}
	for _, item := range s.data {
		if item.embedded == nil {
			continue
		}
		res, ok := item.embedded.CallFirst(m, args, stop)
		if ok && stop(res) {
			return res, true
		}
	}
	return nil, false
}

// Call invokes the method m on s with arguments args.
//
// It returns the method results and whether is was invoked successfully.
func (s *structs) Call(m string, args []interface{}) ([]interface{}, bool) {
	fn := s.value.MethodByName(m)
	if !fn.IsValid() {
		return nil, false
	}
	values := make([]reflect.Value, len(args))
	for i, arg := range args {
		values[i] = reflect.ValueOf(arg)
	}
	rvalues := fn.Call(values)
	results := make([]interface{}, len(rvalues))
	for i, rv := range rvalues {
		results[i] = rv.Interface()
	}
	return results, true
}

// List the fields of the input which must be a pointer to a struct.
func fieldsOf(v interface{}) (res []*structfield) {
	value := reflect.ValueOf(v).Elem()
	vType := value.Type()
	for i, n := 0, value.NumField(); i < n; i++ {
		value := value.Field(i)
		if !value.CanSet() {
			// Cannot set the field, maybe unexported.
			continue
		}
		field := vType.Field(i)

		if tag := field.Tag.Get("cfg"); tag != "" {
			if tag == "-" {
				continue
			}
		}

		if !field.Anonymous {
			// Non embedded field.
			res = append(res, &structfield{&field, value, nil})
			continue
		}

		// Embedded field: recursively descend into its fields.
		if value.Kind() != reflect.Ptr {
			value = value.Addr()
		}
		v := value.Interface()
		fs := &structs{v, value, fieldsOf(v)}
		res = append(res, &structfield{&field, value, fs})
	}
	return
}

// ptrValue returns the interface of the pointer value.
func ptrValue(value reflect.Value) interface{} {
	if value.Kind() != reflect.Ptr {
		value = value.Addr()
	}
	return value.Interface()
}

// getStringSlice converts the csv input string into a slice.
func getStringSlice(s string) ([]string, error) {
	if s == "" {
		return nil, nil
	}
	buf := strings.NewReader(s)
	r := csv.NewReader(buf)
	r.Comma = SliceSeparator
	return r.Read()
}
