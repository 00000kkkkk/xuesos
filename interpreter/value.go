package interpreter

import (
	"fmt"
	"strings"
)

// ValueType represents the type of a runtime value.
type ValueType int

const (
	VAL_INT ValueType = iota
	VAL_FLOAT
	VAL_STRING
	VAL_CHAR
	VAL_BOOL
	VAL_NULL
	VAL_ARRAY
	VAL_MAP
	VAL_STRUCT
	VAL_FUNCTION
	VAL_BUILTIN
	VAL_ENUM_VARIANT
	VAL_RANGE
	VAL_CHANNEL  // channel for concurrency
	VAL_POINTER  // pointer (address-of)
	VAL_RETURN   // wrapper for return values
	VAL_BREAK    // signal for break
	VAL_CONTINUE // signal for continue
)

var valueTypeNames = map[ValueType]string{
	VAL_INT:          "int",
	VAL_FLOAT:        "float",
	VAL_STRING:       "str",
	VAL_CHAR:         "char",
	VAL_BOOL:         "bool",
	VAL_NULL:         "null",
	VAL_ARRAY:        "array",
	VAL_MAP:          "map",
	VAL_STRUCT:       "struct",
	VAL_FUNCTION:     "function",
	VAL_BUILTIN:      "builtin",
	VAL_ENUM_VARIANT: "enum",
	VAL_RANGE:        "range",
	VAL_CHANNEL:      "channel",
	VAL_POINTER:      "pointer",
}

func (v ValueType) String() string {
	if name, ok := valueTypeNames[v]; ok {
		return name
	}
	return "unknown"
}

// Value represents a runtime value in Xuesos++.
type Value struct {
	Type ValueType
	// Only one of these is set depending on Type
	IntVal      int64
	FloatVal    float64
	StringVal   string
	CharVal     rune
	BoolVal     bool
	ArrayVal    []*Value
	MapVal      *MapValue
	StructVal   *StructValue
	FuncVal     *FuncValue
	BuiltinVal  BuiltinFunc
	EnumVal     *EnumVariantValue
	RangeVal    *RangeValue
	ChannelVal  chan *Value
	PointerVal  *PointerValue
	ReturnVal   *Value // wrapped value for VAL_RETURN
}

// BuiltinFunc is the signature for built-in functions.
type BuiltinFunc func(args []*Value) (*Value, error)

// FuncValue stores a user-defined function.
type FuncValue struct {
	Name       string
	ParamNames []string
	Body       interface{} // *parser.BlockStatement — avoid circular import, cast at eval time
	Closure    *Environment
	Receiver   string // struct name if this is a method, "" otherwise
}

// StructValue stores a struct instance.
type StructValue struct {
	TypeName string
	Fields   map[string]*Value
}

// EnumVariantValue stores an enum variant.
type EnumVariantValue struct {
	EnumName    string
	VariantName string
}

// MapValue stores a map/dictionary.
type MapValue struct {
	Pairs map[string]*Value
	Keys  []string // maintains insertion order
}

// RangeValue stores a range (start..end).
type RangeValue struct {
	Start int64
	End   int64
}

// PointerValue stores a pointer to a variable in an environment.
type PointerValue struct {
	Env  *Environment // the environment containing the variable
	Name string       // variable name being pointed to
}

// StructDef stores a struct definition (not an instance).
type StructDef struct {
	Name       string
	FieldNames []string
	FieldTypes []string
	Methods    map[string]*FuncValue
}

// Helpers for creating values

func IntVal(v int64) *Value       { return &Value{Type: VAL_INT, IntVal: v} }
func FloatVal(v float64) *Value   { return &Value{Type: VAL_FLOAT, FloatVal: v} }
func StringVal(v string) *Value   { return &Value{Type: VAL_STRING, StringVal: v} }
func CharValue(v rune) *Value     { return &Value{Type: VAL_CHAR, CharVal: v} }
func BoolValue(v bool) *Value     { return &Value{Type: VAL_BOOL, BoolVal: v} }
func NullValue() *Value           { return &Value{Type: VAL_NULL} }
func BreakSignal() *Value         { return &Value{Type: VAL_BREAK} }
func ContinueSignal() *Value      { return &Value{Type: VAL_CONTINUE} }

func ReturnValue(v *Value) *Value {
	return &Value{Type: VAL_RETURN, ReturnVal: v}
}

func ArrayValue(elements []*Value) *Value {
	return &Value{Type: VAL_ARRAY, ArrayVal: elements}
}

func MapVal(pairs map[string]*Value, keys []string) *Value {
	return &Value{Type: VAL_MAP, MapVal: &MapValue{Pairs: pairs, Keys: keys}}
}

func RangeVal(start, end int64) *Value {
	return &Value{Type: VAL_RANGE, RangeVal: &RangeValue{Start: start, End: end}}
}

// IsTruthy returns whether a value is considered truthy.
func (v *Value) IsTruthy() bool {
	switch v.Type {
	case VAL_BOOL:
		return v.BoolVal
	case VAL_NULL:
		return false
	case VAL_INT:
		return v.IntVal != 0
	case VAL_FLOAT:
		return v.FloatVal != 0
	case VAL_STRING:
		return v.StringVal != ""
	case VAL_ARRAY:
		return len(v.ArrayVal) > 0
	case VAL_MAP:
		return len(v.MapVal.Pairs) > 0
	default:
		return true
	}
}

// String returns a human-readable representation.
func (v *Value) String() string {
	switch v.Type {
	case VAL_INT:
		return fmt.Sprintf("%d", v.IntVal)
	case VAL_FLOAT:
		return fmt.Sprintf("%g", v.FloatVal)
	case VAL_STRING:
		return v.StringVal
	case VAL_CHAR:
		return string(v.CharVal)
	case VAL_BOOL:
		if v.BoolVal {
			return "xuitru"
		}
		return "xuinia"
	case VAL_NULL:
		return "xuinull"
	case VAL_MAP:
		var parts []string
		for _, k := range v.MapVal.Keys {
			val := v.MapVal.Pairs[k]
			parts = append(parts, fmt.Sprintf("%q: %s", k, val.Inspect()))
		}
		return "{" + strings.Join(parts, ", ") + "}"
	case VAL_ARRAY:
		var parts []string
		for _, elem := range v.ArrayVal {
			parts = append(parts, elem.Inspect())
		}
		return "[" + strings.Join(parts, ", ") + "]"
	case VAL_STRUCT:
		var parts []string
		for k, val := range v.StructVal.Fields {
			parts = append(parts, k+" = "+val.Inspect())
		}
		return v.StructVal.TypeName + " { " + strings.Join(parts, ", ") + " }"
	case VAL_FUNCTION:
		return "<xuen " + v.FuncVal.Name + ">"
	case VAL_BUILTIN:
		return "<builtin>"
	case VAL_ENUM_VARIANT:
		return v.EnumVal.EnumName + "." + v.EnumVal.VariantName
	case VAL_RANGE:
		return fmt.Sprintf("%d..%d", v.RangeVal.Start, v.RangeVal.End)
	case VAL_CHANNEL:
		return "<channel>"
	case VAL_POINTER:
		return fmt.Sprintf("<pointer to %s>", v.PointerVal.Name)
	default:
		return "<unknown>"
	}
}

// Inspect returns a debug-friendly representation (strings are quoted).
func (v *Value) Inspect() string {
	if v.Type == VAL_STRING {
		return fmt.Sprintf("%q", v.StringVal)
	}
	return v.String()
}
