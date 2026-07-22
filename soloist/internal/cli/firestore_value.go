package cli

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
)

func fsDecode(v map[string]any) any {
	if v == nil {
		return nil
	}
	if raw, ok := v["stringValue"]; ok {
		if s, ok := raw.(string); ok {
			return s
		}
		return fmt.Sprint(raw)
	}
	if raw, ok := v["integerValue"]; ok {
		s := fmt.Sprint(raw)
		if i, err := strconv.Atoi(s); err == nil {
			return i
		}
		if i64, err := strconv.ParseInt(s, 10, 64); err == nil {
			return i64
		}
		return s
	}
	if raw, ok := v["doubleValue"]; ok {
		switch n := raw.(type) {
		case float64:
			return n
		case float32:
			return float64(n)
		case json.Number:
			f, _ := n.Float64()
			return f
		case string:
			f, err := strconv.ParseFloat(n, 64)
			if err == nil {
				return f
			}
			return n
		default:
			f, err := strconv.ParseFloat(fmt.Sprint(raw), 64)
			if err == nil {
				return f
			}
			return raw
		}
	}
	if raw, ok := v["booleanValue"]; ok {
		if b, ok := raw.(bool); ok {
			return b
		}
		return raw
	}
	if _, ok := v["nullValue"]; ok {
		return nil
	}
	if raw, ok := v["timestampValue"]; ok {
		if s, ok := raw.(string); ok {
			return s
		}
		return fmt.Sprint(raw)
	}
	if raw, ok := v["mapValue"]; ok {
		mapValue, _ := raw.(map[string]any)
		fields, _ := mapValue["fields"].(map[string]any)
		out := make(map[string]any, len(fields))
		for key, field := range fields {
			fieldValue, ok := field.(map[string]any)
			if !ok {
				out[key] = field
				continue
			}
			out[key] = fsDecode(fieldValue)
		}
		return out
	}
	if raw, ok := v["arrayValue"]; ok {
		arrayValue, _ := raw.(map[string]any)
		values, _ := arrayValue["values"].([]any)
		out := make([]any, 0, len(values))
		for _, value := range values {
			fieldValue, ok := value.(map[string]any)
			if !ok {
				out = append(out, value)
				continue
			}
			out = append(out, fsDecode(fieldValue))
		}
		return out
	}
	return nil
}

func fsEncode(v any) map[string]any {
	if v == nil {
		return map[string]any{"nullValue": nil}
	}
	switch typed := v.(type) {
	case string:
		return fsEncodeString(typed)
	case bool:
		return fsEncodeBool(typed)
	case int:
		return fsEncodeInt(typed)
	case int8, int16, int32, int64:
		return map[string]any{"integerValue": strconv.FormatInt(reflect.ValueOf(v).Int(), 10)}
	case uint, uint8, uint16, uint32, uint64:
		return map[string]any{"integerValue": strconv.FormatUint(reflect.ValueOf(v).Uint(), 10)}
	case float32:
		return map[string]any{"doubleValue": float64(typed)}
	case float64:
		return map[string]any{"doubleValue": typed}
	case json.Number:
		if i, err := typed.Int64(); err == nil {
			return map[string]any{"integerValue": strconv.FormatInt(i, 10)}
		}
		if f, err := typed.Float64(); err == nil {
			return map[string]any{"doubleValue": f}
		}
		return fsEncodeString(typed.String())
	case map[string]any:
		fields := make(map[string]any, len(typed))
		for key, value := range typed {
			fields[key] = fsEncode(value)
		}
		return map[string]any{"mapValue": map[string]any{"fields": fields}}
	case []any:
		values := make([]any, 0, len(typed))
		for _, value := range typed {
			values = append(values, fsEncode(value))
		}
		return map[string]any{"arrayValue": map[string]any{"values": values}}
	}

	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Map:
		if rv.Type().Key().Kind() != reflect.String {
			return fsEncodeString(fmt.Sprint(v))
		}
		fields := make(map[string]any, rv.Len())
		iter := rv.MapRange()
		for iter.Next() {
			fields[iter.Key().String()] = fsEncode(iter.Value().Interface())
		}
		return map[string]any{"mapValue": map[string]any{"fields": fields}}
	case reflect.Slice, reflect.Array:
		values := make([]any, 0, rv.Len())
		for i := 0; i < rv.Len(); i++ {
			values = append(values, fsEncode(rv.Index(i).Interface()))
		}
		return map[string]any{"arrayValue": map[string]any{"values": values}}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return map[string]any{"integerValue": strconv.FormatInt(rv.Int(), 10)}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return map[string]any{"integerValue": strconv.FormatUint(rv.Uint(), 10)}
	case reflect.Float32, reflect.Float64:
		return map[string]any{"doubleValue": rv.Convert(reflect.TypeOf(float64(0))).Float()}
	}

	return fsEncodeString(fmt.Sprint(v))
}

func fsEncodeString(s string) map[string]any {
	return map[string]any{"stringValue": s}
}

func fsEncodeBool(b bool) map[string]any {
	return map[string]any{"booleanValue": b}
}

func fsEncodeInt(i int) map[string]any {
	return map[string]any{"integerValue": strconv.Itoa(i)}
}
