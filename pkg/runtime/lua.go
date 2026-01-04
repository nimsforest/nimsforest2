package runtime

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	lua "github.com/yuin/gopher-lua"
)

// LuaVM is a wrapper around a Lua state with helper functions preloaded.
type LuaVM struct {
	state *lua.LState
}

// NewLuaVM creates a new Lua VM with helper functions preloaded.
func NewLuaVM() *LuaVM {
	L := lua.NewState()
	vm := &LuaVM{state: L}
	vm.registerHelpers()
	return vm
}

// Close closes the Lua VM and releases resources.
func (vm *LuaVM) Close() {
	vm.state.Close()
}

// LoadScript loads a Lua script from a file.
func (vm *LuaVM) LoadScript(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read script: %w", err)
	}
	return vm.LoadString(string(data))
}

// LoadString loads a Lua script from a string.
func (vm *LuaVM) LoadString(script string) error {
	if err := vm.state.DoString(script); err != nil {
		return fmt.Errorf("failed to load script: %w", err)
	}
	return nil
}

// CallProcess calls the process(input) function in the loaded script.
// Input is converted from Go map to Lua table, and output is converted back.
func (vm *LuaVM) CallProcess(input map[string]interface{}) (map[string]interface{}, error) {
	// Get the process function
	fn := vm.state.GetGlobal("process")
	if fn == lua.LNil {
		return nil, fmt.Errorf("process function not defined in script")
	}

	// Convert input to Lua table
	inputTable := vm.mapToTable(input)

	// Call process(input)
	if err := vm.state.CallByParam(lua.P{
		Fn:      fn,
		NRet:    1,
		Protect: true,
	}, inputTable); err != nil {
		return nil, fmt.Errorf("process function error: %w", err)
	}

	// Get result
	result := vm.state.Get(-1)
	vm.state.Pop(1)

	// Convert result to Go map
	if result.Type() != lua.LTTable {
		return nil, fmt.Errorf("process function must return a table, got %s", result.Type())
	}

	output := vm.tableToMap(result.(*lua.LTable))
	return output, nil
}

// registerHelpers registers helper functions available to Lua scripts.
func (vm *LuaVM) registerHelpers() {
	// contains(str, substr) - check if string contains substring
	vm.state.SetGlobal("contains", vm.state.NewFunction(luaContains))

	// log(msg) - log a message
	vm.state.SetGlobal("log", vm.state.NewFunction(luaLog))

	// Register json module
	jsonMod := vm.state.NewTable()
	vm.state.SetField(jsonMod, "encode", vm.state.NewFunction(luaJSONEncode))
	vm.state.SetField(jsonMod, "decode", vm.state.NewFunction(luaJSONDecode))
	vm.state.SetGlobal("json", jsonMod)
}

// luaContains implements contains(str, substr) in Lua
func luaContains(L *lua.LState) int {
	str := L.CheckString(1)
	substr := L.CheckString(2)
	L.Push(lua.LBool(strings.Contains(str, substr)))
	return 1
}

// luaLog implements log(msg) in Lua
func luaLog(L *lua.LState) int {
	msg := L.CheckString(1)
	log.Printf("[Lua] %s", msg)
	return 0
}

// luaJSONEncode implements json.encode(table) in Lua
func luaJSONEncode(L *lua.LState) int {
	tbl := L.CheckTable(1)
	data := tableToGoValue(tbl)
	bytes, err := json.Marshal(data)
	if err != nil {
		L.Push(lua.LNil)
		L.Push(lua.LString(err.Error()))
		return 2
	}
	L.Push(lua.LString(string(bytes)))
	return 1
}

// luaJSONDecode implements json.decode(str) in Lua
func luaJSONDecode(L *lua.LState) int {
	str := L.CheckString(1)
	var data interface{}
	if err := json.Unmarshal([]byte(str), &data); err != nil {
		L.Push(lua.LNil)
		L.Push(lua.LString(err.Error()))
		return 2
	}
	L.Push(goValueToLua(L, data))
	return 1
}

// mapToTable converts a Go map to a Lua table
func (vm *LuaVM) mapToTable(m map[string]interface{}) *lua.LTable {
	return goMapToTable(vm.state, m)
}

// goMapToTable converts a Go map to a Lua table
func goMapToTable(L *lua.LState, m map[string]interface{}) *lua.LTable {
	tbl := L.NewTable()
	for k, v := range m {
		tbl.RawSetString(k, goValueToLua(L, v))
	}
	return tbl
}

// goValueToLua converts a Go value to a Lua value
func goValueToLua(L *lua.LState, v interface{}) lua.LValue {
	switch val := v.(type) {
	case nil:
		return lua.LNil
	case bool:
		return lua.LBool(val)
	case float64:
		return lua.LNumber(val)
	case float32:
		return lua.LNumber(val)
	case int:
		return lua.LNumber(val)
	case int64:
		return lua.LNumber(val)
	case int32:
		return lua.LNumber(val)
	case string:
		return lua.LString(val)
	case []interface{}:
		tbl := L.NewTable()
		for i, item := range val {
			tbl.RawSetInt(i+1, goValueToLua(L, item))
		}
		return tbl
	case map[string]interface{}:
		return goMapToTable(L, val)
	default:
		return lua.LString(fmt.Sprintf("%v", val))
	}
}

// tableToMap converts a Lua table to a Go map
func (vm *LuaVM) tableToMap(tbl *lua.LTable) map[string]interface{} {
	return tableToGoMap(tbl)
}

// tableToGoMap converts a Lua table to a Go map
func tableToGoMap(tbl *lua.LTable) map[string]interface{} {
	result := make(map[string]interface{})
	tbl.ForEach(func(k, v lua.LValue) {
		key := luaValueToString(k)
		result[key] = luaValueToGo(v)
	})
	return result
}

// tableToGoValue converts a Lua table to the appropriate Go type (map or slice)
func tableToGoValue(tbl *lua.LTable) interface{} {
	// Check if table is an array (sequential integer keys starting at 1)
	isArray := true
	maxIndex := 0
	tbl.ForEach(func(k, v lua.LValue) {
		if num, ok := k.(lua.LNumber); ok {
			idx := int(num)
			if idx > maxIndex {
				maxIndex = idx
			}
		} else {
			isArray = false
		}
	})

	if isArray && maxIndex > 0 && tbl.Len() == maxIndex {
		// It's an array
		arr := make([]interface{}, maxIndex)
		for i := 1; i <= maxIndex; i++ {
			arr[i-1] = luaValueToGo(tbl.RawGetInt(i))
		}
		return arr
	}

	// It's a map
	return tableToGoMap(tbl)
}

// luaValueToGo converts a Lua value to a Go value
func luaValueToGo(v lua.LValue) interface{} {
	switch val := v.(type) {
	case lua.LBool:
		return bool(val)
	case lua.LNumber:
		return float64(val)
	case lua.LString:
		return string(val)
	case *lua.LTable:
		return tableToGoValue(val)
	case *lua.LNilType:
		return nil
	default:
		return nil
	}
}

// luaValueToString converts a Lua value to a string (for map keys)
func luaValueToString(v lua.LValue) string {
	switch val := v.(type) {
	case lua.LString:
		return string(val)
	case lua.LNumber:
		return fmt.Sprintf("%v", float64(val))
	default:
		return v.String()
	}
}
