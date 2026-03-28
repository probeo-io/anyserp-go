package anyserp

import (
	"testing"
)

func TestJsonStrValid(t *testing.T) {
	m := map[string]interface{}{"name": "test"}
	if got := jsonStr(m, "name"); got != "test" {
		t.Errorf("expected 'test', got %q", got)
	}
}

func TestJsonStrMissing(t *testing.T) {
	m := map[string]interface{}{"name": "test"}
	if got := jsonStr(m, "missing"); got != "" {
		t.Errorf("expected empty string, got %q", got)
	}
}

func TestJsonStrNilMap(t *testing.T) {
	if got := jsonStr(nil, "key"); got != "" {
		t.Errorf("expected empty string, got %q", got)
	}
}

func TestJsonStrNilValue(t *testing.T) {
	m := map[string]interface{}{"key": nil}
	if got := jsonStr(m, "key"); got != "" {
		t.Errorf("expected empty string, got %q", got)
	}
}

func TestJsonStrWrongType(t *testing.T) {
	m := map[string]interface{}{"key": 123}
	if got := jsonStr(m, "key"); got != "" {
		t.Errorf("expected empty string for non-string, got %q", got)
	}
}

func TestJsonIntFloat64(t *testing.T) {
	m := map[string]interface{}{"count": float64(42)}
	if got := jsonInt(m, "count"); got != 42 {
		t.Errorf("expected 42, got %d", got)
	}
}

func TestJsonIntNative(t *testing.T) {
	m := map[string]interface{}{"count": 42}
	if got := jsonInt(m, "count"); got != 42 {
		t.Errorf("expected 42, got %d", got)
	}
}

func TestJsonIntInt64(t *testing.T) {
	m := map[string]interface{}{"count": int64(42)}
	if got := jsonInt(m, "count"); got != 42 {
		t.Errorf("expected 42, got %d", got)
	}
}

func TestJsonIntMissing(t *testing.T) {
	m := map[string]interface{}{}
	if got := jsonInt(m, "missing"); got != 0 {
		t.Errorf("expected 0, got %d", got)
	}
}

func TestJsonIntNilMap(t *testing.T) {
	if got := jsonInt(nil, "key"); got != 0 {
		t.Errorf("expected 0, got %d", got)
	}
}

func TestJsonIntWrongType(t *testing.T) {
	m := map[string]interface{}{"key": "not-int"}
	if got := jsonInt(m, "key"); got != 0 {
		t.Errorf("expected 0 for string type, got %d", got)
	}
}

func TestJsonFloatValid(t *testing.T) {
	m := map[string]interface{}{"val": float64(3.14)}
	if got := jsonFloat(m, "val"); got != 3.14 {
		t.Errorf("expected 3.14, got %f", got)
	}
}

func TestJsonFloatFromInt(t *testing.T) {
	m := map[string]interface{}{"val": 5}
	if got := jsonFloat(m, "val"); got != 5.0 {
		t.Errorf("expected 5.0, got %f", got)
	}
}

func TestJsonFloatFromInt64(t *testing.T) {
	m := map[string]interface{}{"val": int64(5)}
	if got := jsonFloat(m, "val"); got != 5.0 {
		t.Errorf("expected 5.0, got %f", got)
	}
}

func TestJsonFloatNilMap(t *testing.T) {
	if got := jsonFloat(nil, "key"); got != 0 {
		t.Errorf("expected 0, got %f", got)
	}
}

func TestJsonFloatWrongType(t *testing.T) {
	m := map[string]interface{}{"key": "not-float"}
	if got := jsonFloat(m, "key"); got != 0 {
		t.Errorf("expected 0 for string type, got %f", got)
	}
}

func TestJsonObjValid(t *testing.T) {
	inner := map[string]interface{}{"nested": true}
	m := map[string]interface{}{"obj": inner}
	got := jsonObj(m, "obj")
	if got == nil {
		t.Fatal("expected non-nil object")
	}
	if got["nested"] != true {
		t.Errorf("expected nested=true, got %v", got["nested"])
	}
}

func TestJsonObjNilMap(t *testing.T) {
	if got := jsonObj(nil, "key"); got != nil {
		t.Errorf("expected nil, got %v", got)
	}
}

func TestJsonObjMissing(t *testing.T) {
	m := map[string]interface{}{}
	if got := jsonObj(m, "missing"); got != nil {
		t.Errorf("expected nil, got %v", got)
	}
}

func TestJsonObjWrongType(t *testing.T) {
	m := map[string]interface{}{"key": "string-value"}
	if got := jsonObj(m, "key"); got != nil {
		t.Errorf("expected nil for non-object, got %v", got)
	}
}

func TestJsonArrayValid(t *testing.T) {
	m := map[string]interface{}{
		"items": []interface{}{
			map[string]interface{}{"id": float64(1)},
			map[string]interface{}{"id": float64(2)},
		},
	}
	got := jsonArray(m, "items")
	if len(got) != 2 {
		t.Fatalf("expected 2 items, got %d", len(got))
	}
}

func TestJsonArrayNilMap(t *testing.T) {
	if got := jsonArray(nil, "key"); got != nil {
		t.Errorf("expected nil, got %v", got)
	}
}

func TestJsonArrayMissing(t *testing.T) {
	m := map[string]interface{}{}
	if got := jsonArray(m, "missing"); got != nil {
		t.Errorf("expected nil, got %v", got)
	}
}

func TestJsonArrayWrongType(t *testing.T) {
	m := map[string]interface{}{"key": "not-array"}
	if got := jsonArray(m, "key"); got != nil {
		t.Errorf("expected nil for non-array, got %v", got)
	}
}

func TestJsonArraySkipsNonObjects(t *testing.T) {
	m := map[string]interface{}{
		"items": []interface{}{
			map[string]interface{}{"id": float64(1)},
			"not-an-object",
			float64(42),
		},
	}
	got := jsonArray(m, "items")
	if len(got) != 1 {
		t.Fatalf("expected 1 item (skipping non-objects), got %d", len(got))
	}
}

func TestJsonStrArrayValid(t *testing.T) {
	m := map[string]interface{}{
		"tags": []interface{}{"go", "test"},
	}
	got := jsonStrArray(m, "tags")
	if len(got) != 2 {
		t.Fatalf("expected 2 strings, got %d", len(got))
	}
	if got[0] != "go" || got[1] != "test" {
		t.Errorf("unexpected values: %v", got)
	}
}

func TestJsonStrArrayNilMap(t *testing.T) {
	if got := jsonStrArray(nil, "key"); got != nil {
		t.Errorf("expected nil, got %v", got)
	}
}

func TestJsonStrArraySkipsNonStrings(t *testing.T) {
	m := map[string]interface{}{
		"tags": []interface{}{"go", float64(42), "test"},
	}
	got := jsonStrArray(m, "tags")
	if len(got) != 2 {
		t.Fatalf("expected 2 strings, got %d", len(got))
	}
}

func TestJsonIntArrayValid(t *testing.T) {
	m := map[string]interface{}{
		"ids": []interface{}{float64(1), float64(2), float64(3)},
	}
	got := jsonIntArray(m, "ids")
	if len(got) != 3 {
		t.Fatalf("expected 3 ints, got %d", len(got))
	}
	if got[0] != 1 || got[1] != 2 || got[2] != 3 {
		t.Errorf("unexpected values: %v", got)
	}
}

func TestJsonIntArrayNilMap(t *testing.T) {
	if got := jsonIntArray(nil, "key"); got != nil {
		t.Errorf("expected nil, got %v", got)
	}
}

func TestJsonIntArrayRounding(t *testing.T) {
	m := map[string]interface{}{
		"ids": []interface{}{float64(1.7), float64(2.3)},
	}
	got := jsonIntArray(m, "ids")
	if got[0] != 2 {
		t.Errorf("expected 2 from rounding 1.7, got %d", got[0])
	}
	if got[1] != 2 {
		t.Errorf("expected 2 from rounding 2.3, got %d", got[1])
	}
}
