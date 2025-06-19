package main

import (
	"encoding"
	"encoding/json"
	"fmt"
)

func Str[T any](x T) string {
	if any(x) == nil {
		return "<nil>"
	}
	switch v := any(x).(type) {
	case string:
		return v
	case *string:
		if v != nil {
			return *v
		}
	case []byte:
		return string(v)
	case *[]byte:
		if v != nil {
			return string(*v)
		}
	case fmt.Stringer:
		return v.String()
	case encoding.TextMarshaler:
		if b, err := v.MarshalText(); err == nil {
			return string(b)
		}
	}
	if b, err := json.Marshal(x); err == nil {
		return string(b)
	}
	return fmt.Sprintf("%v", x)
}
