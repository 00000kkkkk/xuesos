package interpreter

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/00000kkkkk/xusesosplusplus/parser"
)

// InterfaceDef stores an interface (trait) definition.
type InterfaceDef struct {
	Name    string
	Methods []string // required method names
}

// Interpreter evaluates an AST.
type Interpreter struct {
	globals       *Environment
	structDefs    map[string]*StructDef
	interfaceDefs map[string]*InterfaceDef
	output        []string // captured output for testing
	Imports       *ImportResolver
}

// New creates a new interpreter with built-in functions.
func New() *Interpreter {
	interp := &Interpreter{
		globals:       NewEnvironment(),
		structDefs:    make(map[string]*StructDef),
		interfaceDefs: make(map[string]*InterfaceDef),
	}
	interp.registerBuiltins()
	return interp
}

// Run executes a parsed program. If a main() function is defined, it is called automatically.
func (i *Interpreter) Run(program *parser.Program) error {
	for _, stmt := range program.Statements {
		val, err := i.execStatement(stmt, i.globals)
		if err != nil {
			return err
		}
		if val != nil && val.Type == VAL_RETURN {
			return nil
		}
	}

	// Auto-call main() if defined
	if mainVal, ok := i.globals.Get("main"); ok && mainVal.Type == VAL_FUNCTION {
		_, err := i.callUserFunc(mainVal.FuncVal, nil, &parser.CallExpression{})
		if err != nil {
			return err
		}
	}

	return nil
}

// RunLine executes a parsed program for REPL use: no auto-call of main(),
// returns the last expression value (if any) so the REPL can print it.
func (i *Interpreter) RunLine(program *parser.Program) (*Value, error) {
	var last *Value
	for _, stmt := range program.Statements {
		val, err := i.execStatement(stmt, i.globals)
		if err != nil {
			return nil, err
		}
		if val != nil && val.Type == VAL_RETURN {
			return val.ReturnVal, nil
		}
		// Only keep the value from expression statements (not assignments/decls which return nil)
		if _, ok := stmt.(*parser.ExpressionStatement); ok && val != nil {
			last = val
		}
	}
	return last, nil
}

// Output returns captured print output (for testing).
func (i *Interpreter) Output() []string {
	return i.output
}

// --- Built-in functions ---

func (i *Interpreter) registerBuiltins() {
	i.globals.Define("print", &Value{
		Type: VAL_BUILTIN,
		BuiltinVal: func(args []*Value) (*Value, error) {
			parts := make([]string, len(args))
			for idx, a := range args {
				parts[idx] = a.String()
			}
			line := strings.Join(parts, " ")
			fmt.Println(line)
			i.output = append(i.output, line)
			return NullValue(), nil
		},
	}, false)

	i.globals.Define("println", &Value{
		Type: VAL_BUILTIN,
		BuiltinVal: func(args []*Value) (*Value, error) {
			parts := make([]string, len(args))
			for idx, a := range args {
				parts[idx] = a.String()
			}
			line := strings.Join(parts, " ")
			fmt.Println(line)
			i.output = append(i.output, line)
			return NullValue(), nil
		},
	}, false)

	i.globals.Define("len", &Value{
		Type: VAL_BUILTIN,
		BuiltinVal: func(args []*Value) (*Value, error) {
			if len(args) != 1 {
				return nil, fmt.Errorf("len() takes exactly 1 argument, got %d", len(args))
			}
			switch args[0].Type {
			case VAL_STRING:
				return IntVal(int64(len([]rune(args[0].StringVal)))), nil
			case VAL_ARRAY:
				return IntVal(int64(len(args[0].ArrayVal))), nil
			case VAL_MAP:
				return IntVal(int64(len(args[0].MapVal.Pairs))), nil
			default:
				return nil, fmt.Errorf("len() not supported for type %s", args[0].Type)
			}
		},
	}, false)

	i.globals.Define("type", &Value{
		Type: VAL_BUILTIN,
		BuiltinVal: func(args []*Value) (*Value, error) {
			if len(args) != 1 {
				return nil, fmt.Errorf("type() takes exactly 1 argument, got %d", len(args))
			}
			return StringVal(args[0].Type.String()), nil
		},
	}, false)

	i.globals.Define("to_str", &Value{
		Type: VAL_BUILTIN,
		BuiltinVal: func(args []*Value) (*Value, error) {
			if len(args) != 1 {
				return nil, fmt.Errorf("str() takes exactly 1 argument, got %d", len(args))
			}
			return StringVal(args[0].String()), nil
		},
	}, false)

	i.globals.Define("to_int", &Value{
		Type: VAL_BUILTIN,
		BuiltinVal: func(args []*Value) (*Value, error) {
			if len(args) != 1 {
				return nil, fmt.Errorf("int() takes exactly 1 argument, got %d", len(args))
			}
			switch args[0].Type {
			case VAL_INT:
				return args[0], nil
			case VAL_FLOAT:
				return IntVal(int64(args[0].FloatVal)), nil
			case VAL_BOOL:
				if args[0].BoolVal {
					return IntVal(1), nil
				}
				return IntVal(0), nil
			default:
				return nil, fmt.Errorf("cannot convert %s to int", args[0].Type)
			}
		},
	}, false)

	i.globals.Define("to_float", &Value{
		Type: VAL_BUILTIN,
		BuiltinVal: func(args []*Value) (*Value, error) {
			if len(args) != 1 {
				return nil, fmt.Errorf("float() takes exactly 1 argument, got %d", len(args))
			}
			switch args[0].Type {
			case VAL_FLOAT:
				return args[0], nil
			case VAL_INT:
				return FloatVal(float64(args[0].IntVal)), nil
			default:
				return nil, fmt.Errorf("cannot convert %s to float", args[0].Type)
			}
		},
	}, false)

	i.globals.Define("sqrt", &Value{
		Type: VAL_BUILTIN,
		BuiltinVal: func(args []*Value) (*Value, error) {
			if len(args) != 1 {
				return nil, fmt.Errorf("sqrt() takes exactly 1 argument, got %d", len(args))
			}
			var v float64
			switch args[0].Type {
			case VAL_FLOAT:
				v = args[0].FloatVal
			case VAL_INT:
				v = float64(args[0].IntVal)
			default:
				return nil, fmt.Errorf("sqrt() requires numeric argument, got %s", args[0].Type)
			}
			return FloatVal(math.Sqrt(v)), nil
		},
	}, false)

	i.globals.Define("append", &Value{
		Type: VAL_BUILTIN,
		BuiltinVal: func(args []*Value) (*Value, error) {
			if len(args) < 2 {
				return nil, fmt.Errorf("append() requires at least 2 arguments")
			}
			if args[0].Type != VAL_ARRAY {
				return nil, fmt.Errorf("append() first argument must be an array, got %s", args[0].Type)
			}
			newArr := make([]*Value, len(args[0].ArrayVal))
			copy(newArr, args[0].ArrayVal)
			newArr = append(newArr, args[1:]...)
			return ArrayValue(newArr), nil
		},
	}, false)

	i.globals.Define("input", &Value{Type: VAL_BUILTIN, BuiltinVal: func(args []*Value) (*Value, error) {
		if len(args) > 0 { fmt.Print(args[0].String()) }
		reader := bufio.NewReader(os.Stdin)
		line, _ := reader.ReadString('\n')
		return StringVal(strings.TrimRight(line, "\r\n")), nil
	}}, false)

	i.globals.Define("exit", &Value{Type: VAL_BUILTIN, BuiltinVal: func(args []*Value) (*Value, error) {
		code := 0
		if len(args) > 0 && args[0].Type == VAL_INT { code = int(args[0].IntVal) }
		os.Exit(code)
		return NullValue(), nil
	}}, false)

	i.globals.Define("abs", &Value{Type: VAL_BUILTIN, BuiltinVal: func(args []*Value) (*Value, error) {
		if len(args) != 1 { return nil, fmt.Errorf("abs() takes 1 argument") }
		switch args[0].Type {
		case VAL_INT:
			v := args[0].IntVal; if v < 0 { v = -v }; return IntVal(v), nil
		case VAL_FLOAT:
			return FloatVal(math.Abs(args[0].FloatVal)), nil
		default:
			return nil, fmt.Errorf("abs() requires numeric argument")
		}
	}}, false)

	i.globals.Define("max", &Value{Type: VAL_BUILTIN, BuiltinVal: func(args []*Value) (*Value, error) {
		if len(args) != 2 { return nil, fmt.Errorf("max() takes 2 arguments") }
		a, b := args[0], args[1]
		if a.Type == VAL_INT && b.Type == VAL_INT {
			if a.IntVal > b.IntVal { return a, nil }; return b, nil
		}
		af, bf, ok := toFloats(a, b)
		if !ok { return nil, fmt.Errorf("max() requires numeric arguments") }
		if af > bf { return FloatVal(af), nil }; return FloatVal(bf), nil
	}}, false)

	i.globals.Define("min", &Value{Type: VAL_BUILTIN, BuiltinVal: func(args []*Value) (*Value, error) {
		if len(args) != 2 { return nil, fmt.Errorf("min() takes 2 arguments") }
		a, b := args[0], args[1]
		if a.Type == VAL_INT && b.Type == VAL_INT {
			if a.IntVal < b.IntVal { return a, nil }; return b, nil
		}
		af, bf, ok := toFloats(a, b)
		if !ok { return nil, fmt.Errorf("min() requires numeric arguments") }
		if af < bf { return FloatVal(af), nil }; return FloatVal(bf), nil
	}}, false)

	// push(arr, val) - append to array IN PLACE
	i.globals.Define("push", &Value{
		Type: VAL_BUILTIN,
		BuiltinVal: func(args []*Value) (*Value, error) {
			if len(args) != 2 {
				return nil, fmt.Errorf("push() takes exactly 2 arguments, got %d", len(args))
			}
			if args[0].Type != VAL_ARRAY {
				return nil, fmt.Errorf("push() first argument must be an array, got %s", args[0].Type)
			}
			args[0].ArrayVal = append(args[0].ArrayVal, args[1])
			return NullValue(), nil
		},
	}, false)

	i.globals.Define("contains", &Value{Type: VAL_BUILTIN, BuiltinVal: func(args []*Value) (*Value, error) {
		if len(args) != 2 || args[0].Type != VAL_STRING || args[1].Type != VAL_STRING {
			return nil, fmt.Errorf("contains() takes 2 string arguments")
		}
		return BoolValue(strings.Contains(args[0].StringVal, args[1].StringVal)), nil
	}}, false)

	i.globals.Define("split", &Value{Type: VAL_BUILTIN, BuiltinVal: func(args []*Value) (*Value, error) {
		if len(args) != 2 || args[0].Type != VAL_STRING || args[1].Type != VAL_STRING {
			return nil, fmt.Errorf("split() takes 2 string arguments")
		}
		parts := strings.Split(args[0].StringVal, args[1].StringVal)
		elems := make([]*Value, len(parts))
		for idx, p := range parts { elems[idx] = StringVal(p) }
		return ArrayValue(elems), nil
	}}, false)

	i.globals.Define("trim", &Value{Type: VAL_BUILTIN, BuiltinVal: func(args []*Value) (*Value, error) {
		if len(args) != 1 || args[0].Type != VAL_STRING { return nil, fmt.Errorf("trim() takes 1 string argument") }
		return StringVal(strings.TrimSpace(args[0].StringVal)), nil
	}}, false)

	i.globals.Define("replace", &Value{Type: VAL_BUILTIN, BuiltinVal: func(args []*Value) (*Value, error) {
		if len(args) != 3 || args[0].Type != VAL_STRING || args[1].Type != VAL_STRING || args[2].Type != VAL_STRING {
			return nil, fmt.Errorf("replace() takes 3 string arguments")
		}
		return StringVal(strings.ReplaceAll(args[0].StringVal, args[1].StringVal, args[2].StringVal)), nil
	}}, false)

	i.globals.Define("upper", &Value{Type: VAL_BUILTIN, BuiltinVal: func(args []*Value) (*Value, error) {
		if len(args) != 1 || args[0].Type != VAL_STRING { return nil, fmt.Errorf("upper() takes 1 string argument") }
		return StringVal(strings.ToUpper(args[0].StringVal)), nil
	}}, false)

	i.globals.Define("lower", &Value{Type: VAL_BUILTIN, BuiltinVal: func(args []*Value) (*Value, error) {
		if len(args) != 1 || args[0].Type != VAL_STRING { return nil, fmt.Errorf("lower() takes 1 string argument") }
		return StringVal(strings.ToLower(args[0].StringVal)), nil
	}}, false)

	i.globals.Define("starts_with", &Value{Type: VAL_BUILTIN, BuiltinVal: func(args []*Value) (*Value, error) {
		if len(args) != 2 || args[0].Type != VAL_STRING || args[1].Type != VAL_STRING {
			return nil, fmt.Errorf("starts_with() takes 2 string arguments")
		}
		return BoolValue(strings.HasPrefix(args[0].StringVal, args[1].StringVal)), nil
	}}, false)

	i.globals.Define("ends_with", &Value{Type: VAL_BUILTIN, BuiltinVal: func(args []*Value) (*Value, error) {
		if len(args) != 2 || args[0].Type != VAL_STRING || args[1].Type != VAL_STRING {
			return nil, fmt.Errorf("ends_with() takes 2 string arguments")
		}
		return BoolValue(strings.HasSuffix(args[0].StringVal, args[1].StringVal)), nil
	}}, false)

	i.globals.Define("join", &Value{Type: VAL_BUILTIN, BuiltinVal: func(args []*Value) (*Value, error) {
		if len(args) != 2 || args[0].Type != VAL_ARRAY || args[1].Type != VAL_STRING {
			return nil, fmt.Errorf("join() takes array and string separator")
		}
		parts := make([]string, len(args[0].ArrayVal))
		for idx, v := range args[0].ArrayVal { parts[idx] = v.String() }
		return StringVal(strings.Join(parts, args[1].StringVal)), nil
	}}, false)

	// range_arr(start, end) - create array from range
	i.globals.Define("range_arr", &Value{
		Type: VAL_BUILTIN,
		BuiltinVal: func(args []*Value) (*Value, error) {
			if len(args) != 2 {
				return nil, fmt.Errorf("range_arr() takes exactly 2 arguments, got %d", len(args))
			}
			if args[0].Type != VAL_INT || args[1].Type != VAL_INT {
				return nil, fmt.Errorf("range_arr() requires int arguments, got %s and %s", args[0].Type, args[1].Type)
			}
			start := args[0].IntVal
			end := args[1].IntVal
			var elems []*Value
			for v := start; v < end; v++ {
				elems = append(elems, IntVal(v))
			}
			return ArrayValue(elems), nil
		},
	}, false)

	// slice(arr_or_str, start, end) - get sub-array or sub-string
	i.globals.Define("slice", &Value{
		Type: VAL_BUILTIN,
		BuiltinVal: func(args []*Value) (*Value, error) {
			if len(args) != 3 {
				return nil, fmt.Errorf("slice() takes exactly 3 arguments, got %d", len(args))
			}
			if args[1].Type != VAL_INT || args[2].Type != VAL_INT {
				return nil, fmt.Errorf("slice() start and end must be int")
			}
			start := int(args[1].IntVal)
			end := int(args[2].IntVal)
			switch args[0].Type {
			case VAL_ARRAY:
				arr := args[0].ArrayVal
				if start < 0 {
					start = 0
				}
				if end > len(arr) {
					end = len(arr)
				}
				if start > end {
					return ArrayValue([]*Value{}), nil
				}
				sliced := make([]*Value, end-start)
				copy(sliced, arr[start:end])
				return ArrayValue(sliced), nil
			case VAL_STRING:
				runes := []rune(args[0].StringVal)
				if start < 0 {
					start = 0
				}
				if end > len(runes) {
					end = len(runes)
				}
				if start > end {
					return StringVal(""), nil
				}
				return StringVal(string(runes[start:end])), nil
			default:
				return nil, fmt.Errorf("slice() first argument must be array or string, got %s", args[0].Type)
			}
		},
	}, false)

	// Map built-ins
	i.globals.Define("has_key", &Value{Type: VAL_BUILTIN, BuiltinVal: func(args []*Value) (*Value, error) {
		if len(args) != 2 || args[0].Type != VAL_MAP {
			return nil, fmt.Errorf("has_key() takes a map and a key")
		}
		_, ok := args[0].MapVal.Pairs[args[1].String()]
		return BoolValue(ok), nil
	}}, false)

	i.globals.Define("keys", &Value{Type: VAL_BUILTIN, BuiltinVal: func(args []*Value) (*Value, error) {
		if len(args) != 1 || args[0].Type != VAL_MAP {
			return nil, fmt.Errorf("keys() takes 1 map argument")
		}
		elems := make([]*Value, len(args[0].MapVal.Keys))
		for idx, k := range args[0].MapVal.Keys {
			elems[idx] = StringVal(k)
		}
		return ArrayValue(elems), nil
	}}, false)

	i.globals.Define("values", &Value{Type: VAL_BUILTIN, BuiltinVal: func(args []*Value) (*Value, error) {
		if len(args) != 1 || args[0].Type != VAL_MAP {
			return nil, fmt.Errorf("values() takes 1 map argument")
		}
		elems := make([]*Value, len(args[0].MapVal.Keys))
		for idx, k := range args[0].MapVal.Keys {
			elems[idx] = args[0].MapVal.Pairs[k]
		}
		return ArrayValue(elems), nil
	}}, false)

	i.globals.Define("delete", &Value{Type: VAL_BUILTIN, BuiltinVal: func(args []*Value) (*Value, error) {
		if len(args) != 2 || args[0].Type != VAL_MAP {
			return nil, fmt.Errorf("delete() takes a map and a key")
		}
		keyStr := args[1].String()
		delete(args[0].MapVal.Pairs, keyStr)
		newKeys := make([]string, 0, len(args[0].MapVal.Keys))
		for _, k := range args[0].MapVal.Keys {
			if k != keyStr { newKeys = append(newKeys, k) }
		}
		args[0].MapVal.Keys = newKeys
		return NullValue(), nil
	}}, false)

	// Stdlib: math
	i.globals.Define("math_pi", &Value{Type: VAL_BUILTIN, BuiltinVal: func(args []*Value) (*Value, error) {
		return FloatVal(math.Pi), nil
	}}, false)
	i.globals.Define("math_e", &Value{Type: VAL_BUILTIN, BuiltinVal: func(args []*Value) (*Value, error) {
		return FloatVal(math.E), nil
	}}, false)
	i.globals.Define("math_floor", &Value{Type: VAL_BUILTIN, BuiltinVal: func(args []*Value) (*Value, error) {
		if len(args) != 1 { return nil, fmt.Errorf("math_floor() takes 1 argument") }
		f, _ := toFloat(args[0])
		return IntVal(int64(math.Floor(f))), nil
	}}, false)
	i.globals.Define("math_ceil", &Value{Type: VAL_BUILTIN, BuiltinVal: func(args []*Value) (*Value, error) {
		if len(args) != 1 { return nil, fmt.Errorf("math_ceil() takes 1 argument") }
		f, _ := toFloat(args[0])
		return IntVal(int64(math.Ceil(f))), nil
	}}, false)
	i.globals.Define("math_pow", &Value{Type: VAL_BUILTIN, BuiltinVal: func(args []*Value) (*Value, error) {
		if len(args) != 2 { return nil, fmt.Errorf("math_pow() takes 2 arguments") }
		base, _ := toFloat(args[0])
		exp, _ := toFloat(args[1])
		return FloatVal(math.Pow(base, exp)), nil
	}}, false)
	i.globals.Define("math_round", &Value{Type: VAL_BUILTIN, BuiltinVal: func(args []*Value) (*Value, error) {
		if len(args) != 1 { return nil, fmt.Errorf("math_round() takes 1 argument") }
		f, _ := toFloat(args[0])
		return IntVal(int64(math.Round(f))), nil
	}}, false)
	i.globals.Define("math_sin", &Value{Type: VAL_BUILTIN, BuiltinVal: func(args []*Value) (*Value, error) {
		if len(args) != 1 { return nil, fmt.Errorf("math_sin() takes 1 argument") }
		f, _ := toFloat(args[0])
		return FloatVal(math.Sin(f)), nil
	}}, false)
	i.globals.Define("math_cos", &Value{Type: VAL_BUILTIN, BuiltinVal: func(args []*Value) (*Value, error) {
		if len(args) != 1 { return nil, fmt.Errorf("math_cos() takes 1 argument") }
		f, _ := toFloat(args[0])
		return FloatVal(math.Cos(f)), nil
	}}, false)
	i.globals.Define("math_log", &Value{Type: VAL_BUILTIN, BuiltinVal: func(args []*Value) (*Value, error) {
		if len(args) != 1 { return nil, fmt.Errorf("math_log() takes 1 argument") }
		f, _ := toFloat(args[0])
		return FloatVal(math.Log(f)), nil
	}}, false)

	// Stdlib: os
	i.globals.Define("os_args", &Value{Type: VAL_BUILTIN, BuiltinVal: func(args []*Value) (*Value, error) {
		elems := make([]*Value, len(os.Args))
		for idx, a := range os.Args {
			elems[idx] = StringVal(a)
		}
		return ArrayValue(elems), nil
	}}, false)
	i.globals.Define("os_getenv", &Value{Type: VAL_BUILTIN, BuiltinVal: func(args []*Value) (*Value, error) {
		if len(args) != 1 || args[0].Type != VAL_STRING {
			return nil, fmt.Errorf("os_getenv() takes 1 string argument")
		}
		return StringVal(os.Getenv(args[0].StringVal)), nil
	}}, false)

	// File I/O
	i.globals.Define("io_read_file", &Value{Type: VAL_BUILTIN, BuiltinVal: func(args []*Value) (*Value, error) {
		if len(args) != 1 || args[0].Type != VAL_STRING {
			return nil, fmt.Errorf("io_read_file() takes 1 string argument")
		}
		data, err := os.ReadFile(args[0].StringVal)
		if err != nil {
			return nil, fmt.Errorf("io_read_file: %s", err)
		}
		return StringVal(string(data)), nil
	}}, false)

	i.globals.Define("io_write_file", &Value{Type: VAL_BUILTIN, BuiltinVal: func(args []*Value) (*Value, error) {
		if len(args) != 2 || args[0].Type != VAL_STRING || args[1].Type != VAL_STRING {
			return nil, fmt.Errorf("io_write_file() takes filename and content strings")
		}
		if err := os.WriteFile(args[0].StringVal, []byte(args[1].StringVal), 0644); err != nil {
			return nil, fmt.Errorf("io_write_file: %s", err)
		}
		return NullValue(), nil
	}}, false)

	i.globals.Define("char_at", &Value{Type: VAL_BUILTIN, BuiltinVal: func(args []*Value) (*Value, error) {
		if len(args) != 2 || args[0].Type != VAL_STRING || args[1].Type != VAL_INT {
			return nil, fmt.Errorf("char_at() takes string and int")
		}
		idx := int(args[1].IntVal)
		if idx < 0 || idx >= len(args[0].StringVal) {
			return StringVal(""), nil
		}
		return StringVal(string(args[0].StringVal[idx])), nil
	}}, false)

	i.globals.Define("char_code", &Value{Type: VAL_BUILTIN, BuiltinVal: func(args []*Value) (*Value, error) {
		if len(args) != 1 || args[0].Type != VAL_STRING {
			return nil, fmt.Errorf("char_code() takes 1 string argument")
		}
		if len(args[0].StringVal) == 0 {
			return IntVal(0), nil
		}
		return IntVal(int64(args[0].StringVal[0])), nil
	}}, false)

	i.globals.Define("from_char_code", &Value{Type: VAL_BUILTIN, BuiltinVal: func(args []*Value) (*Value, error) {
		if len(args) != 1 || args[0].Type != VAL_INT {
			return nil, fmt.Errorf("from_char_code() takes 1 int argument")
		}
		return StringVal(string(rune(args[0].IntVal))), nil
	}}, false)

	i.globals.Define("substr", &Value{Type: VAL_BUILTIN, BuiltinVal: func(args []*Value) (*Value, error) {
		if len(args) != 3 || args[0].Type != VAL_STRING || args[1].Type != VAL_INT || args[2].Type != VAL_INT {
			return nil, fmt.Errorf("substr() takes string, start, length")
		}
		s := args[0].StringVal
		start := int(args[1].IntVal)
		length := int(args[2].IntVal)
		if start < 0 { start = 0 }
		if start >= len(s) { return StringVal(""), nil }
		end := start + length
		if end > len(s) { end = len(s) }
		return StringVal(s[start:end]), nil
	}}, false)

	i.globals.Define("index_of", &Value{Type: VAL_BUILTIN, BuiltinVal: func(args []*Value) (*Value, error) {
		if len(args) != 2 || args[0].Type != VAL_STRING || args[1].Type != VAL_STRING {
			return nil, fmt.Errorf("index_of() takes 2 string arguments")
		}
		return IntVal(int64(strings.Index(args[0].StringVal, args[1].StringVal))), nil
	}}, false)

	i.globals.Define("string_len", &Value{Type: VAL_BUILTIN, BuiltinVal: func(args []*Value) (*Value, error) {
		if len(args) != 1 || args[0].Type != VAL_STRING {
			return nil, fmt.Errorf("string_len() takes 1 string argument")
		}
		return IntVal(int64(len(args[0].StringVal))), nil
	}}, false)

	i.globals.Define("panic", &Value{Type: VAL_BUILTIN, BuiltinVal: func(args []*Value) (*Value, error) {
		msg := "panic"
		if len(args) > 0 { msg = args[0].String() }
		fmt.Fprintf(os.Stderr, "panic: %s\n", msg)
		os.Exit(1)
		return NullValue(), nil
	}}, false)

	// --- Concurrency built-ins ---

	// spawn(func) — run function in a goroutine
	i.globals.Define("spawn", &Value{Type: VAL_BUILTIN, BuiltinVal: func(args []*Value) (*Value, error) {
		if len(args) != 1 || args[0].Type != VAL_FUNCTION {
			return nil, fmt.Errorf("spawn() takes 1 function argument")
		}
		fn := args[0]
		go func() {
			body := fn.FuncVal.Body.(*parser.BlockStatement)
			env := NewEnclosedEnvironment(fn.FuncVal.Closure)
			i.execBlock(body, env)
		}()
		return NullValue(), nil
	}}, false)

	// sleep(ms) — sleep for milliseconds
	i.globals.Define("sleep", &Value{Type: VAL_BUILTIN, BuiltinVal: func(args []*Value) (*Value, error) {
		if len(args) != 1 || args[0].Type != VAL_INT {
			return nil, fmt.Errorf("sleep() takes 1 int argument (milliseconds)")
		}
		time.Sleep(time.Duration(args[0].IntVal) * time.Millisecond)
		return NullValue(), nil
	}}, false)

	// wait(ms) — alias for sleep
	i.globals.Define("wait", &Value{Type: VAL_BUILTIN, BuiltinVal: func(args []*Value) (*Value, error) {
		if len(args) != 1 || args[0].Type != VAL_INT {
			return nil, fmt.Errorf("wait() takes 1 int argument (milliseconds)")
		}
		time.Sleep(time.Duration(args[0].IntVal) * time.Millisecond)
		return NullValue(), nil
	}}, false)

	// channel() — create a new channel
	i.globals.Define("channel", &Value{Type: VAL_BUILTIN, BuiltinVal: func(args []*Value) (*Value, error) {
		size := 0
		if len(args) > 0 && args[0].Type == VAL_INT {
			size = int(args[0].IntVal)
		}
		ch := make(chan *Value, size)
		return &Value{Type: VAL_CHANNEL, ChannelVal: ch}, nil
	}}, false)

	// send(channel, value) — send value to channel
	i.globals.Define("send", &Value{Type: VAL_BUILTIN, BuiltinVal: func(args []*Value) (*Value, error) {
		if len(args) != 2 || args[0].Type != VAL_CHANNEL {
			return nil, fmt.Errorf("send() takes a channel and a value")
		}
		args[0].ChannelVal <- args[1]
		return NullValue(), nil
	}}, false)

	// recv(channel) — receive value from channel
	i.globals.Define("recv", &Value{Type: VAL_BUILTIN, BuiltinVal: func(args []*Value) (*Value, error) {
		if len(args) != 1 || args[0].Type != VAL_CHANNEL {
			return nil, fmt.Errorf("recv() takes 1 channel argument")
		}
		val := <-args[0].ChannelVal
		return val, nil
	}}, false)

	// --- Memory management built-ins ---

	// alloc(size) — allocate array of given size filled with 0
	i.globals.Define("alloc", &Value{Type: VAL_BUILTIN, BuiltinVal: func(args []*Value) (*Value, error) {
		if len(args) != 1 || args[0].Type != VAL_INT {
			return nil, fmt.Errorf("alloc() takes 1 int argument")
		}
		size := int(args[0].IntVal)
		elems := make([]*Value, size)
		for idx := range elems {
			elems[idx] = IntVal(0)
		}
		return ArrayValue(elems), nil
	}}, false)

	// sizeof(value) — return size info
	i.globals.Define("sizeof", &Value{Type: VAL_BUILTIN, BuiltinVal: func(args []*Value) (*Value, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("sizeof() takes 1 argument")
		}
		switch args[0].Type {
		case VAL_INT:
			return IntVal(8), nil
		case VAL_FLOAT:
			return IntVal(8), nil
		case VAL_BOOL:
			return IntVal(1), nil
		case VAL_STRING:
			return IntVal(int64(len(args[0].StringVal))), nil
		case VAL_ARRAY:
			return IntVal(int64(len(args[0].ArrayVal))), nil
		default:
			return IntVal(0), nil
		}
	}}, false)

	// --- HTTP Client built-ins ---

	// http_get(url) — returns response body as string
	i.globals.Define("http_get", &Value{Type: VAL_BUILTIN, BuiltinVal: func(args []*Value) (*Value, error) {
		if len(args) != 1 || args[0].Type != VAL_STRING {
			return nil, fmt.Errorf("http_get() takes 1 string argument")
		}
		resp, err := http.Get(args[0].StringVal)
		if err != nil {
			return nil, fmt.Errorf("http_get: %s", err)
		}
		defer resp.Body.Close()
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("http_get: %s", err)
		}
		return StringVal(string(body)), nil
	}}, false)

	// http_status(url) — returns HTTP status code as int
	i.globals.Define("http_status", &Value{Type: VAL_BUILTIN, BuiltinVal: func(args []*Value) (*Value, error) {
		if len(args) != 1 || args[0].Type != VAL_STRING {
			return nil, fmt.Errorf("http_status() takes 1 string argument")
		}
		resp, err := http.Get(args[0].StringVal)
		if err != nil {
			return nil, fmt.Errorf("http_status: %s", err)
		}
		defer resp.Body.Close()
		return IntVal(int64(resp.StatusCode)), nil
	}}, false)

	// --- JSON built-ins ---

	// json_parse(str) — parse JSON string into Xuesos++ value
	i.globals.Define("json_parse", &Value{Type: VAL_BUILTIN, BuiltinVal: func(args []*Value) (*Value, error) {
		if len(args) != 1 || args[0].Type != VAL_STRING {
			return nil, fmt.Errorf("json_parse() takes 1 string argument")
		}
		var raw interface{}
		if err := json.Unmarshal([]byte(args[0].StringVal), &raw); err != nil {
			return nil, fmt.Errorf("json_parse: %s", err)
		}
		return goToXuesos(raw), nil
	}}, false)

	// json_stringify(value) — convert Xuesos++ value to JSON string
	i.globals.Define("json_stringify", &Value{Type: VAL_BUILTIN, BuiltinVal: func(args []*Value) (*Value, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("json_stringify() takes 1 argument")
		}
		result := xuesosToGo(args[0])
		data, err := json.Marshal(result)
		if err != nil {
			return nil, fmt.Errorf("json_stringify: %s", err)
		}
		return StringVal(string(data)), nil
	}}, false)

	// --- Filesystem built-ins ---

	// file_exists(path) — check if file exists
	i.globals.Define("file_exists", &Value{Type: VAL_BUILTIN, BuiltinVal: func(args []*Value) (*Value, error) {
		if len(args) != 1 || args[0].Type != VAL_STRING {
			return nil, fmt.Errorf("file_exists() takes 1 string argument")
		}
		_, err := os.Stat(args[0].StringVal)
		return BoolValue(err == nil), nil
	}}, false)

	// list_dir(path) — list files in directory
	i.globals.Define("list_dir", &Value{Type: VAL_BUILTIN, BuiltinVal: func(args []*Value) (*Value, error) {
		if len(args) != 1 || args[0].Type != VAL_STRING {
			return nil, fmt.Errorf("list_dir() takes 1 string argument")
		}
		entries, err := os.ReadDir(args[0].StringVal)
		if err != nil {
			return nil, fmt.Errorf("list_dir: %s", err)
		}
		elems := make([]*Value, len(entries))
		for idx, e := range entries {
			elems[idx] = StringVal(e.Name())
		}
		return ArrayValue(elems), nil
	}}, false)

	// mkdir(path) — create directory
	i.globals.Define("mkdir", &Value{Type: VAL_BUILTIN, BuiltinVal: func(args []*Value) (*Value, error) {
		if len(args) != 1 || args[0].Type != VAL_STRING {
			return nil, fmt.Errorf("mkdir() takes 1 string argument")
		}
		return NullValue(), os.MkdirAll(args[0].StringVal, 0755)
	}}, false)

	// remove(path) — delete file or empty directory
	i.globals.Define("remove", &Value{Type: VAL_BUILTIN, BuiltinVal: func(args []*Value) (*Value, error) {
		if len(args) != 1 || args[0].Type != VAL_STRING {
			return nil, fmt.Errorf("remove() takes 1 string argument")
		}
		return NullValue(), os.Remove(args[0].StringVal)
	}}, false)

	// path_join(parts...) — join path components
	i.globals.Define("path_join", &Value{Type: VAL_BUILTIN, BuiltinVal: func(args []*Value) (*Value, error) {
		parts := make([]string, len(args))
		for idx, a := range args {
			parts[idx] = a.String()
		}
		return StringVal(filepath.Join(parts...)), nil
	}}, false)

	// implements(value, interfaceName) — check if a struct implements an interface
	i.globals.Define("implements", &Value{Type: VAL_BUILTIN, BuiltinVal: func(args []*Value) (*Value, error) {
		if len(args) != 2 {
			return nil, fmt.Errorf("implements() takes 2 arguments (value, interfaceName)")
		}
		if args[1].Type != VAL_STRING {
			return nil, fmt.Errorf("implements() second argument must be a string (interface name)")
		}
		ifaceName := args[1].StringVal
		ifaceDef, ok := i.interfaceDefs[ifaceName]
		if !ok {
			return nil, fmt.Errorf("implements(): unknown interface %q", ifaceName)
		}
		if args[0].Type != VAL_STRUCT {
			return BoolValue(false), nil
		}
		structName := args[0].StructVal.TypeName
		structDef, ok := i.structDefs[structName]
		if !ok {
			return BoolValue(false), nil
		}
		for _, methodName := range ifaceDef.Methods {
			if _, exists := structDef.Methods[methodName]; !exists {
				return BoolValue(false), nil
			}
		}
		return BoolValue(true), nil
	}}, false)

	// cast_int(value) — force cast to int
	i.globals.Define("cast_int", &Value{Type: VAL_BUILTIN, BuiltinVal: func(args []*Value) (*Value, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("cast_int() takes 1 argument")
		}
		switch args[0].Type {
		case VAL_INT:
			return args[0], nil
		case VAL_FLOAT:
			return IntVal(int64(args[0].FloatVal)), nil
		case VAL_BOOL:
			if args[0].BoolVal {
				return IntVal(1), nil
			}
			return IntVal(0), nil
		case VAL_STRING:
			v, err := strconv.ParseInt(args[0].StringVal, 10, 64)
			if err != nil {
				return nil, fmt.Errorf("cannot cast %q to int", args[0].StringVal)
			}
			return IntVal(v), nil
		default:
			return nil, fmt.Errorf("cannot cast %s to int", args[0].Type)
		}
	}}, false)

	// cast_float(value) — force cast to float
	i.globals.Define("cast_float", &Value{Type: VAL_BUILTIN, BuiltinVal: func(args []*Value) (*Value, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("cast_float() takes 1 argument")
		}
		switch args[0].Type {
		case VAL_FLOAT:
			return args[0], nil
		case VAL_INT:
			return FloatVal(float64(args[0].IntVal)), nil
		case VAL_STRING:
			v, err := strconv.ParseFloat(args[0].StringVal, 64)
			if err != nil {
				return nil, fmt.Errorf("cannot cast %q to float", args[0].StringVal)
			}
			return FloatVal(v), nil
		default:
			return nil, fmt.Errorf("cannot cast %s to float", args[0].Type)
		}
	}}, false)

	// cast_str(value) — force cast to string
	i.globals.Define("cast_str", &Value{Type: VAL_BUILTIN, BuiltinVal: func(args []*Value) (*Value, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("cast_str() takes 1 argument")
		}
		return StringVal(args[0].String()), nil
	}}, false)

	// cast_bool(value) — force cast to bool
	i.globals.Define("cast_bool", &Value{Type: VAL_BUILTIN, BuiltinVal: func(args []*Value) (*Value, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("cast_bool() takes 1 argument")
		}
		return BoolValue(args[0].IsTruthy()), nil
	}}, false)
}

func goToXuesos(v interface{}) *Value {
	switch val := v.(type) {
	case nil:
		return NullValue()
	case bool:
		return BoolValue(val)
	case float64:
		if val == float64(int64(val)) {
			return IntVal(int64(val))
		}
		return FloatVal(val)
	case string:
		return StringVal(val)
	case []interface{}:
		elems := make([]*Value, len(val))
		for i, elem := range val {
			elems[i] = goToXuesos(elem)
		}
		return ArrayValue(elems)
	case map[string]interface{}:
		pairs := make(map[string]*Value)
		keys := make([]string, 0, len(val))
		for k, v := range val {
			pairs[k] = goToXuesos(v)
			keys = append(keys, k)
		}
		return MapVal(pairs, keys)
	default:
		return StringVal(fmt.Sprintf("%v", val))
	}
}

func xuesosToGo(v *Value) interface{} {
	switch v.Type {
	case VAL_INT:
		return v.IntVal
	case VAL_FLOAT:
		return v.FloatVal
	case VAL_STRING:
		return v.StringVal
	case VAL_BOOL:
		return v.BoolVal
	case VAL_NULL:
		return nil
	case VAL_ARRAY:
		result := make([]interface{}, len(v.ArrayVal))
		for i, elem := range v.ArrayVal {
			result[i] = xuesosToGo(elem)
		}
		return result
	case VAL_MAP:
		result := make(map[string]interface{})
		for k, val := range v.MapVal.Pairs {
			result[k] = xuesosToGo(val)
		}
		return result
	default:
		return v.String()
	}
}

func toFloat(v *Value) (float64, bool) {
	switch v.Type {
	case VAL_FLOAT: return v.FloatVal, true
	case VAL_INT: return float64(v.IntVal), true
	default: return 0, false
	}
}

// --- Statement execution ---

func (i *Interpreter) execStatement(stmt parser.Statement, env *Environment) (*Value, error) {
	switch s := stmt.(type) {
	case *parser.XuetStatement:
		return i.execXuet(s, env)
	case *parser.XuiarStatement:
		return i.execXuiar(s, env)
	case *parser.AssignStatement:
		return i.execAssign(s, env)
	case *parser.XuenStatement:
		return i.execXuen(s, env)
	case *parser.XueturnStatement:
		return i.execXueturn(s, env)
	case *parser.XueakStatement:
		return BreakSignal(), nil
	case *parser.XuitinueStatement:
		return ContinueSignal(), nil
	case *parser.XuifStatement:
		return i.execXuif(s, env)
	case *parser.XuiorStatement:
		return i.execXuior(s, env)
	case *parser.XuileStatement:
		return i.execXuile(s, env)
	case *parser.XuiructStatement:
		return i.execXuiruct(s)
	case *parser.XuimplStatement:
		return i.execXuimpl(s, env)
	case *parser.XuenumStatement:
		return i.execXuenum(s, env)
	case *parser.XuimportStatement:
		if i.Imports != nil {
			if err := i.Imports.Resolve(s.Path, i); err != nil {
				return nil, err
			}
		}
		return nil, nil
	case *parser.XuiatchStatement:
		return i.execXuiatch(s, env)
	case *parser.TryStatement:
		return i.execTry(s, env)
	case *parser.XuinterfaceStatement:
		return i.execXuinterface(s)
	case *parser.ExpressionStatement:
		return i.evalExpression(s.Expr, env)
	case *parser.BlockStatement:
		return i.execBlock(s, env)
	default:
		return nil, fmt.Errorf("unknown statement type: %T", stmt)
	}
}

func (i *Interpreter) execBlock(block *parser.BlockStatement, env *Environment) (*Value, error) {
	var result *Value
	for _, stmt := range block.Statements {
		val, err := i.execStatement(stmt, env)
		if err != nil {
			return nil, err
		}
		if val != nil && (val.Type == VAL_RETURN || val.Type == VAL_BREAK || val.Type == VAL_CONTINUE) {
			return val, nil
		}
		result = val
	}
	return result, nil
}

func (i *Interpreter) execXuet(s *parser.XuetStatement, env *Environment) (*Value, error) {
	val, err := i.evalExpression(s.Value, env)
	if err != nil {
		return nil, err
	}
	env.Define(s.Name, val, false)
	return nil, nil
}

func (i *Interpreter) execXuiar(s *parser.XuiarStatement, env *Environment) (*Value, error) {
	val, err := i.evalExpression(s.Value, env)
	if err != nil {
		return nil, err
	}
	env.Define(s.Name, val, true)
	return nil, nil
}

func (i *Interpreter) execAssign(s *parser.AssignStatement, env *Environment) (*Value, error) {
	val, err := i.evalExpression(s.Value, env)
	if err != nil {
		return nil, err
	}

	switch target := s.Target.(type) {
	case *parser.Identifier:
		if err := env.Set(target.Value, val); err != nil {
			return nil, fmt.Errorf("%s: %w", target.TokenPos(), err)
		}
	case *parser.MemberExpression:
		obj, err := i.evalExpression(target.Object, env)
		if err != nil {
			return nil, err
		}
		if obj.Type != VAL_STRUCT {
			return nil, fmt.Errorf("%s: cannot set field on %s", target.TokenPos(), obj.Type)
		}
		obj.StructVal.Fields[target.Member] = val
	case *parser.IndexExpression:
		container, err := i.evalExpression(target.Left, env)
		if err != nil {
			return nil, err
		}
		idx, err := i.evalExpression(target.Index, env)
		if err != nil {
			return nil, err
		}
		if container.Type == VAL_MAP {
			keyStr := idx.String()
			if _, exists := container.MapVal.Pairs[keyStr]; !exists {
				container.MapVal.Keys = append(container.MapVal.Keys, keyStr)
			}
			container.MapVal.Pairs[keyStr] = val
		} else if container.Type == VAL_ARRAY {
			if idx.Type != VAL_INT {
				return nil, fmt.Errorf("%s: array index must be int, got %s", target.TokenPos(), idx.Type)
			}
			index := int(idx.IntVal)
			if index < 0 || index >= len(container.ArrayVal) {
				return nil, fmt.Errorf("%s: index %d out of bounds (len=%d)", target.TokenPos(), index, len(container.ArrayVal))
			}
			container.ArrayVal[index] = val
		} else {
			return nil, fmt.Errorf("%s: cannot index %s", target.TokenPos(), container.Type)
		}
	default:
		return nil, fmt.Errorf("%s: invalid assignment target", s.TokenPos())
	}

	return nil, nil
}

func (i *Interpreter) execXuen(s *parser.XuenStatement, env *Environment) (*Value, error) {
	paramNames := make([]string, len(s.Params))
	for idx, p := range s.Params {
		paramNames[idx] = p.Name
	}

	fn := &FuncValue{
		Name:       s.Name,
		ParamNames: paramNames,
		Body:       s.Body,
		Closure:    env,
	}
	env.Define(s.Name, &Value{Type: VAL_FUNCTION, FuncVal: fn}, false)
	return nil, nil
}

func (i *Interpreter) execXueturn(s *parser.XueturnStatement, env *Environment) (*Value, error) {
	if s.Value == nil {
		return ReturnValue(NullValue()), nil
	}
	val, err := i.evalExpression(s.Value, env)
	if err != nil {
		return nil, err
	}
	return ReturnValue(val), nil
}

func (i *Interpreter) execXuif(s *parser.XuifStatement, env *Environment) (*Value, error) {
	cond, err := i.evalExpression(s.Condition, env)
	if err != nil {
		return nil, err
	}

	if cond.IsTruthy() {
		return i.execBlock(s.Consequence, NewEnclosedEnvironment(env))
	}

	if s.Alternative != nil {
		switch alt := s.Alternative.(type) {
		case *parser.BlockStatement:
			return i.execBlock(alt, NewEnclosedEnvironment(env))
		case *parser.XuifStatement:
			return i.execXuif(alt, env)
		}
	}

	return nil, nil
}

func (i *Interpreter) execXuior(s *parser.XuiorStatement, env *Environment) (*Value, error) {
	iterable, err := i.evalExpression(s.Iterable, env)
	if err != nil {
		return nil, err
	}

	var items []*Value

	switch iterable.Type {
	case VAL_RANGE:
		for idx := iterable.RangeVal.Start; idx < iterable.RangeVal.End; idx++ {
			items = append(items, IntVal(idx))
		}
	case VAL_ARRAY:
		items = iterable.ArrayVal
	case VAL_STRING:
		for _, ch := range iterable.StringVal {
			items = append(items, CharValue(ch))
		}
	default:
		return nil, fmt.Errorf("%s: cannot iterate over %s", s.TokenPos(), iterable.Type)
	}

	for _, item := range items {
		loopEnv := NewEnclosedEnvironment(env)
		loopEnv.Define(s.Variable, item, false)

		val, err := i.execBlock(s.Body, loopEnv)
		if err != nil {
			return nil, err
		}
		if val != nil {
			if val.Type == VAL_BREAK {
				break
			}
			if val.Type == VAL_RETURN {
				return val, nil
			}
			// VAL_CONTINUE: just continue the loop
		}
	}

	return nil, nil
}

func (i *Interpreter) execXuile(s *parser.XuileStatement, env *Environment) (*Value, error) {
	for {
		cond, err := i.evalExpression(s.Condition, env)
		if err != nil {
			return nil, err
		}
		if !cond.IsTruthy() {
			break
		}

		loopEnv := NewEnclosedEnvironment(env)
		val, err := i.execBlock(s.Body, loopEnv)
		if err != nil {
			return nil, err
		}
		if val != nil {
			if val.Type == VAL_BREAK {
				break
			}
			if val.Type == VAL_RETURN {
				return val, nil
			}
		}
	}
	return nil, nil
}

func (i *Interpreter) execXuiruct(s *parser.XuiructStatement) (*Value, error) {
	fieldNames := make([]string, len(s.Fields))
	fieldTypes := make([]string, len(s.Fields))
	for idx, f := range s.Fields {
		fieldNames[idx] = f.Name
		fieldTypes[idx] = f.TypeName
	}

	i.structDefs[s.Name] = &StructDef{
		Name:       s.Name,
		FieldNames: fieldNames,
		FieldTypes: fieldTypes,
		Methods:    make(map[string]*FuncValue),
	}
	return nil, nil
}

func (i *Interpreter) execXuimpl(s *parser.XuimplStatement, env *Environment) (*Value, error) {
	def, ok := i.structDefs[s.Name]
	if !ok {
		return nil, fmt.Errorf("%s: cannot implement methods for undefined struct %q", s.TokenPos(), s.Name)
	}

	for _, method := range s.Methods {
		paramNames := make([]string, len(method.Params))
		for idx, p := range method.Params {
			paramNames[idx] = p.Name
		}

		fn := &FuncValue{
			Name:       method.Name,
			ParamNames: paramNames,
			Body:       method.Body,
			Closure:    env,
			Receiver:   s.Name,
		}
		def.Methods[method.Name] = fn
	}

	return nil, nil
}

func (i *Interpreter) execXuinterface(s *parser.XuinterfaceStatement) (*Value, error) {
	methods := make([]string, len(s.Methods))
	for idx, m := range s.Methods {
		methods[idx] = m.Name
	}
	i.interfaceDefs[s.Name] = &InterfaceDef{
		Name:    s.Name,
		Methods: methods,
	}
	return nil, nil
}

func (i *Interpreter) execXuenum(s *parser.XuenumStatement, env *Environment) (*Value, error) {
	for _, variant := range s.Variants {
		env.Define(variant, &Value{
			Type: VAL_ENUM_VARIANT,
			EnumVal: &EnumVariantValue{
				EnumName:    s.Name,
				VariantName: variant,
			},
		}, false)
	}
	return nil, nil
}

func (i *Interpreter) execXuiatch(s *parser.XuiatchStatement, env *Environment) (*Value, error) {
	val, err := i.evalExpression(s.Value, env)
	if err != nil {
		return nil, err
	}

	for _, arm := range s.Arms {
		// Check for wildcard "_"
		if ident, ok := arm.Pattern.(*parser.Identifier); ok && ident.Value == "_" {
			return i.execStatement(arm.Body, NewEnclosedEnvironment(env))
		}

		pattern, err := i.evalExpression(arm.Pattern, env)
		if err != nil {
			return nil, err
		}

		if valuesEqual(val, pattern) {
			return i.execStatement(arm.Body, NewEnclosedEnvironment(env))
		}
	}

	return nil, nil
}

// --- Expression evaluation ---

func (i *Interpreter) evalExpression(expr parser.Expression, env *Environment) (*Value, error) {
	switch e := expr.(type) {
	case *parser.IntegerLiteral:
		return IntVal(e.Value), nil
	case *parser.FloatLiteral:
		return FloatVal(e.Value), nil
	case *parser.StringLiteral:
		return StringVal(e.Value), nil
	case *parser.CharLiteral:
		return CharValue(e.Value), nil
	case *parser.BoolLiteral:
		return BoolValue(e.Value), nil
	case *parser.NullLiteral:
		return NullValue(), nil
	case *parser.Identifier:
		return i.evalIdentifier(e, env)
	case *parser.PrefixExpression:
		return i.evalPrefix(e, env)
	case *parser.InfixExpression:
		return i.evalInfix(e, env)
	case *parser.CallExpression:
		return i.evalCall(e, env)
	case *parser.MemberExpression:
		return i.evalMember(e, env)
	case *parser.IndexExpression:
		return i.evalIndex(e, env)
	case *parser.ArrayLiteral:
		return i.evalArray(e, env)
	case *parser.RangeExpression:
		return i.evalRange(e, env)
	case *parser.StructLiteral:
		return i.evalStructLiteral(e, env)
	case *parser.MapLiteral:
		return i.evalMapLiteral(e, env)
	case *parser.InterpolatedString:
		return i.evalInterpolatedString(e, env)
	case *parser.LambdaExpression:
		return i.evalLambda(e, env)
	case *parser.ThrowExpression:
		val, err := i.evalExpression(e.Value, env)
		if err != nil {
			return nil, err
		}
		return nil, fmt.Errorf("%s", val.String())
	case *parser.AddressOfExpression:
		return i.evalAddressOf(e, env)
	case *parser.DerefExpression:
		return i.evalDeref(e, env)
	default:
		return nil, fmt.Errorf("unknown expression type: %T", expr)
	}
}

func (i *Interpreter) evalIdentifier(e *parser.Identifier, env *Environment) (*Value, error) {
	val, ok := env.Get(e.Value)
	if !ok {
		return nil, fmt.Errorf("%s: undefined variable %q", e.TokenPos(), e.Value)
	}
	return val, nil
}

func (i *Interpreter) evalAddressOf(e *parser.AddressOfExpression, env *Environment) (*Value, error) {
	// e.Value must be an Identifier
	ident, ok := e.Value.(*parser.Identifier)
	if !ok {
		return nil, fmt.Errorf("can only take address of a variable")
	}
	// Check variable exists
	if _, ok := env.Get(ident.Value); !ok {
		return nil, fmt.Errorf("undefined variable %q", ident.Value)
	}
	return &Value{Type: VAL_POINTER, PointerVal: &PointerValue{Env: env, Name: ident.Value}}, nil
}

func (i *Interpreter) evalDeref(e *parser.DerefExpression, env *Environment) (*Value, error) {
	val, err := i.evalExpression(e.Value, env)
	if err != nil {
		return nil, err
	}
	if val.Type != VAL_POINTER {
		return nil, fmt.Errorf("cannot dereference %s", val.Type)
	}
	result, ok := val.PointerVal.Env.Get(val.PointerVal.Name)
	if !ok {
		return nil, fmt.Errorf("dangling pointer")
	}
	return result, nil
}

func (i *Interpreter) evalPrefix(e *parser.PrefixExpression, env *Environment) (*Value, error) {
	right, err := i.evalExpression(e.Right, env)
	if err != nil {
		return nil, err
	}

	switch e.Operator {
	case "-":
		switch right.Type {
		case VAL_INT:
			return IntVal(-right.IntVal), nil
		case VAL_FLOAT:
			return FloatVal(-right.FloatVal), nil
		default:
			return nil, fmt.Errorf("%s: cannot negate %s", e.TokenPos(), right.Type)
		}
	case "!":
		return BoolValue(!right.IsTruthy()), nil
	default:
		return nil, fmt.Errorf("%s: unknown prefix operator %q", e.TokenPos(), e.Operator)
	}
}

func (i *Interpreter) evalInfix(e *parser.InfixExpression, env *Environment) (*Value, error) {
	// Short-circuit for && and ||
	if e.Operator == "&&" {
		left, err := i.evalExpression(e.Left, env)
		if err != nil {
			return nil, err
		}
		if !left.IsTruthy() {
			return BoolValue(false), nil
		}
		right, err := i.evalExpression(e.Right, env)
		if err != nil {
			return nil, err
		}
		return BoolValue(right.IsTruthy()), nil
	}
	if e.Operator == "||" {
		left, err := i.evalExpression(e.Left, env)
		if err != nil {
			return nil, err
		}
		if left.IsTruthy() {
			return BoolValue(true), nil
		}
		right, err := i.evalExpression(e.Right, env)
		if err != nil {
			return nil, err
		}
		return BoolValue(right.IsTruthy()), nil
	}

	left, err := i.evalExpression(e.Left, env)
	if err != nil {
		return nil, err
	}
	right, err := i.evalExpression(e.Right, env)
	if err != nil {
		return nil, err
	}

	// String concatenation
	if e.Operator == "+" && (left.Type == VAL_STRING || right.Type == VAL_STRING) {
		return StringVal(left.String() + right.String()), nil
	}

	// == and != work on all types
	if e.Operator == "==" {
		return BoolValue(valuesEqual(left, right)), nil
	}
	if e.Operator == "!=" {
		return BoolValue(!valuesEqual(left, right)), nil
	}

	// Numeric operations — promote int to float if mixed
	if left.Type == VAL_INT && right.Type == VAL_INT {
		return i.evalIntInfix(e.Operator, left.IntVal, right.IntVal, e)
	}

	lf, rf, ok := toFloats(left, right)
	if ok {
		return i.evalFloatInfix(e.Operator, lf, rf, e)
	}

	// Check for operator overloading on structs
	if left.Type == VAL_STRUCT {
		methodName := ""
		switch e.Operator {
		case "+":
			methodName = "__add"
		case "-":
			methodName = "__sub"
		case "*":
			methodName = "__mul"
		case "==":
			methodName = "__eq"
		case "<":
			methodName = "__lt"
		case ">":
			methodName = "__gt"
		}
		if methodName != "" {
			def, ok := i.structDefs[left.StructVal.TypeName]
			if ok {
				if method, ok := def.Methods[methodName]; ok {
					methodEnv := NewEnclosedEnvironment(method.Closure)
					methodEnv.Define("self", left, true)
					methodEnv.Define(method.ParamNames[1], right, true) // skip self, use second param
					body := method.Body.(*parser.BlockStatement)
					result, err := i.execBlock(body, methodEnv)
					if err != nil {
						return nil, err
					}
					if result != nil && result.Type == VAL_RETURN {
						return result.ReturnVal, nil
					}
					return NullValue(), nil
				}
			}
		}
	}

	return nil, fmt.Errorf("%s: unsupported operation %s %s %s", e.TokenPos(), left.Type, e.Operator, right.Type)
}

func (i *Interpreter) evalIntInfix(op string, l, r int64, e *parser.InfixExpression) (*Value, error) {
	switch op {
	case "+":
		return IntVal(l + r), nil
	case "-":
		return IntVal(l - r), nil
	case "*":
		return IntVal(l * r), nil
	case "/":
		if r == 0 {
			return nil, fmt.Errorf("%s: division by zero", e.TokenPos())
		}
		return IntVal(l / r), nil
	case "%":
		if r == 0 {
			return nil, fmt.Errorf("%s: modulo by zero", e.TokenPos())
		}
		return IntVal(l % r), nil
	case "<":
		return BoolValue(l < r), nil
	case ">":
		return BoolValue(l > r), nil
	case "<=":
		return BoolValue(l <= r), nil
	case ">=":
		return BoolValue(l >= r), nil
	default:
		return nil, fmt.Errorf("%s: unknown operator %q for int", e.TokenPos(), op)
	}
}

func (i *Interpreter) evalFloatInfix(op string, l, r float64, e *parser.InfixExpression) (*Value, error) {
	switch op {
	case "+":
		return FloatVal(l + r), nil
	case "-":
		return FloatVal(l - r), nil
	case "*":
		return FloatVal(l * r), nil
	case "/":
		if r == 0 {
			return nil, fmt.Errorf("%s: division by zero", e.TokenPos())
		}
		return FloatVal(l / r), nil
	case "%":
		return FloatVal(math.Mod(l, r)), nil
	case "<":
		return BoolValue(l < r), nil
	case ">":
		return BoolValue(l > r), nil
	case "<=":
		return BoolValue(l <= r), nil
	case ">=":
		return BoolValue(l >= r), nil
	default:
		return nil, fmt.Errorf("%s: unknown operator %q for float", e.TokenPos(), op)
	}
}

func (i *Interpreter) evalCall(e *parser.CallExpression, env *Environment) (*Value, error) {
	// Check for method call: obj.method(args)
	if member, ok := e.Function.(*parser.MemberExpression); ok {
		return i.evalMethodCall(member, e.Arguments, env)
	}

	fn, err := i.evalExpression(e.Function, env)
	if err != nil {
		return nil, err
	}

	args, err := i.evalExpressions(e.Arguments, env)
	if err != nil {
		return nil, err
	}

	return i.callFunction(fn, args, e)
}

func (i *Interpreter) evalMethodCall(member *parser.MemberExpression, argExprs []parser.Expression, env *Environment) (*Value, error) {
	obj, err := i.evalExpression(member.Object, env)
	if err != nil {
		return nil, err
	}

	args, err := i.evalExpressions(argExprs, env)
	if err != nil {
		return nil, err
	}

	if obj.Type != VAL_STRUCT {
		return nil, fmt.Errorf("%s: cannot call method on %s", member.TokenPos(), obj.Type)
	}

	def, ok := i.structDefs[obj.StructVal.TypeName]
	if !ok {
		return nil, fmt.Errorf("%s: no methods for struct %q", member.TokenPos(), obj.StructVal.TypeName)
	}

	method, ok := def.Methods[member.Member]
	if !ok {
		return nil, fmt.Errorf("%s: struct %q has no method %q", member.TokenPos(), obj.StructVal.TypeName, member.Member)
	}

	// Create method environment: self + params
	methodEnv := NewEnclosedEnvironment(method.Closure)
	paramIdx := 0
	for _, pName := range method.ParamNames {
		if pName == "self" {
			methodEnv.Define("self", obj, true)
		} else {
			if paramIdx >= len(args) {
				return nil, fmt.Errorf("%s: not enough arguments for method %s", member.TokenPos(), method.Name)
			}
			methodEnv.Define(pName, args[paramIdx], true)
			paramIdx++
		}
	}

	body := method.Body.(*parser.BlockStatement)
	result, err := i.execBlock(body, methodEnv)
	if err != nil {
		return nil, err
	}
	if result != nil && result.Type == VAL_RETURN {
		return result.ReturnVal, nil
	}
	return NullValue(), nil
}

func (i *Interpreter) callFunction(fn *Value, args []*Value, e *parser.CallExpression) (*Value, error) {
	switch fn.Type {
	case VAL_BUILTIN:
		return fn.BuiltinVal(args)
	case VAL_FUNCTION:
		return i.callUserFunc(fn.FuncVal, args, e)
	default:
		return nil, fmt.Errorf("%s: %s is not callable", e.TokenPos(), fn.Type)
	}
}

func (i *Interpreter) callUserFunc(fn *FuncValue, args []*Value, e *parser.CallExpression) (*Value, error) {
	if len(args) != len(fn.ParamNames) {
		return nil, fmt.Errorf("%s: %s() expects %d arguments, got %d",
			e.TokenPos(), fn.Name, len(fn.ParamNames), len(args))
	}

	funcEnv := NewEnclosedEnvironment(fn.Closure)
	for idx, paramName := range fn.ParamNames {
		funcEnv.Define(paramName, args[idx], true)
	}

	body := fn.Body.(*parser.BlockStatement)
	result, err := i.execBlock(body, funcEnv)
	if err != nil {
		return nil, err
	}
	if result != nil && result.Type == VAL_RETURN {
		return result.ReturnVal, nil
	}
	return NullValue(), nil
}

func (i *Interpreter) evalMember(e *parser.MemberExpression, env *Environment) (*Value, error) {
	obj, err := i.evalExpression(e.Object, env)
	if err != nil {
		return nil, err
	}

	if obj.Type == VAL_STRUCT {
		val, ok := obj.StructVal.Fields[e.Member]
		if !ok {
			return nil, fmt.Errorf("%s: struct %q has no field %q", e.TokenPos(), obj.StructVal.TypeName, e.Member)
		}
		return val, nil
	}

	// Array/string .length
	if e.Member == "length" {
		switch obj.Type {
		case VAL_ARRAY:
			return IntVal(int64(len(obj.ArrayVal))), nil
		case VAL_STRING:
			return IntVal(int64(len([]rune(obj.StringVal)))), nil
		}
	}

	return nil, fmt.Errorf("%s: cannot access member %q on %s", e.TokenPos(), e.Member, obj.Type)
}

func (i *Interpreter) evalIndex(e *parser.IndexExpression, env *Environment) (*Value, error) {
	left, err := i.evalExpression(e.Left, env)
	if err != nil {
		return nil, err
	}
	index, err := i.evalExpression(e.Index, env)
	if err != nil {
		return nil, err
	}

	switch left.Type {
	case VAL_ARRAY:
		if index.Type != VAL_INT {
			return nil, fmt.Errorf("%s: array index must be int, got %s", e.TokenPos(), index.Type)
		}
		idx := int(index.IntVal)
		if idx < 0 || idx >= len(left.ArrayVal) {
			return nil, fmt.Errorf("%s: index %d out of bounds (len=%d)", e.TokenPos(), idx, len(left.ArrayVal))
		}
		return left.ArrayVal[idx], nil
	case VAL_MAP:
		keyStr := index.String()
		if val, ok := left.MapVal.Pairs[keyStr]; ok {
			return val, nil
		}
		return NullValue(), nil
	case VAL_STRING:
		if index.Type != VAL_INT {
			return nil, fmt.Errorf("%s: string index must be int, got %s", e.TokenPos(), index.Type)
		}
		runes := []rune(left.StringVal)
		idx := int(index.IntVal)
		if idx < 0 || idx >= len(runes) {
			return nil, fmt.Errorf("%s: index %d out of bounds (len=%d)", e.TokenPos(), idx, len(runes))
		}
		return CharValue(runes[idx]), nil
	default:
		return nil, fmt.Errorf("%s: cannot index %s", e.TokenPos(), left.Type)
	}
}

func (i *Interpreter) evalArray(e *parser.ArrayLiteral, env *Environment) (*Value, error) {
	elements, err := i.evalExpressions(e.Elements, env)
	if err != nil {
		return nil, err
	}
	return ArrayValue(elements), nil
}

func (i *Interpreter) evalRange(e *parser.RangeExpression, env *Environment) (*Value, error) {
	start, err := i.evalExpression(e.Start, env)
	if err != nil {
		return nil, err
	}
	end, err := i.evalExpression(e.End, env)
	if err != nil {
		return nil, err
	}

	if start.Type != VAL_INT || end.Type != VAL_INT {
		return nil, fmt.Errorf("%s: range requires int operands, got %s..%s", e.TokenPos(), start.Type, end.Type)
	}

	return RangeVal(start.IntVal, end.IntVal), nil
}

func (i *Interpreter) evalStructLiteral(e *parser.StructLiteral, env *Environment) (*Value, error) {
	def, ok := i.structDefs[e.Name]
	if !ok {
		return nil, fmt.Errorf("%s: undefined struct %q", e.TokenPos(), e.Name)
	}

	fields := make(map[string]*Value)
	// Initialize all fields with defaults
	for _, name := range def.FieldNames {
		fields[name] = NullValue()
	}
	// Set provided fields
	for _, f := range e.Fields {
		val, err := i.evalExpression(f.Value, env)
		if err != nil {
			return nil, err
		}
		fields[f.Name] = val
	}

	return &Value{
		Type: VAL_STRUCT,
		StructVal: &StructValue{
			TypeName: e.Name,
			Fields:   fields,
		},
	}, nil
}

func (i *Interpreter) evalExpressions(exprs []parser.Expression, env *Environment) ([]*Value, error) {
	result := make([]*Value, len(exprs))
	for idx, expr := range exprs {
		val, err := i.evalExpression(expr, env)
		if err != nil {
			return nil, err
		}
		result[idx] = val
	}
	return result, nil
}

func (i *Interpreter) evalLambda(e *parser.LambdaExpression, env *Environment) (*Value, error) {
	paramNames := make([]string, len(e.Params))
	for idx, p := range e.Params {
		paramNames[idx] = p.Name
	}

	fn := &FuncValue{
		Name:       "<lambda>",
		ParamNames: paramNames,
		Closure:    env,
	}

	if e.Block != nil {
		fn.Body = e.Block
	} else {
		// Wrap single expression in a block with return
		fn.Body = &parser.BlockStatement{
			Pos: e.Pos,
			Statements: []parser.Statement{
				&parser.XueturnStatement{Pos: e.Pos, Value: e.Body},
			},
		}
	}

	return &Value{Type: VAL_FUNCTION, FuncVal: fn}, nil
}

func (i *Interpreter) execTry(s *parser.TryStatement, env *Environment) (*Value, error) {
	tryEnv := NewEnclosedEnvironment(env)
	val, err := i.execBlock(s.Body, tryEnv)
	if err != nil {
		// Error caught — run catch block
		catchEnv := NewEnclosedEnvironment(env)
		catchEnv.Define(s.CatchVar, StringVal(err.Error()), false)
		return i.execBlock(s.CatchBody, catchEnv)
	}
	return val, nil
}

func (i *Interpreter) evalMapLiteral(e *parser.MapLiteral, env *Environment) (*Value, error) {
	pairs := make(map[string]*Value)
	var keys []string
	for _, pair := range e.Pairs {
		key, err := i.evalExpression(pair.Key, env)
		if err != nil {
			return nil, err
		}
		val, err := i.evalExpression(pair.Value, env)
		if err != nil {
			return nil, err
		}
		keyStr := key.String()
		pairs[keyStr] = val
		keys = append(keys, keyStr)
	}
	return MapVal(pairs, keys), nil
}

func (i *Interpreter) evalInterpolatedString(e *parser.InterpolatedString, env *Environment) (*Value, error) {
	var buf strings.Builder
	for _, part := range e.Parts {
		val, err := i.evalExpression(part, env)
		if err != nil {
			return nil, err
		}
		buf.WriteString(val.String())
	}
	return StringVal(buf.String()), nil
}

// --- Helpers ---

func valuesEqual(a, b *Value) bool {
	if a.Type != b.Type {
		// Allow int/float comparison
		af, bf, ok := toFloats(a, b)
		if ok {
			return af == bf
		}
		return false
	}
	switch a.Type {
	case VAL_INT:
		return a.IntVal == b.IntVal
	case VAL_FLOAT:
		return a.FloatVal == b.FloatVal
	case VAL_STRING:
		return a.StringVal == b.StringVal
	case VAL_CHAR:
		return a.CharVal == b.CharVal
	case VAL_BOOL:
		return a.BoolVal == b.BoolVal
	case VAL_NULL:
		return true
	case VAL_ENUM_VARIANT:
		return a.EnumVal.EnumName == b.EnumVal.EnumName && a.EnumVal.VariantName == b.EnumVal.VariantName
	default:
		return false
	}
}

func toFloats(a, b *Value) (float64, float64, bool) {
	var af, bf float64
	switch a.Type {
	case VAL_INT:
		af = float64(a.IntVal)
	case VAL_FLOAT:
		af = a.FloatVal
	default:
		return 0, 0, false
	}
	switch b.Type {
	case VAL_INT:
		bf = float64(b.IntVal)
	case VAL_FLOAT:
		bf = b.FloatVal
	default:
		return 0, 0, false
	}
	return af, bf, true
}
