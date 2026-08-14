package masker

import (
	"bytes"
	"encoding/json"
	"reflect"
)

// maxObjectDepth 限制递归深度，避免深层嵌套导致的栈问题。
const maxObjectDepth = 32

// MaskObject 递归脱敏 JSON 对象中的所有字符串字段。
func (m *Masker) MaskObject(obj interface{}, typ string) interface{} {
	typ = NormalizeType(typ)
	if typ == "" {
		typ = string(TypePhone)
	}
	return m.maskObjectValue(obj, typ, 0)
}

// MaskObjectJSON 对 JSON 字节做递归脱敏。
func (m *Masker) MaskObjectJSON(data []byte, typ string) ([]byte, error) {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()
	var obj interface{}
	if err := dec.Decode(&obj); err != nil {
		return nil, err
	}
	masked := m.MaskObject(obj, typ)
	return json.Marshal(masked)
}

// maskObjectValue 递归处理单个值。
func (m *Masker) maskObjectValue(obj interface{}, typ string, depth int) interface{} {
	if depth > maxObjectDepth {
		return obj
	}

	switch v := obj.(type) {
	case nil:
		return nil
	case string:
		return m.MaskValue(v, typ)
	case bool:
		return v
	case float64:
		return v
	case float32:
		return v
	case int:
		return v
	case int64:
		return v
	case json.Number:
		return v
	case map[string]interface{}:
		out := make(map[string]interface{}, len(v))
		for k, val := range v {
			out[k] = m.maskObjectValue(val, typ, depth+1)
		}
		return out
	case []interface{}:
		out := make([]interface{}, len(v))
		for i, val := range v {
			out[i] = m.maskObjectValue(val, typ, depth+1)
		}
		return out
	default:
		return m.maskObjectReflect(obj, typ, depth)
	}
}

// maskObjectReflect 通过反射处理 map/slice 等其它形态。
func (m *Masker) maskObjectReflect(obj interface{}, typ string, depth int) interface{} {
	rv := reflect.ValueOf(obj)
	if !rv.IsValid() {
		return nil
	}

	switch rv.Kind() {
	case reflect.Map:
		out := reflect.MakeMapWithSize(rv.Type(), rv.Len())
		iter := rv.MapRange()
		for iter.Next() {
			masked := m.maskObjectValue(iter.Value().Interface(), typ, depth+1)
			out.SetMapIndex(iter.Key(), reflect.ValueOf(masked))
		}
		return out.Interface()
	case reflect.Slice, reflect.Array:
		out := reflect.MakeSlice(rv.Type(), rv.Len(), rv.Len())
		for i := 0; i < rv.Len(); i++ {
			masked := m.maskObjectValue(rv.Index(i).Interface(), typ, depth+1)
			out.Index(i).Set(reflect.ValueOf(masked))
		}
		return out.Interface()
	case reflect.String:
		return m.MaskValue(rv.String(), typ)
	case reflect.Ptr:
		if rv.IsNil() {
			return nil
		}
		return m.maskObjectReflect(rv.Elem().Interface(), typ, depth)
	case reflect.Interface:
		return m.maskObjectValue(rv.Interface(), typ, depth)
	default:
		return obj
	}
}

// isMaskableString 判断值是否为字符串。
func isMaskableString(v interface{}) bool {
	_, ok := v.(string)
	return ok
}

// cloneStringMap 深拷贝一个字符串 map。
func cloneStringMap(in map[string]interface{}) map[string]interface{} {
	out := make(map[string]interface{}, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}
