package urest

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"

	"uw/pkg/cast"
	"uw/pkg/tagparser"
	"uw/ulog"
)

var (
	ErrUnsupportedSliceType = errors.New("unsupported slice type")
	ErrUnsupportedType      = errors.New("unsupported type")
	ErrValueNotSettable     = errors.New("value not settable")
)

type reflectMethodValue struct {
	reflect.Method
	Value reflect.Value
}

func structReflectMethodValue(val any) (value []*reflectMethodValue, e error) {
	v := reflect.ValueOf(val)
	t := v.Type()

	if v.Kind() != reflect.Pointer && v.Kind() != reflect.Struct {
		return nil, ErrValueNotStructOrPointer
	}

	for i := 0; i < v.NumMethod(); i++ {
		vt := v.Method(i)
		mt := t.Method(i)

		if !vt.IsValid() || !vt.CanInterface() {
			continue
		}

		value = append(value, &reflectMethodValue{
			Method: mt,
			Value:  vt,
		})
	}

	return value, nil
}

func DefaultRequest[T any](r *http.Request, input *T) error {
	v := reflect.ValueOf(input).Elem()
	if v.Kind() != reflect.Struct {
		return nil
	}

	return setRequestField(r, v)
}

func setRequestField(r *http.Request, v reflect.Value) error {
	if !v.CanSet() {
		return ErrValueNotSettable
	}

	t := v.Type()

	if r.Form == nil {
		r.ParseForm()
	}

	for i := 0; i < t.NumField(); i++ {
		tag, key := tagparser.Parse(string(t.Field(i).Tag)), ""

		if n, ok := tag["key"]; ok && n != "" {
			key = n
		}
		if n, ok := tag["json"]; ok && n != "" && key == "" {
			key = n
		}

		if n, ok := tag["header"]; ok {
			if vs := r.Header[n]; len(vs) > 0 {
				if e := setReflectValue(r, v.Field(i), vs[0]); e != nil {
					return fmt.Errorf("set reflect header value %s (%s) error: %w", t.Field(i).Name, n, e)
				}

				continue
			}
		}

		if r.Form != nil {
			if vs, ok := r.Form[key]; ok && len(vs) > 0 {
				if e := setReflectValue(r, v.Field(i), vs[0]); e != nil {
					return fmt.Errorf("set reflect form value %s (%s) error: %w", t.Field(i).Name, key, e)
				}

				continue
			}
		}

		if vs, ok := r.URL.Query()[key]; ok && len(vs) > 0 {
			if e := setReflectValue(r, v.Field(i), vs[0]); e != nil {
				return fmt.Errorf("set reflect query value %s (%s) error: %w", t.Field(i).Name, key, e)
			}

			continue
		}

		if val, ok := tag["default"]; ok {
			if e := setReflectValue(r, v.Field(i), val); e != nil {
				return fmt.Errorf("set reflect default value %s error: %w", t.Field(i).Name, e)
			}

			continue
		}

		if v.Field(i).Kind() != reflect.Pointer ||
			t.Field(i).Type.Elem().Kind() != reflect.Struct {
			continue
		}

		if v.Field(i).IsNil() {
			v.Field(i).Set(reflect.New(t.Field(i).Type.Elem()))
		}

		if e := setRequestField(r, v.Field(i).Elem()); e != nil {
			return fmt.Errorf("set reflect field %s error: %w", t.Field(i).Name, e)
		}
	}

	return nil
}

func setReflectValue(r *http.Request, v reflect.Value, val interface{}) error {
	if !v.CanSet() {
		return ErrValueNotSettable
	}

	switch v.Kind() {
	case reflect.String:
		val = cast.ToString(val)
	case reflect.Int64:
		val = cast.ToInt64(val)
	case reflect.Int32:
		val = cast.ToInt32(val)
	case reflect.Int16:
		val = cast.ToInt16(val)
	case reflect.Int8:
		val = cast.ToInt8(val)
	case reflect.Int:
		val = cast.ToInt(val)
	case reflect.Float64:
		val = cast.ToFloat64(val)
	case reflect.Float32:
		val = cast.ToFloat32(val)
	case reflect.Bool:
		val = cast.ToBool(val)
	case reflect.Slice:
		switch v.Type().Elem().Kind() {
		case reflect.String:
			val = cast.ToStringSlice(val)
		case reflect.Int:
			val = cast.ToIntSlice(val)
		case reflect.Bool:
			val = cast.ToBoolSlice(val)
		default:
			return ErrUnsupportedSliceType
		}
	case reflect.Struct:
		return setRequestField(r, v.Elem())
	case reflect.Pointer:
		if v.IsNil() {
			v.Set(reflect.New(v.Type().Elem()))
		}

		return setReflectValue(r, v.Elem(), val)
	default:
		return ErrUnsupportedType
	}

	v.Set(reflect.ValueOf(val))
	return nil
}

func DefaultResponse(w http.ResponseWriter, resp interface{}) {
	switch resp := resp.(type) {
	case string:
		if _, ok := w.Header()["Content-Type"]; !ok {
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		}

		if _, e := w.Write([]byte(resp)); e != nil {
			panic(e)
		}
	case []byte:
		if _, ok := w.Header()["Content-Type"]; !ok {
			w.Header().Set("Content-Type", "application/octet-stream")
		}

		if _, e := w.Write(resp); e != nil {
			panic(e)
		}
	case io.ReadCloser:
		defer resp.Close()

		if _, ok := w.Header()["Content-Type"]; !ok {
			w.Header().Set("Content-Type", "application/octet-stream")
		}

		if _, e := io.Copy(w, resp); e != nil {
			panic(e)
		}
	case io.WriterTo:
		if _, ok := w.Header()["Content-Type"]; !ok {
			w.Header().Set("Content-Type", "application/octet-stream")
		}

		if _, e := resp.WriteTo(w); e != nil {
			panic(e)
		}
	default:
		w.Header().Set("Content-Type", "application/json; charset=utf-8")

		if e := json.NewEncoder(w).Encode(resp); e != nil {
			panic(e)
		}
	}
}

func DefaultRecover(c *Context, e error) {
	c.Writer.WriteHeader(http.StatusInternalServerError)
	fmt.Fprintf(c.Writer, "Internal Server Error: %v", e)
	ulog.Error("Internal Server Error: %v", e)
	c.End()
}
