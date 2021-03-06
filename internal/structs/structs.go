package structs

import (
	"fmt"
	htemplate "html/template"
	"net"
	"net/url"
	"reflect"
	"regexp"
	"strings"
	"text/template"
	"time"

	"github.com/kr/pretty"
	"github.com/pkg/errors"
)

const (
	// SliceSeparator is used to separate slice and map items.
	SliceSeparator = ','

	// MapKeySeparator is used to separate map keys and their value.
	MapKeySeparator = ':'
)

var (
	errNoStruct        = errors.Errorf("not a struct")
	errNoPointer       = errors.Errorf("not a pointer")
	errCannotUnmarshal = errors.Errorf("cannot unmarshal value")
	errInvalidMapKey   = errors.Errorf("invalid map key")
	errCannotSet       = errors.Errorf("cannot set value")
)

// Supported types.
var (
	durationType     = reflect.TypeOf(time.Second)
	timeType         = reflect.TypeOf(time.Time{})
	urlType          = reflect.TypeOf(new(url.URL))
	texttemplateType = reflect.TypeOf(template.New(""))
	htmltemplateType = reflect.TypeOf(htemplate.New(""))
	regexpType       = reflect.TypeOf(regexp.MustCompile("."))
	ipaddrType       = reflect.TypeOf(new(net.IPAddr))
	ipnetType        = reflect.TypeOf(new(net.IPNet))
)

// NewStruct recursively decomposes the input struct into its fields
// and embedded structs.
// Fields tags with "-" will be skipped.
// Fields tags with a non empty value will be renamed to that value.
//
// The input must be a pointer to a struct.
func NewStruct(s interface{}, tagid, septagid string) (*StructStruct, error) {
	if s, ok := s.(*StructStruct); ok {
		return s, nil
	}

	v := reflect.ValueOf(s)
	if v.Kind() != reflect.Ptr {
		return nil, errNoPointer
	}
	if v.Elem().Kind() != reflect.Struct {
		return nil, errNoStruct
	}
	fields, err := fieldsOf(s, tagid, septagid)
	if err != nil {
		return nil, err
	}

	return &StructStruct{
		name:  fmt.Sprintf("%T", s),
		raw:   s,
		value: v,
		data:  fields,
	}, nil
}

// StructField represents a struct field.
type StructField struct {
	name     string
	field    *reflect.StructField
	value    reflect.Value
	tag      reflect.StructTag
	seps     []rune
	embedded *StructStruct
}

// Name returns the field name.
func (f *StructField) Name() string {
	return f.name
}

// Embedded returns the embedded struct if the field is embedded.
func (f *StructField) Embedded() *StructStruct {
	return f.embedded
}

// Set assigns the given value to the field.
// If the value is a string but the field is not,
// then its value is deserialized using encoding.Unmarshaler
// or in a best effort way.
func (f *StructField) Set(v interface{}) error {
	switch v := v.(type) {
	case []interface{}:
		if f.value.Kind() != reflect.Slice {
			return errors.Errorf("%v: cannot assign a slice to a non slice field", f)
		}
		vType := f.value.Type()
		sliceValues := reflect.MakeSlice(vType, len(v), len(v))
		for i, item := range v {
			v := sliceValues.Index(i)
			if !v.CanAddr() {
				v = v.Addr()
			}
			if err := Set(v, item, nil); err != nil {
				return errors.Errorf("%v: %v", f, err)
			}
		}
		f.value.Set(sliceValues)
	case map[string]interface{}:
		if f.value.Kind() != reflect.Struct {
			return errors.Errorf("%v: cannot assign a map to a non struct field", f)
		}
		s := f.value.Addr()
		return setFromMap(s, v)
	case []map[string]interface{}:
		if f.value.Kind() != reflect.Slice {
			return errors.Errorf("%v: cannot assign a slice map to a non slice field", f)
		}
		vType := f.value.Type()
		if vType.Elem().Kind() != reflect.Struct {
			return errors.Errorf("%v: cannot assign a slice map item to a non struct field", f)
		}
		sliceValues := reflect.MakeSlice(vType, len(v), len(v))
		for i, item := range v {
			v := sliceValues.Index(i)
			if !v.CanAddr() {
				v = v.Addr()
			}
			if err := setFromMap(v.Interface(), item); err != nil {
				return errors.Errorf("%v: %v", f, err)
			}
		}
		f.value.Set(sliceValues)
	default:
		return Set(f.value, v, f.seps)
	}
	return nil
}

// Interface returns the interface value of the field.
func (f *StructField) Interface() interface{} {
	return f.value.Interface()
}

// PtrValue returns the interface pointer value of the field.
func (f *StructField) PtrValue() interface{} {
	return f.value.Addr().Interface()
}

// Tag returns the tags defined on the field.
func (f *StructField) Tag() reflect.StructTag {
	return f.tag
}

// Separators returns the field separators.
func (f *StructField) Separators() []rune {
	return f.seps
}

// MarshalValue returns the field value marshaled by MarshalValue().
func (f *StructField) MarshalValue() (interface{}, error) {
	return MarshalValue(f.Interface(), f.seps)
}

// StructStruct represents a decomposed struct.
type StructStruct struct {
	name    string
	raw     interface{}
	inlined bool
	value   reflect.Value
	data    []*StructField
}

// Name returns the underlying type name.
func (s *StructStruct) Name() string {
	return s.name
}

// Inlined returns whether or not the struct is inlined.
func (s *StructStruct) Inlined() bool {
	return s.inlined
}

// GoString is used to debug a StructStruct and returns a full
// and human readable representation of its elements.
func (s *StructStruct) GoString() string {
	return pretty.Sprint(s)
}

// String gives a simple string representation of the StructStruct.
func (s *StructStruct) String() string {
	return s.string(0)
}

// n: field padding
func (s *StructStruct) string(n int) string {
	sname := s.Name()
	pad := strings.Repeat(" ", n)

	var res string
	res += fmt.Sprintf("%s%s {\n", pad, sname)

	var fn int
	for _, field := range s.data {
		var n int
		if emb := field.Embedded(); emb != nil {
			n = len(emb.Name())
		} else {
			n = len(field.Name())
		}
		if n > fn {
			fn = n
		}
	}

	f := fmt.Sprintf("%s%%%ds %%T\n", pad, fn+1)
	for _, field := range s.data {
		if emb := field.Embedded(); emb != nil {
			res += emb.string(n + fn)
			continue
		}
		res += fmt.Sprintf(f, field.Name(), field.value.Interface())
	}

	res += fmt.Sprintf("%s}\n", pad)
	return res
}

// Lookup returns the field for the corresponding path.
func (s *StructStruct) Lookup(path ...string) *StructField {
	name := path[0]
	if len(path) == 1 {
		for _, item := range s.data {
			emb := item.Embedded()
			if emb == nil || !emb.Inlined() {
				if item.Name() == name {
					return item
				}
				continue
			}
			if field := emb.Lookup(name); field != nil {
				return field
			}
		}
		return nil
	}
	for _, item := range s.data {
		emb := item.Embedded()
		if emb == nil {
			continue
		}
		var field *StructField
		if emb.Inlined() {
			field = emb.Lookup(path...)
		} else if item.Name() == name {
			field = emb.Lookup(path[1:]...)
		}
		if field != nil {
			return field
		}
	}
	return nil
}

// Fields returns all the fields of the parsed struct.
func (s *StructStruct) Fields() []*StructField {
	return s.data
}

// Interface returns the raw interface of the underlying struct.
func (s *StructStruct) Interface() interface{} {
	return s.raw
}

// CallUntil recursively calls the given method on its StructStruct fields
// and stops at the first one satisfying the stop condition.
func (s *StructStruct) CallUntil(m string, args []interface{}, until func([]interface{}) bool) ([]interface{}, bool) {
	res, ok := s.Call(m, args)
	if ok && until(res) {
		return res, true
	}
	for _, item := range s.data {
		if item.embedded == nil {
			continue
		}
		res, ok := item.embedded.CallUntil(m, args, until)
		if ok && until(res) {
			return res, true
		}
	}
	return nil, false
}

// Call invokes the method m on s with arguments args.
//
// It returns the method results and whether is was invoked successfully.
func (s *StructStruct) Call(m string, args []interface{}) ([]interface{}, bool) {
	fn := s.value.MethodByName(m)
	if !fn.IsValid() {
		if s.value.CanAddr() {
			// Try with a pointer receiver.
			fn = s.value.Addr().MethodByName(m)
		}
		if !fn.IsValid() {
			return nil, false
		}
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
func fieldsOf(v interface{}, tagid, septagid string) (res []*StructField, err error) {
	value := reflect.ValueOf(v).Elem()
	vType := value.Type()
	for i, n := 0, value.NumField(); i < n; i++ {
		value := value.Field(i)
		if !value.CanSet() {
			// Cannot set the field, maybe unexported.
			continue
		}
		field := vType.Field(i)
		fname := field.Name

		tag := field.Tag
		tagval := tag.Get(tagid)
		tagvalues := strings.Split(tagval, ",")

		// The name is the first item in a coma separated list.
		switch tagvalues[0] {
		case "":
		case "-":
			continue
		default:
			// Set the field name according to the struct tag.
			fname = tagvalues[0]
		}

		// Apply the tag flags.
		var inline bool
		for _, flag := range tagvalues[1:] {
			switch flag {
			case "inline":
				inline = true
			default:
				return nil, errors.Errorf("unkown tag flag %s", flag)
			}
		}

		var fs *StructStruct
		switch kind := value.Kind(); kind {
		case reflect.Invalid,
			reflect.Complex64, reflect.Complex128,
			reflect.Chan, reflect.Func, reflect.Interface,
			reflect.UnsafePointer:
			// Unsupported field types.
			continue
		case reflect.Struct:
			if field.Type.Name() == "" {
				// unnamed type: no methods can be defined, ignore.
				continue
			}

			if field.Anonymous {
				// Embedded field: recursively descend into its fields.
				v := value.Addr().Interface()
				fields, err := fieldsOf(v, tagid, septagid)
				if err != nil {
					return nil, errors.Errorf("%s: %v", fname, err)
				}

				fs = &StructStruct{fname, v, inline, value, fields}
			}
		}
		seps := []rune(tag.Get(septagid))
		res = append(res, &StructField{fname, &field, value, tag, seps, fs})
	}
	return
}
