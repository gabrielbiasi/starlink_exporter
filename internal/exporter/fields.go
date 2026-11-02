package exporter

import (
	"fmt"

	"github.com/jhump/protoreflect/dynamic"
	"google.golang.org/protobuf/types/descriptorpb"
)

func field(m *dynamic.Message, name string) (interface{}, bool) {
	if m == nil {
		return nil, false
	}
	fd := m.GetMessageDescriptor().FindFieldByName(name)
	if fd == nil || !m.HasField(fd) {
		return nil, false
	}
	v, err := m.TryGetField(fd)
	if err != nil {
		return nil, false
	}
	return v, true
}

func child(m *dynamic.Message, name string) *dynamic.Message {
	v, ok := field(m, name)
	if !ok {
		return nil
	}
	dm, _ := v.(*dynamic.Message)
	return dm
}

func stringField(m *dynamic.Message, name string) string {
	v, ok := field(m, name)
	if !ok {
		return ""
	}
	s, _ := v.(string)
	return s
}

func floatField(m *dynamic.Message, name string) (float64, bool) {
	v, ok := field(m, name)
	if !ok {
		return 0, false
	}
	return asFloat(v)
}

func boolField(m *dynamic.Message, name string) (bool, bool) {
	if m == nil {
		return false, false
	}
	fd := m.GetMessageDescriptor().FindFieldByName(name)
	if fd == nil || fd.GetType() != descriptorpb.FieldDescriptorProto_TYPE_BOOL {
		return false, false
	}
	// Proto3 leaves false booleans unset. The field still exists and the value is false.
	if !m.HasField(fd) {
		return false, true
	}
	v, err := m.TryGetField(fd)
	if err != nil {
		return false, false
	}
	b, ok := v.(bool)
	return b, ok
}

func enumName(m *dynamic.Message, name string) string {
	if m == nil {
		return ""
	}
	fd := m.GetMessageDescriptor().FindFieldByName(name)
	if fd == nil || fd.GetEnumType() == nil || !m.HasField(fd) {
		return ""
	}
	v, err := m.TryGetField(fd)
	if err != nil {
		return ""
	}
	num, ok := asInt32(v)
	if !ok {
		return ""
	}
	ev := fd.GetEnumType().FindValueByNumber(num)
	if ev == nil {
		return fmt.Sprintf("%d", num)
	}
	return ev.GetName()
}

func boolFlags(m *dynamic.Message) map[string]float64 {
	out := map[string]float64{}
	if m == nil {
		return out
	}
	for _, fd := range m.GetMessageDescriptor().GetFields() {
		if fd.IsRepeated() || fd.GetType() != descriptorpb.FieldDescriptorProto_TYPE_BOOL {
			continue
		}
		val := 0.0
		if m.HasField(fd) {
			v, err := m.TryGetField(fd)
			if err == nil {
				if b, ok := v.(bool); ok && b {
					val = 1
				}
			}
		}
		out[fd.GetName()] = val
	}
	return out
}

func floatList(m *dynamic.Message, name string) ([]float64, error) {
	v, ok := field(m, name)
	if !ok {
		return nil, nil
	}
	xs, ok := flattenFloats(v)
	if !ok {
		return nil, fmt.Errorf("%s has unexpected type %T", name, v)
	}
	return xs, nil
}

func messageList(m *dynamic.Message, name string) ([]*dynamic.Message, error) {
	v, ok := field(m, name)
	if !ok {
		return nil, nil
	}
	switch xs := v.(type) {
	case []interface{}:
		out := make([]*dynamic.Message, 0, len(xs))
		for _, item := range xs {
			dm, ok := item.(*dynamic.Message)
			if !ok {
				return nil, fmt.Errorf("%s element has unexpected type %T", name, item)
			}
			out = append(out, dm)
		}
		return out, nil
	case []*dynamic.Message:
		return xs, nil
	default:
		return nil, fmt.Errorf("%s has unexpected type %T", name, v)
	}
}

func wedgeSeries(m *dynamic.Message) map[string][]float64 {
	out := map[string][]float64{}
	if m == nil {
		return out
	}
	for _, fd := range m.GetMessageDescriptor().GetFields() {
		if !fd.IsRepeated() || fd.GetMessageType() != nil || !containsWedge(fd.GetName()) || !m.HasField(fd) {
			continue
		}
		v, err := m.TryGetField(fd)
		if err != nil {
			continue
		}
		xs, ok := flattenFloats(v)
		if !ok {
			continue
		}
		out[fd.GetName()] = xs
	}
	return out
}

func containsWedge(name string) bool {
	for i := 0; i+5 <= len(name); i++ {
		if name[i:i+5] == "wedge" {
			return true
		}
	}
	return false
}

func asFloat(v interface{}) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case float32:
		return float64(n), true
	case int:
		return float64(n), true
	case int32:
		return float64(n), true
	case int64:
		return float64(n), true
	case uint32:
		return float64(n), true
	case uint64:
		return float64(n), true
	default:
		return 0, false
	}
}

func asInt32(v interface{}) (int32, bool) {
	switch n := v.(type) {
	case int32:
		return n, true
	case int64:
		return int32(n), true
	case uint32:
		return int32(n), true
	case uint64:
		return int32(n), true
	case int:
		return int32(n), true
	default:
		return 0, false
	}
}

func flattenFloats(v interface{}) ([]float64, bool) {
	switch xs := v.(type) {
	case []float64:
		return append([]float64(nil), xs...), true
	case []float32:
		out := make([]float64, len(xs))
		for i, n := range xs {
			out[i] = float64(n)
		}
		return out, true
	case []interface{}:
		out := make([]float64, len(xs))
		for i, n := range xs {
			f, ok := asFloat(n)
			if !ok {
				return nil, false
			}
			out[i] = f
		}
		return out, true
	default:
		return nil, false
	}
}

func mean(xs []float64) (float64, bool) {
	if len(xs) == 0 {
		return 0, false
	}
	var sum float64
	for _, v := range xs {
		sum += v
	}
	return sum / float64(len(xs)), true
}

func last(xs []float64) (float64, bool) {
	if len(xs) == 0 {
		return 0, false
	}
	return xs[len(xs)-1], true
}

func stateNumber(state string) float64 {
	switch state {
	case "CONNECTED":
		return 1
	case "SEARCHING", "SKY_SEARCH", "NO_SCHEDULE":
		return 2
	case "BOOTING":
		return 3
	case "OFF":
		return 5
	case "":
		return 0
	default:
		return 4
	}
}

func gaugeBool(v bool) float64 {
	if v {
		return 1
	}
	return 0
}

func orUnknown(v string) string {
	if v == "" {
		return "unknown"
	}
	return v
}
