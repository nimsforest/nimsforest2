package runtime

import (
	"testing"
)

func TestLuaVMBasic(t *testing.T) {
	vm := NewLuaVM()
	defer vm.Close()

	script := `
function process(input)
    return {
        result = input.value * 2,
        name = input.name
    }
end
`
	if err := vm.LoadString(script); err != nil {
		t.Fatalf("LoadString failed: %v", err)
	}

	input := map[string]interface{}{
		"value": float64(21),
		"name":  "test",
	}

	output, err := vm.CallProcess(input)
	if err != nil {
		t.Fatalf("CallProcess failed: %v", err)
	}

	if output["result"] != float64(42) {
		t.Errorf("expected result=42, got %v", output["result"])
	}
	if output["name"] != "test" {
		t.Errorf("expected name='test', got %v", output["name"])
	}
}

func TestLuaContainsHelper(t *testing.T) {
	vm := NewLuaVM()
	defer vm.Close()

	script := `
function process(input)
    return {
        has_ceo = contains(input.title, "CEO"),
        has_vp = contains(input.title, "VP"),
        has_manager = contains(input.title, "Manager")
    }
end
`
	if err := vm.LoadString(script); err != nil {
		t.Fatalf("LoadString failed: %v", err)
	}

	tests := []struct {
		title    string
		expected map[string]bool
	}{
		{"CEO of Company", map[string]bool{"has_ceo": true, "has_vp": false, "has_manager": false}},
		{"VP Engineering", map[string]bool{"has_ceo": false, "has_vp": true, "has_manager": false}},
		{"Product Manager", map[string]bool{"has_ceo": false, "has_vp": false, "has_manager": true}},
		{"Engineer", map[string]bool{"has_ceo": false, "has_vp": false, "has_manager": false}},
	}

	for _, tt := range tests {
		input := map[string]interface{}{"title": tt.title}
		output, err := vm.CallProcess(input)
		if err != nil {
			t.Fatalf("CallProcess failed for %s: %v", tt.title, err)
		}

		for key, expected := range tt.expected {
			if output[key] != expected {
				t.Errorf("title=%s, %s: expected %v, got %v", tt.title, key, expected, output[key])
			}
		}
	}
}

func TestLuaJSONHelpers(t *testing.T) {
	vm := NewLuaVM()
	defer vm.Close()

	script := `
function process(input)
    -- Test encode
    local encoded = json.encode({foo = "bar", num = 42})
    
    -- Test decode
    local decoded = json.decode('{"key": "value", "array": [1, 2, 3]}')
    
    return {
        encoded = encoded,
        decoded_key = decoded.key,
        array_len = #decoded.array
    }
end
`
	if err := vm.LoadString(script); err != nil {
		t.Fatalf("LoadString failed: %v", err)
	}

	output, err := vm.CallProcess(map[string]interface{}{})
	if err != nil {
		t.Fatalf("CallProcess failed: %v", err)
	}

	if output["decoded_key"] != "value" {
		t.Errorf("expected decoded_key='value', got %v", output["decoded_key"])
	}
	if output["array_len"] != float64(3) {
		t.Errorf("expected array_len=3, got %v", output["array_len"])
	}
}

func TestLuaArrayHandling(t *testing.T) {
	vm := NewLuaVM()
	defer vm.Close()

	script := `
function process(input)
    local signals = {}
    table.insert(signals, "first")
    table.insert(signals, "second")
    table.insert(signals, "third")
    return {
        signals = signals,
        count = #signals
    }
end
`
	if err := vm.LoadString(script); err != nil {
		t.Fatalf("LoadString failed: %v", err)
	}

	output, err := vm.CallProcess(map[string]interface{}{})
	if err != nil {
		t.Fatalf("CallProcess failed: %v", err)
	}

	signals, ok := output["signals"].([]interface{})
	if !ok {
		t.Fatalf("signals should be an array, got %T", output["signals"])
	}
	if len(signals) != 3 {
		t.Errorf("expected 3 signals, got %d", len(signals))
	}
	if signals[0] != "first" {
		t.Errorf("expected signals[0]='first', got %v", signals[0])
	}
}

func TestLuaNoProcessFunction(t *testing.T) {
	vm := NewLuaVM()
	defer vm.Close()

	// Script without process function
	script := `
function other_func(x)
    return x
end
`
	if err := vm.LoadString(script); err != nil {
		t.Fatalf("LoadString failed: %v", err)
	}

	_, err := vm.CallProcess(map[string]interface{}{})
	if err == nil {
		t.Error("expected error when process function is not defined")
	}
}

func TestLuaProcessError(t *testing.T) {
	vm := NewLuaVM()
	defer vm.Close()

	// Script with error
	script := `
function process(input)
    error("intentional error")
end
`
	if err := vm.LoadString(script); err != nil {
		t.Fatalf("LoadString failed: %v", err)
	}

	_, err := vm.CallProcess(map[string]interface{}{})
	if err == nil {
		t.Error("expected error when process function throws")
	}
}

func TestLuaNestedData(t *testing.T) {
	vm := NewLuaVM()
	defer vm.Close()

	script := `
function process(input)
    return {
        user = {
            name = input.name,
            details = {
                age = input.age,
                active = true
            }
        }
    }
end
`
	if err := vm.LoadString(script); err != nil {
		t.Fatalf("LoadString failed: %v", err)
	}

	input := map[string]interface{}{
		"name": "Alice",
		"age":  float64(30),
	}

	output, err := vm.CallProcess(input)
	if err != nil {
		t.Fatalf("CallProcess failed: %v", err)
	}

	user, ok := output["user"].(map[string]interface{})
	if !ok {
		t.Fatalf("user should be a map, got %T", output["user"])
	}
	if user["name"] != "Alice" {
		t.Errorf("expected user.name='Alice', got %v", user["name"])
	}

	details, ok := user["details"].(map[string]interface{})
	if !ok {
		t.Fatalf("details should be a map, got %T", user["details"])
	}
	if details["age"] != float64(30) {
		t.Errorf("expected details.age=30, got %v", details["age"])
	}
	if details["active"] != true {
		t.Errorf("expected details.active=true, got %v", details["active"])
	}
}
