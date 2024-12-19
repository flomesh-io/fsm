package urlenc

import (
	"errors"
	"math"
	"net/url"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"sync"
)

type structField struct {
	// FieldName is the name of the field, so we can use FieldByName to
	// index into the struct. This is better (albeit less efficient) than
	// using numeric indices, because then we can work directly with
	// embedded structs
	FieldName string
	// KeyName is the name that is used in the resulting query for this field
	KeyName string
	// If true, the field is not included in the query if its value is
	// equal to the zero value of the field type
	OmitEmpty bool
	// Type is the type of this struct field
	Type reflect.Type
}

var t2f = &type2fields{
	types: make(map[reflect.Type][]structField),
	lock:  new(sync.RWMutex),
}

type type2fields struct {
	lock  *sync.RWMutex
	types map[reflect.Type][]structField
}

func isStringOrNumeric(rk reflect.Kind) bool {
	switch rk {
	case reflect.String, reflect.Bool:
		return true
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Float32, reflect.Float64:
		return true
	}
	return false
}

func isSupportedType(rt reflect.Type, recurse bool) bool {
	switch rk := rt.Kind(); rk {
	case reflect.Slice, reflect.Array:
		if !recurse {
			return false
		}
		ok := isSupportedType(rt.Elem(), false)
		if !ok {
			return false
		}
		return true
	default:
		return isStringOrNumeric(rk)
	}
}

type Valuer interface {
	Value() interface{}
}

var valuerIf = reflect.TypeOf((*Valuer)(nil)).Elem()

func getValuerMethod(fv reflect.Value) reflect.Value {
	const methodName = "Value"
	var mv reflect.Value
	if fv.Type().Implements(valuerIf) {
		mv = fv.MethodByName(methodName)
	} else if fv.CanAddr() && fv.Addr().Type().Implements(valuerIf) {
		mv = fv.Addr().MethodByName(methodName)
	}
	return mv
}

func convertToString(rv reflect.Value) (string, error) {
	switch rv.Kind() {
	case reflect.Bool:
		if rv.Bool() {
			return "true", nil
		}
		return "false", nil
	case reflect.String:
		return rv.String(), nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return strconv.FormatInt(rv.Int(), 10), nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return strconv.FormatUint(rv.Uint(), 10), nil
	case reflect.Float32, reflect.Float64:
		return strconv.FormatFloat(rv.Float(), 'f', -1, 64), nil
	}
	return "", errors.New("urlenc: unsupported type to convert: " + rv.Type().String())
}

func convertFromString(k reflect.Kind, v string) (reflect.Value, error) {
	if convFunc, exists := _kindConvFuncs[k]; exists {
		return convFunc(v)
	} else {
		return zeroval, errors.New("unsupported type")
	}
}

type kindConverter func(v string) (reflect.Value, error)

var _nameToType map[string]reflect.Type
var _kindConvFuncs map[reflect.Kind]kindConverter

func init() {
	_nameToType = map[string]reflect.Type{
		"string":  reflect.TypeOf(""),
		"int":     reflect.TypeOf(int(0)),
		"int8":    reflect.TypeOf(int8(0)),
		"int16":   reflect.TypeOf(int16(0)),
		"int32":   reflect.TypeOf(int32(0)),
		"int64":   reflect.TypeOf(int64(0)),
		"uint":    reflect.TypeOf(uint(0)),
		"uint8":   reflect.TypeOf(uint8(0)),
		"uint16":  reflect.TypeOf(uint16(0)),
		"uint32":  reflect.TypeOf(uint32(0)),
		"uint64":  reflect.TypeOf(uint64(0)),
		"float32": reflect.TypeOf(float32(0)),
		"float64": reflect.TypeOf(float64(0)),
	}

	_kindConvFuncs = map[reflect.Kind]kindConverter{
		reflect.Bool: func(v string) (reflect.Value, error) {
			if bv, err := strconv.ParseBool(v); err != nil {
				return zeroval, err
			} else {
				return reflect.ValueOf(bv), nil
			}
		},
		reflect.String: func(v string) (reflect.Value, error) {
			return reflect.ValueOf(v), nil
		},
		reflect.Int: func(v string) (reflect.Value, error) {
			if nv, err := strconv.ParseInt(v, 10, 64); err != nil {
				return zeroval, err
			} else if nv >= math.MinInt && nv <= math.MaxInt {
				return reflect.ValueOf(int(nv)), nil
			} else {
				return reflect.ValueOf(int(0)), nil
			}
		},
		reflect.Int64: func(v string) (reflect.Value, error) {
			if nv, err := strconv.ParseInt(v, 10, 64); err != nil {
				return zeroval, err
			} else {
				return reflect.ValueOf(nv), nil
			}
		},
		reflect.Int8: func(v string) (reflect.Value, error) {
			if nv, err := strconv.ParseInt(v, 10, 8); err != nil {
				return zeroval, err
			} else {
				return reflect.ValueOf(int8(nv)), nil
			}
		},
		reflect.Int16: func(v string) (reflect.Value, error) {
			if nv, err := strconv.ParseInt(v, 10, 16); err != nil {
				return zeroval, err
			} else {
				return reflect.ValueOf(int16(nv)), nil
			}
		},
		reflect.Int32: func(v string) (reflect.Value, error) {
			if nv, err := strconv.ParseInt(v, 10, 32); err != nil {
				return zeroval, err
			} else {
				return reflect.ValueOf(int32(nv)), nil
			}
		},
		reflect.Uint: func(v string) (reflect.Value, error) {
			if nv, err := strconv.ParseUint(v, 10, 64); err != nil {
				return zeroval, err
			} else if nv <= math.MaxUint {
				return reflect.ValueOf(uint(nv)), nil
			} else {
				return reflect.ValueOf(uint(0)), nil
			}
		},
		reflect.Uint64: func(v string) (reflect.Value, error) {
			if nv, err := strconv.ParseUint(v, 10, 64); err != nil {
				return zeroval, err
			} else {
				return reflect.ValueOf(nv), nil
			}
		},
		reflect.Uint8: func(v string) (reflect.Value, error) {
			if nv, err := strconv.ParseUint(v, 10, 8); err != nil {
				return zeroval, err
			} else {
				return reflect.ValueOf(uint8(nv)), nil
			}
		},
		reflect.Uint16: func(v string) (reflect.Value, error) {
			if nv, err := strconv.ParseUint(v, 10, 16); err != nil {
				return zeroval, err
			} else {
				return reflect.ValueOf(uint16(nv)), nil
			}
		},
		reflect.Uint32: func(v string) (reflect.Value, error) {
			if nv, err := strconv.ParseUint(v, 10, 32); err != nil {
				return zeroval, err
			} else {
				return reflect.ValueOf(uint32(nv)), nil
			}
		},
		reflect.Float64: func(v string) (reflect.Value, error) {
			if nv, err := strconv.ParseFloat(v, 64); err != nil {
				return zeroval, err
			} else {
				return reflect.ValueOf(float64(nv)), nil
			}
		},
		reflect.Float32: func(v string) (reflect.Value, error) {
			if nv, err := strconv.ParseFloat(v, 32); err != nil {
				return zeroval, err
			} else {
				return reflect.ValueOf(float32(nv)), nil
			}
		},
	}
}

func nameToType(s string, recurse bool) reflect.Type {
	if strings.HasPrefix(s, "[]") {
		t := nameToType(s[2:], false)
		if t == nil {
			return nil
		}
		return reflect.SliceOf(t)
	}
	if t, ok := _nameToType[s]; ok {
		return t
	}
	return nil
}

var wssplitRx = regexp.MustCompile(`\s+`)

func (tkm type2fields) getStructFields(t reflect.Type) ([]structField, error) {
	if t.Kind() != reflect.Struct {
		return nil, errors.New("target is not a struct (Kind: " + t.Kind().String() + ")")
	}

	tkm.lock.RLock()

	km, ok := tkm.types[t]
	if ok {
		tkm.lock.RUnlock()
		return km, nil
	}

	// the fields did not exist in the registry. create and register
	km = make([]structField, 0, t.NumField())
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if len(f.PkgPath) > 0 {
			// If PkgPath is non empty, then it's an unexported field
			continue
		}

		var keyname string
		var omitempty bool
		fieldtype := f.Type
		if len(f.Tag) == 0 {
			// no tag at all. Use the name of the field as-is
			keyname = f.Name
		} else {
			// This is silly, but reflect.StructTag.Get cannot differentiate between
			// an empty struct tag with a non-existent struct tag. This is what we
			// like to do:
			// 1) "urlenc" exists, and is non-empty: parse it, use it as planned
			// 2) "urlenc" exists, and is empty: use field name as-is
			// 3) "json" exists: do the same as 1+2 using its value
			//
			// We do a really half-assed parsing here. reading the reflect docs,
			// the authors expect "name:" where name does not contain spaces...
			// hmm, we could be really smart about it, or we could just handwave it.
			// I handwaved it:

			var tagname string
			possibletags := wssplitRx.Split(string(f.Tag), -1)
		OUTER:
			for _, candidate := range []string{"urlenc", "json"} {
				for _, target := range possibletags {
					if strings.HasPrefix(target, candidate+":") {
						tagname = candidate
						break OUTER
					}
				}
			}

			st := f.Tag.Get(tagname)
			if st == "-" {
				// ignore this field
				continue
			}

			// urlenc:"foo,omitempty,<type>"
			parts := strings.SplitN(st, ",", 3)
			if len(parts) > 2 {
				var err error
				name := strings.TrimSpace(parts[2])
				fieldtype = nameToType(name, false)
				if err != nil {
					tkm.lock.RUnlock()
					return nil, errors.New("urlenc: unsupported type from struct tag: '" + name + "'")
				}
			}

			if len(parts) > 1 {
				if strings.TrimSpace(parts[1]) == "omitempty" {
					omitempty = true
				}
			}
			keyname = strings.TrimSpace(parts[0])
		}

		// strings, numbers, and slices of those two are allowed
		if ok := isSupportedType(fieldtype, true); !ok {
			tkm.lock.RUnlock()
			return nil, errors.New("urlenc: unsupported type on struct field " + f.Name + ": " + f.Type.String())
		}

		sf := structField{
			FieldName: f.Name,
			KeyName:   keyname,
			OmitEmpty: omitempty,
			Type:      fieldtype,
		}
		km = append(km, sf)
	}

	tkm.lock.RUnlock()
	tkm.lock.Lock()
	tkm.types[t] = km
	tkm.lock.Unlock()
	return km, nil
}

type Marshaler interface {
	MarshalURL() ([]byte, error)
}

type Unmarshaler interface {
	UnmarshalURL([]byte) error
}

// Encode encodes the given value into a query string. Only structs and maps
// with string keys and several types of types as values are supported.
func Encode(v interface{}) ([]byte, error) {
	if u, ok := v.(Marshaler); ok {
		return u.MarshalURL()
	}

	rv := reflect.ValueOf(v)
	if rv == zeroval {
		return nil, errors.New("can not unmarshal into a nil value")
	}

	// This better be a pointer
	switch rv.Kind() {
	case reflect.Ptr, reflect.Interface:
		// Get the value beyond the pointer
		rv = rv.Elem()
	}

	switch rv.Kind() {
	case reflect.Map:
		if kk := rv.Type().Key().Kind(); kk != reflect.String {
			return nil, errors.New("urlenc.Marshal: map key must be string type (Kind: " + kk.String() + ")")
		}
		return marshalMap(rv)
	case reflect.Struct:
		return marshalStruct(rv)
	default:
		return nil, errors.New("urlenc.Marshal: unsupported type (" + rv.Type().String() + ")")
	}
}

func addValue(uv *url.Values, name string, fv reflect.Value, ft reflect.Type) error {
	if mv := getValuerMethod(fv); mv != zeroval {
		out := mv.Call(nil)
		fv = out[0]
		switch fv.Kind() {
		case reflect.Ptr, reflect.Interface:
			fv = fv.Elem()
		}
	}

	if isStringOrNumeric(ft.Kind()) {
		s, err := convertToString(fv)
		if err != nil {
			return err
		}
		uv.Add(name, s)
	} else {
		for i := 0; i < fv.Len(); i++ {
			ev := fv.Index(i)
			s, err := convertToString(ev)
			if err != nil {
				return err
			}
			uv.Add(name, s)
		}
	}
	return nil
}

func marshalMap(rv reflect.Value) ([]byte, error) {
	if rv.Kind() != reflect.Map {
		return nil, errors.New("target is not a map (Kind: " + rv.Kind().String() + ")")
	}

	uv := url.Values{}
	for _, key := range rv.MapKeys() {
		fv := rv.MapIndex(key)
		switch fv.Kind() {
		case reflect.Ptr, reflect.Interface:
			fv = fv.Elem()
		}

		if ok := isSupportedType(fv.Type(), true); !ok {
			return nil, errors.New("urlenc: unsupported type on map element " + key.String() + " (" + fv.Type().String() + ")")
		}

		if err := addValue(&uv, key.String(), fv, fv.Type()); err != nil {
			return nil, err
		}
	}
	return []byte(uv.Encode()), nil
}

func marshalStruct(rv reflect.Value) ([]byte, error) {
	fields, err := t2f.getStructFields(rv.Type())
	if err != nil {
		return nil, err
	}

	uv := url.Values{}
	for _, f := range fields {
		fv := rv.FieldByName(f.FieldName)

		// Check for empty values
		if f.OmitEmpty {
			if !fv.IsValid() {
				continue
			}

			switch ft := fv.Type(); ft.Kind() {
			case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Ptr, reflect.Slice:
				if fv.IsNil() {
					continue
				}
			case reflect.Struct:
				if fv.Type().Comparable() {
					if fv.Interface() == reflect.Zero(ft).Interface() {
						continue
					}
				}
				if reflect.DeepEqual(fv.Interface(), reflect.Zero(ft).Interface()) {
					continue
				}
			default:
				switch {
				case fv == zeroval:
					continue
				case fv.CanInterface() && fv.Interface() == reflect.Zero(ft).Interface():
					continue
				}
			}
		}

		if err := addValue(&uv, f.KeyName, fv, f.Type); err != nil {
			return nil, err
		}
	}
	return []byte(uv.Encode()), nil
}

var zeroval = reflect.Value{}

func Decode(data []byte, v interface{}) ([]byte, error) {
	if u, ok := v.(Unmarshaler); ok {
		return nil, u.UnmarshalURL(data)
	}

	rv := reflect.ValueOf(v)
	if rv == zeroval {
		return nil, errors.New("can not unmarshal into a nil value")
	}

	// This better be a pointer
	if rv.Kind() != reflect.Ptr {
		return nil, errors.New("pointer value required")
	}

	// Get the value beyond the pointer
	rv = rv.Elem()

	switch rv.Kind() {
	case reflect.Map:
		if kk := rv.Type().Key().Kind(); kk != reflect.String {
			return nil, errors.New("urlenc.Unmarshal: map key must be string type (Kind: " + kk.String() + ")")
		}
		return nil, unmarshalMap(data, rv)
	case reflect.Struct:
		return unmarshalStruct(data, rv)
	default:
		return nil, errors.New("urlenc.Unmarshal: unsupported type (Kind: " + rv.Kind().String() + ")")
	}
}

func unmarshalMap(data []byte, rv reflect.Value) error {
	q, err := url.ParseQuery(string(data))
	if err != nil {
		return err
	}

	for k, v := range q {
		if len(v) == 1 {
			rv.SetMapIndex(reflect.ValueOf(k), reflect.ValueOf(v[0]))
		} else {
			rv.SetMapIndex(reflect.ValueOf(k), reflect.ValueOf(v))
		}
	}

	return nil
}

type Setter interface {
	Set(interface{}) error
}

var setterIf = reflect.TypeOf((*Setter)(nil)).Elem()

func getSetterMethod(fv reflect.Value) reflect.Value {
	const methodName = "Set"
	var mv reflect.Value
	if fv.Type().Implements(setterIf) {
		mv = fv.MethodByName(methodName)
	} else if fv.CanAddr() && fv.Addr().Type().Implements(setterIf) {
		mv = fv.Addr().MethodByName(methodName)
	}
	return mv
}

func unmarshalStruct(data []byte, rv reflect.Value) ([]byte, error) {
	// Grab the mapping from struct tags
	fields, err := t2f.getStructFields(rv.Type())
	if err != nil {
		return nil, err
	}
	q, err := url.ParseQuery(string(data))
	if err != nil {
		return nil, err
	}

	for _, f := range fields {
		values, exists := q[f.KeyName]
		if exists {
			delete(q, f.KeyName)
		}
		if len(values) <= 0 {
			continue
		}

		fv := rv.FieldByName(f.FieldName)
		switch fv.Kind() {
		case reflect.Ptr, reflect.Interface:
			fv = fv.Elem()
		}

		var err error
		var sv reflect.Value // value to be set
		switch rk := f.Type.Kind(); rk {
		case reflect.Slice, reflect.Array:
			et := f.Type.Elem() // slice/array element type
			ek := et.Kind()     // slice/array element kind
			sv = reflect.MakeSlice(reflect.SliceOf(et), len(values), len(values))
			for i := 0; i < len(values); i++ {
				ev := sv.Index(i)
				if cv, err := convertFromString(ek, values[i]); err != nil {
					return nil, err
				} else {
					ev.Set(cv)
				}
			}
		default:
			// This is checking for the REGISTERED type, not the actual type of the field
			if !isStringOrNumeric(rk) {
				return nil, errors.New("urlenc.Unmarshal: unsupported type for field " + f.FieldName + " (Kind: " + rk.String() + ")")
			}

			// Now convert the value
			if sv, err = convertFromString(f.Type.Kind(), values[0]); err != nil {
				return nil, err
			}
		}

		// See if our value can Set()
		mv := getSetterMethod(fv)
		if mv == zeroval {
			// No set. Try doing it the orthodox way
			fv.Set(sv)
		} else {
			out := mv.Call([]reflect.Value{sv})
			if !out[0].IsNil() {
				return nil, out[0].Interface().(error)
			}
		}
	}

	if len(q) > 0 {
		return []byte(q.Encode()), nil
	}
	return nil, nil
}
