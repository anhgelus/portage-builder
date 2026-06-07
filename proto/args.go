package proto

import (
	"bytes"
	"errors"
	"fmt"
	"reflect"
	"regexp"

	"anhgelus.world/portage-builder/proto/cbor"
)

var (
	ErrInvalidArg = errors.New("invalid argument")
	ErrArgsNumber = errors.New("invalid number of arguments")
)

type Package string

var packageRegexp = regexp.MustCompile(`^[a-zA-Z0-9-]+/[a-zA-Z0-9-]+$`)

// IsPackage indicates if the string is a valid Gentoo package.
func IsPackage(s string) bool {
	return packageRegexp.MatchString(s)
}

func (p *Package) UnmarshalCBOR(b []byte) ([]byte, error) {
	var s string
	rest, err := cbor.Unmarshal(b, &s)
	if err != nil {
		return nil, err
	}
	if !IsPackage(s) {
		return nil, fmt.Errorf("%w: not a package", ErrInvalidArg)
	}
	*p = Package(s)
	return rest, nil
}

func UnmarshalArgsFor[T any](b []byte) (T, error) {
	var arg T
	return arg, UnmarshalArgs(b, &arg)
}

func UnmarshalArgs(b []byte, v any) error {
	val := reflect.ValueOf(v)
	if val.Kind() != reflect.Pointer || val.IsNil() {
		return fmt.Errorf("%w: expected a non nil pointer to a struct, not %T", ErrInvalidArg, v)
	}
	if val.Elem().Kind() != reflect.Struct {
		return fmt.Errorf("%w: expected pointing to a struct, not %T", ErrInvalidArg, v)
	}
	for f, v := range val.Elem().Fields() {
		n := reflect.New(f.Type)
		var err error
		b, err = cbor.Unmarshal(b, n.Interface())
		if err != nil {
			return err
		}
		v.Set(n.Elem())
	}
	if len(b) != 0 {
		return fmt.Errorf("%w: requires %d", ErrArgsNumber, val.Elem().NumField())
	}
	return nil
}

func MarshalArgs(v any) ([]byte, error) {
	val := reflect.ValueOf(v)
	if val.Kind() == reflect.Pointer {
		if val.IsNil() {
			return nil, nil
		}
		return MarshalArgs(val.Elem().Interface())
	}
	if val.Kind() != reflect.Struct {
		return nil, fmt.Errorf("%w: expected a struct, not %T", ErrInvalidArg, v)
	}
	var buf bytes.Buffer
	for _, v := range val.Fields() {
		b, err := cbor.Marshal(v.Interface())
		if err != nil {
			return nil, err
		}
		buf.Write(b)
	}
	return buf.Bytes(), nil
}
