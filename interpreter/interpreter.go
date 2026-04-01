package interpreter

import (
	"bufio"
	"crypto/md5"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/00000kkkkk/xusesosplusplus/parser"
)

var wgRegistry = make(map[int64]*sync.WaitGroup)
var wgCounter int64
var muRegistry = make(map[int64]*sync.Mutex)
var muCounter int64
var tcpListeners = make(map[int64]net.Listener)
var tcpListenerCounter int64
var tcpConns = make(map[int64]net.Conn)
var tcpConnCounter int64

// InterfaceDef stores an interface (trait) definition.
type InterfaceDef struct {
	Name    string
	Methods []string // required method names
}

// deferredCall stores a deferred function call and its environment.
type deferredCall struct {
	expr parser.Expression
	env  *Environment
}

// Interpreter evaluates an AST.
type Interpreter struct {
	globals       *Environment
	structDefs    map[string]*StructDef
	interfaceDefs map[string]*InterfaceDef
	output        []string // captured output for testing
	Imports       *ImportResolver
	deferStack    []deferredCall
}

// New creates a new interpreter with built-in functions.
func New() *Interpreter {
	interp := &Interpreter{
		globals:       NewEnvironment(),
		structDefs:    make(map[string]*StructDef),
		interfaceDefs: make(map[string]*InterfaceDef),
		deferStack:    make([]deferredCall, 0),
	}
	interp.registerBuiltins()
	return interp
}

// Run executes a parsed program. If a main() function is defined, it is called automatically.
func (i *Interpreter) Run(program *parser.Program) error {
	for _, stmt := range program.Statements {
		val, err := i.execStatement(stmt, i.globals)
		if err != nil {
			i.runDeferred()
			return err
		}
		if val != nil && val.Type == VAL_RETURN {
			i.runDeferred()
			return nil
		}
	}

	// Auto-call main() if defined
	if mainVal, ok := i.globals.Get("main"); ok && mainVal.Type == VAL_FUNCTION {
		_, err := i.callUserFunc(mainVal.FuncVal, nil, &parser.CallExpression{})
		if err != nil {
			i.runDeferred()
			return err
		}
	}

	// Execute deferred calls in reverse order
	i.runDeferred()

	return nil
}

// runDeferred executes all deferred calls in LIFO order and clears the stack.
func (i *Interpreter) runDeferred() {
	for idx := len(i.deferStack) - 1; idx >= 0; idx-- {
		d := i.deferStack[idx]
		i.evalExpression(d.expr, d.env)
	}
	i.deferStack = nil
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

	// --- Tuple built-ins (multiple return values) ---

	// tuple(...) — create a tuple from arguments
	i.globals.Define("tuple", &Value{Type: VAL_BUILTIN, BuiltinVal: func(args []*Value) (*Value, error) {
		return TupleValue(args), nil
	}}, false)

	// first(tuple) — get first element
	i.globals.Define("first", &Value{Type: VAL_BUILTIN, BuiltinVal: func(args []*Value) (*Value, error) {
		if len(args) != 1 || args[0].Type != VAL_TUPLE || len(args[0].TupleVal) < 1 {
			return nil, fmt.Errorf("first() requires a tuple with at least 1 element")
		}
		return args[0].TupleVal[0], nil
	}}, false)

	// second(tuple) — get second element
	i.globals.Define("second", &Value{Type: VAL_BUILTIN, BuiltinVal: func(args []*Value) (*Value, error) {
		if len(args) != 1 || args[0].Type != VAL_TUPLE || len(args[0].TupleVal) < 2 {
			return nil, fmt.Errorf("second() requires a tuple with at least 2 elements")
		}
		return args[0].TupleVal[1], nil
	}}, false)

	// unpack(tuple, index) — get element at index
	i.globals.Define("unpack", &Value{Type: VAL_BUILTIN, BuiltinVal: func(args []*Value) (*Value, error) {
		if len(args) != 2 || args[0].Type != VAL_TUPLE || args[1].Type != VAL_INT {
			return nil, fmt.Errorf("unpack() takes a tuple and an index")
		}
		idx := int(args[1].IntVal)
		if idx < 0 || idx >= len(args[0].TupleVal) {
			return nil, fmt.Errorf("unpack: index %d out of bounds", idx)
		}
		return args[0].TupleVal[idx], nil
	}}, false)

	// is_error(value) — check if value is an error (null or non-empty string)
	i.globals.Define("is_error", &Value{Type: VAL_BUILTIN, BuiltinVal: func(args []*Value) (*Value, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("is_error() takes 1 argument")
		}
		return BoolValue(args[0].Type == VAL_NULL || (args[0].Type == VAL_STRING && args[0].StringVal != "")), nil
	}}, false)

	// --- WaitGroup built-ins ---

	// wg_new() — create a new WaitGroup, returns its ID
	i.globals.Define("wg_new", &Value{Type: VAL_BUILTIN, BuiltinVal: func(args []*Value) (*Value, error) {
		wgCounter++
		wgRegistry[wgCounter] = &sync.WaitGroup{}
		return IntVal(wgCounter), nil
	}}, false)

	// wg_add(id, n?) — add delta to WaitGroup (default 1)
	i.globals.Define("wg_add", &Value{Type: VAL_BUILTIN, BuiltinVal: func(args []*Value) (*Value, error) {
		if len(args) < 1 || args[0].Type != VAL_INT {
			return nil, fmt.Errorf("wg_add() takes WaitGroup ID")
		}
		wg, ok := wgRegistry[args[0].IntVal]
		if !ok {
			return nil, fmt.Errorf("invalid WaitGroup")
		}
		n := 1
		if len(args) > 1 && args[1].Type == VAL_INT {
			n = int(args[1].IntVal)
		}
		wg.Add(n)
		return NullValue(), nil
	}}, false)

	// wg_done(id) — decrement WaitGroup counter
	i.globals.Define("wg_done", &Value{Type: VAL_BUILTIN, BuiltinVal: func(args []*Value) (*Value, error) {
		if len(args) != 1 || args[0].Type != VAL_INT {
			return nil, fmt.Errorf("wg_done() takes WaitGroup ID")
		}
		wg, ok := wgRegistry[args[0].IntVal]
		if !ok {
			return nil, fmt.Errorf("invalid WaitGroup")
		}
		wg.Done()
		return NullValue(), nil
	}}, false)

	// wg_wait(id) — block until WaitGroup counter is zero
	i.globals.Define("wg_wait", &Value{Type: VAL_BUILTIN, BuiltinVal: func(args []*Value) (*Value, error) {
		if len(args) != 1 || args[0].Type != VAL_INT {
			return nil, fmt.Errorf("wg_wait() takes WaitGroup ID")
		}
		wg, ok := wgRegistry[args[0].IntVal]
		if !ok {
			return nil, fmt.Errorf("invalid WaitGroup")
		}
		wg.Wait()
		return NullValue(), nil
	}}, false)

	// --- Mutex built-ins ---

	// mutex_new() — create a new Mutex, returns its ID
	i.globals.Define("mutex_new", &Value{Type: VAL_BUILTIN, BuiltinVal: func(args []*Value) (*Value, error) {
		muCounter++
		muRegistry[muCounter] = &sync.Mutex{}
		return IntVal(muCounter), nil
	}}, false)

	// mutex_lock(id) — lock the Mutex
	i.globals.Define("mutex_lock", &Value{Type: VAL_BUILTIN, BuiltinVal: func(args []*Value) (*Value, error) {
		if len(args) != 1 || args[0].Type != VAL_INT {
			return nil, fmt.Errorf("mutex_lock() takes Mutex ID")
		}
		mu, ok := muRegistry[args[0].IntVal]
		if !ok {
			return nil, fmt.Errorf("invalid Mutex")
		}
		mu.Lock()
		return NullValue(), nil
	}}, false)

	// mutex_unlock(id) — unlock the Mutex
	i.globals.Define("mutex_unlock", &Value{Type: VAL_BUILTIN, BuiltinVal: func(args []*Value) (*Value, error) {
		if len(args) != 1 || args[0].Type != VAL_INT {
			return nil, fmt.Errorf("mutex_unlock() takes Mutex ID")
		}
		mu, ok := muRegistry[args[0].IntVal]
		if !ok {
			return nil, fmt.Errorf("invalid Mutex")
		}
		mu.Unlock()
		return NullValue(), nil
	}}, false)

	// benchmark(name, func, iterations) — run a function N times and print timing
	i.globals.Define("benchmark", &Value{Type: VAL_BUILTIN, BuiltinVal: func(args []*Value) (*Value, error) {
		if len(args) < 2 {
			return nil, fmt.Errorf("benchmark() takes name, function, and optional iterations")
		}
		name := args[0].String()
		fn := args[1]
		iterations := int64(1000)
		if len(args) > 2 && args[2].Type == VAL_INT {
			iterations = args[2].IntVal
		}

		if fn.Type != VAL_FUNCTION && fn.Type != VAL_BUILTIN {
			return nil, fmt.Errorf("benchmark() second argument must be a function")
		}

		start := time.Now()
		for j := int64(0); j < iterations; j++ {
			if fn.Type == VAL_BUILTIN {
				_, _ = fn.BuiltinVal(nil)
			} else {
				body := fn.FuncVal.Body.(*parser.BlockStatement)
				env := NewEnclosedEnvironment(fn.FuncVal.Closure)
				_, _ = i.execBlock(body, env)
			}
		}
		elapsed := time.Since(start)
		nsPerOp := elapsed.Nanoseconds() / iterations

		line := fmt.Sprintf("benchmark %s: %d iterations, %s total, %d ns/op", name, iterations, elapsed.Round(time.Microsecond), nsPerOp)
		fmt.Println(line)
		i.output = append(i.output, line)
		return NullValue(), nil
	}}, false)

	// time_now() — returns current time in milliseconds
	i.globals.Define("time_now", &Value{Type: VAL_BUILTIN, BuiltinVal: func(args []*Value) (*Value, error) {
		return IntVal(time.Now().UnixMilli()), nil
	}}, false)

	// time_since(start_ms) — returns elapsed milliseconds since start
	i.globals.Define("time_since", &Value{Type: VAL_BUILTIN, BuiltinVal: func(args []*Value) (*Value, error) {
		if len(args) != 1 || args[0].Type != VAL_INT {
			return nil, fmt.Errorf("time_since() takes 1 int argument (start milliseconds)")
		}
		elapsed := time.Now().UnixMilli() - args[0].IntVal
		return IntVal(elapsed), nil
	}}, false)

	// --- String functions ---

	// repeat(str, n) — repeat string n times
	i.globals.Define("repeat", &Value{Type: VAL_BUILTIN, BuiltinVal: func(args []*Value) (*Value, error) {
		if len(args) != 2 || args[0].Type != VAL_STRING || args[1].Type != VAL_INT {
			return nil, fmt.Errorf("repeat() takes string and int")
		}
		return StringVal(strings.Repeat(args[0].StringVal, int(args[1].IntVal))), nil
	}}, false)

	// pad_left(str, length, char) — pad string on the left
	i.globals.Define("pad_left", &Value{Type: VAL_BUILTIN, BuiltinVal: func(args []*Value) (*Value, error) {
		if len(args) != 3 || args[0].Type != VAL_STRING || args[1].Type != VAL_INT || args[2].Type != VAL_STRING {
			return nil, fmt.Errorf("pad_left() takes string, int, string")
		}
		s := args[0].StringVal
		target := int(args[1].IntVal)
		pad := args[2].StringVal
		for len(s) < target {
			s = pad + s
		}
		return StringVal(s[:target]), nil
	}}, false)

	// pad_right(str, length, char)
	i.globals.Define("pad_right", &Value{Type: VAL_BUILTIN, BuiltinVal: func(args []*Value) (*Value, error) {
		if len(args) != 3 || args[0].Type != VAL_STRING || args[1].Type != VAL_INT || args[2].Type != VAL_STRING {
			return nil, fmt.Errorf("pad_right() takes string, int, string")
		}
		s := args[0].StringVal
		target := int(args[1].IntVal)
		pad := args[2].StringVal
		for len(s) < target {
			s = s + pad
		}
		return StringVal(s[:target]), nil
	}}, false)

	// count(str, substr) — count occurrences
	i.globals.Define("count", &Value{Type: VAL_BUILTIN, BuiltinVal: func(args []*Value) (*Value, error) {
		if len(args) != 2 || args[0].Type != VAL_STRING || args[1].Type != VAL_STRING {
			return nil, fmt.Errorf("count() takes 2 strings")
		}
		return IntVal(int64(strings.Count(args[0].StringVal, args[1].StringVal))), nil
	}}, false)

	// reverse(str_or_arr) — reverse string or array
	i.globals.Define("reverse", &Value{Type: VAL_BUILTIN, BuiltinVal: func(args []*Value) (*Value, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("reverse() takes 1 argument")
		}
		switch args[0].Type {
		case VAL_STRING:
			runes := []rune(args[0].StringVal)
			for ii, j := 0, len(runes)-1; ii < j; ii, j = ii+1, j-1 {
				runes[ii], runes[j] = runes[j], runes[ii]
			}
			return StringVal(string(runes)), nil
		case VAL_ARRAY:
			arr := args[0].ArrayVal
			result := make([]*Value, len(arr))
			for ii, j := 0, len(arr)-1; j >= 0; ii, j = ii+1, j-1 {
				result[ii] = arr[j]
			}
			return ArrayValue(result), nil
		default:
			return nil, fmt.Errorf("reverse() takes string or array")
		}
	}}, false)

	// --- Array functions ---

	// sort_arr(arr) — sort array of ints/strings (returns new sorted array)
	i.globals.Define("sort_arr", &Value{Type: VAL_BUILTIN, BuiltinVal: func(args []*Value) (*Value, error) {
		if len(args) != 1 || args[0].Type != VAL_ARRAY {
			return nil, fmt.Errorf("sort_arr() takes 1 array argument")
		}
		arr := make([]*Value, len(args[0].ArrayVal))
		copy(arr, args[0].ArrayVal)
		// Simple bubble sort
		for ii := 0; ii < len(arr); ii++ {
			for j := 0; j < len(arr)-1-ii; j++ {
				swap := false
				if arr[j].Type == VAL_INT && arr[j+1].Type == VAL_INT {
					swap = arr[j].IntVal > arr[j+1].IntVal
				} else if arr[j].Type == VAL_STRING && arr[j+1].Type == VAL_STRING {
					swap = arr[j].StringVal > arr[j+1].StringVal
				}
				if swap {
					arr[j], arr[j+1] = arr[j+1], arr[j]
				}
			}
		}
		return ArrayValue(arr), nil
	}}, false)

	// unique(arr) — remove duplicates
	i.globals.Define("unique", &Value{Type: VAL_BUILTIN, BuiltinVal: func(args []*Value) (*Value, error) {
		if len(args) != 1 || args[0].Type != VAL_ARRAY {
			return nil, fmt.Errorf("unique() takes 1 array argument")
		}
		seen := make(map[string]bool)
		var result []*Value
		for _, v := range args[0].ArrayVal {
			key := v.Inspect()
			if !seen[key] {
				seen[key] = true
				result = append(result, v)
			}
		}
		return ArrayValue(result), nil
	}}, false)

	// flatten(arr) — flatten nested arrays one level
	i.globals.Define("flatten", &Value{Type: VAL_BUILTIN, BuiltinVal: func(args []*Value) (*Value, error) {
		if len(args) != 1 || args[0].Type != VAL_ARRAY {
			return nil, fmt.Errorf("flatten() takes 1 array argument")
		}
		var result []*Value
		for _, v := range args[0].ArrayVal {
			if v.Type == VAL_ARRAY {
				result = append(result, v.ArrayVal...)
			} else {
				result = append(result, v)
			}
		}
		return ArrayValue(result), nil
	}}, false)

	// zip(arr1, arr2) — zip two arrays into array of tuples
	i.globals.Define("zip", &Value{Type: VAL_BUILTIN, BuiltinVal: func(args []*Value) (*Value, error) {
		if len(args) != 2 || args[0].Type != VAL_ARRAY || args[1].Type != VAL_ARRAY {
			return nil, fmt.Errorf("zip() takes 2 arrays")
		}
		a, b := args[0].ArrayVal, args[1].ArrayVal
		minLen := len(a)
		if len(b) < minLen {
			minLen = len(b)
		}
		result := make([]*Value, minLen)
		for idx := 0; idx < minLen; idx++ {
			result[idx] = TupleValue([]*Value{a[idx], b[idx]})
		}
		return ArrayValue(result), nil
	}}, false)

	// enumerate(arr) — returns array of (index, value) tuples
	i.globals.Define("enumerate", &Value{Type: VAL_BUILTIN, BuiltinVal: func(args []*Value) (*Value, error) {
		if len(args) != 1 || args[0].Type != VAL_ARRAY {
			return nil, fmt.Errorf("enumerate() takes 1 array")
		}
		result := make([]*Value, len(args[0].ArrayVal))
		for idx, v := range args[0].ArrayVal {
			result[idx] = TupleValue([]*Value{IntVal(int64(idx)), v})
		}
		return ArrayValue(result), nil
	}}, false)

	// --- Math functions ---

	// math_random() — random float between 0 and 1
	i.globals.Define("math_random", &Value{Type: VAL_BUILTIN, BuiltinVal: func(args []*Value) (*Value, error) {
		return FloatVal(rand.Float64()), nil
	}}, false)

	// math_rand_int(min, max) — random int in range [min, max)
	i.globals.Define("math_rand_int", &Value{Type: VAL_BUILTIN, BuiltinVal: func(args []*Value) (*Value, error) {
		if len(args) != 2 || args[0].Type != VAL_INT || args[1].Type != VAL_INT {
			return nil, fmt.Errorf("math_rand_int() takes 2 int arguments")
		}
		mn := args[0].IntVal
		mx := args[1].IntVal
		if mn >= mx {
			return IntVal(mn), nil
		}
		return IntVal(mn + rand.Int63n(mx-mn)), nil
	}}, false)

	// math_max_int — max int64 value
	i.globals.Define("math_max_int", &Value{Type: VAL_BUILTIN, BuiltinVal: func(args []*Value) (*Value, error) {
		return IntVal(9223372036854775807), nil
	}}, false)

	// math_min_int
	i.globals.Define("math_min_int", &Value{Type: VAL_BUILTIN, BuiltinVal: func(args []*Value) (*Value, error) {
		return IntVal(-9223372036854775807), nil
	}}, false)

	// --- OS/System functions ---

	// os_setenv(key, value)
	i.globals.Define("os_setenv", &Value{Type: VAL_BUILTIN, BuiltinVal: func(args []*Value) (*Value, error) {
		if len(args) != 2 || args[0].Type != VAL_STRING || args[1].Type != VAL_STRING {
			return nil, fmt.Errorf("os_setenv() takes 2 string arguments")
		}
		return NullValue(), os.Setenv(args[0].StringVal, args[1].StringVal)
	}}, false)

	// os_cwd() — current working directory
	i.globals.Define("os_cwd", &Value{Type: VAL_BUILTIN, BuiltinVal: func(args []*Value) (*Value, error) {
		dir, err := os.Getwd()
		if err != nil {
			return nil, err
		}
		return StringVal(dir), nil
	}}, false)

	// os_hostname()
	i.globals.Define("os_hostname", &Value{Type: VAL_BUILTIN, BuiltinVal: func(args []*Value) (*Value, error) {
		name, err := os.Hostname()
		if err != nil {
			return nil, err
		}
		return StringVal(name), nil
	}}, false)

	// os_platform() — "windows", "linux", "darwin"
	i.globals.Define("os_platform", &Value{Type: VAL_BUILTIN, BuiltinVal: func(args []*Value) (*Value, error) {
		return StringVal(runtime.GOOS), nil
	}}, false)

	// os_arch() — "amd64", "arm64"
	i.globals.Define("os_arch", &Value{Type: VAL_BUILTIN, BuiltinVal: func(args []*Value) (*Value, error) {
		return StringVal(runtime.GOARCH), nil
	}}, false)

	// --- Regex built-ins ---

	// regex_match(pattern, str) — check if string matches regex
	i.globals.Define("regex_match", &Value{Type: VAL_BUILTIN, BuiltinVal: func(args []*Value) (*Value, error) {
		if len(args) != 2 || args[0].Type != VAL_STRING || args[1].Type != VAL_STRING {
			return nil, fmt.Errorf("regex_match() takes pattern and string")
		}
		matched, err := regexp.MatchString(args[0].StringVal, args[1].StringVal)
		if err != nil {
			return nil, fmt.Errorf("regex_match: %s", err)
		}
		return BoolValue(matched), nil
	}}, false)

	// regex_find(pattern, str) — find first match
	i.globals.Define("regex_find", &Value{Type: VAL_BUILTIN, BuiltinVal: func(args []*Value) (*Value, error) {
		if len(args) != 2 || args[0].Type != VAL_STRING || args[1].Type != VAL_STRING {
			return nil, fmt.Errorf("regex_find() takes pattern and string")
		}
		re, err := regexp.Compile(args[0].StringVal)
		if err != nil {
			return nil, fmt.Errorf("regex_find: %s", err)
		}
		match := re.FindString(args[1].StringVal)
		if match == "" {
			return NullValue(), nil
		}
		return StringVal(match), nil
	}}, false)

	// regex_find_all(pattern, str) — find all matches
	i.globals.Define("regex_find_all", &Value{Type: VAL_BUILTIN, BuiltinVal: func(args []*Value) (*Value, error) {
		if len(args) != 2 || args[0].Type != VAL_STRING || args[1].Type != VAL_STRING {
			return nil, fmt.Errorf("regex_find_all() takes pattern and string")
		}
		re, err := regexp.Compile(args[0].StringVal)
		if err != nil {
			return nil, fmt.Errorf("regex_find_all: %s", err)
		}
		matches := re.FindAllString(args[1].StringVal, -1)
		elems := make([]*Value, len(matches))
		for idx, m := range matches {
			elems[idx] = StringVal(m)
		}
		return ArrayValue(elems), nil
	}}, false)

	// regex_replace(pattern, str, replacement) — replace matches
	i.globals.Define("regex_replace", &Value{Type: VAL_BUILTIN, BuiltinVal: func(args []*Value) (*Value, error) {
		if len(args) != 3 || args[0].Type != VAL_STRING || args[1].Type != VAL_STRING || args[2].Type != VAL_STRING {
			return nil, fmt.Errorf("regex_replace() takes pattern, string, replacement")
		}
		re, err := regexp.Compile(args[0].StringVal)
		if err != nil {
			return nil, fmt.Errorf("regex_replace: %s", err)
		}
		return StringVal(re.ReplaceAllString(args[1].StringVal, args[2].StringVal)), nil
	}}, false)

	// --- Crypto built-ins ---

	// sha256(str) — returns SHA-256 hash as hex string
	i.globals.Define("sha256", &Value{Type: VAL_BUILTIN, BuiltinVal: func(args []*Value) (*Value, error) {
		if len(args) != 1 || args[0].Type != VAL_STRING {
			return nil, fmt.Errorf("sha256() takes 1 string argument")
		}
		h := sha256.Sum256([]byte(args[0].StringVal))
		return StringVal(hex.EncodeToString(h[:])), nil
	}}, false)

	// md5_hash(str) — returns MD5 hash as hex string
	i.globals.Define("md5_hash", &Value{Type: VAL_BUILTIN, BuiltinVal: func(args []*Value) (*Value, error) {
		if len(args) != 1 || args[0].Type != VAL_STRING {
			return nil, fmt.Errorf("md5_hash() takes 1 string argument")
		}
		h := md5.Sum([]byte(args[0].StringVal))
		return StringVal(hex.EncodeToString(h[:])), nil
	}}, false)

	// --- Encoding built-ins ---

	// base64_encode(str) — encode string to base64
	i.globals.Define("base64_encode", &Value{Type: VAL_BUILTIN, BuiltinVal: func(args []*Value) (*Value, error) {
		if len(args) != 1 || args[0].Type != VAL_STRING {
			return nil, fmt.Errorf("base64_encode() takes 1 string argument")
		}
		return StringVal(base64.StdEncoding.EncodeToString([]byte(args[0].StringVal))), nil
	}}, false)

	// base64_decode(str) — decode base64 string
	i.globals.Define("base64_decode", &Value{Type: VAL_BUILTIN, BuiltinVal: func(args []*Value) (*Value, error) {
		if len(args) != 1 || args[0].Type != VAL_STRING {
			return nil, fmt.Errorf("base64_decode() takes 1 string argument")
		}
		data, err := base64.StdEncoding.DecodeString(args[0].StringVal)
		if err != nil {
			return nil, fmt.Errorf("base64_decode: %s", err)
		}
		return StringVal(string(data)), nil
	}}, false)

	// url_encode(str)
	i.globals.Define("url_encode", &Value{Type: VAL_BUILTIN, BuiltinVal: func(args []*Value) (*Value, error) {
		if len(args) != 1 || args[0].Type != VAL_STRING {
			return nil, fmt.Errorf("url_encode() takes 1 string argument")
		}
		return StringVal(url.QueryEscape(args[0].StringVal)), nil
	}}, false)

	// url_decode(str)
	i.globals.Define("url_decode", &Value{Type: VAL_BUILTIN, BuiltinVal: func(args []*Value) (*Value, error) {
		if len(args) != 1 || args[0].Type != VAL_STRING {
			return nil, fmt.Errorf("url_decode() takes 1 string argument")
		}
		decoded, err := url.QueryUnescape(args[0].StringVal)
		if err != nil {
			return nil, fmt.Errorf("url_decode: %s", err)
		}
		return StringVal(decoded), nil
	}}, false)

	// type_of(x) — alias for type(), returns the type name as a string
	i.globals.Define("type_of", &Value{Type: VAL_BUILTIN, BuiltinVal: func(args []*Value) (*Value, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("type_of() takes exactly 1 argument, got %d", len(args))
		}
		return StringVal(args[0].Type.String()), nil
	}}, false)

	// format(template, args...) — replace {} placeholders with arguments
	i.globals.Define("format", &Value{Type: VAL_BUILTIN, BuiltinVal: func(args []*Value) (*Value, error) {
		if len(args) == 0 {
			return StringVal(""), nil
		}
		result := args[0].String()
		for idx := 1; idx < len(args); idx++ {
			result = strings.Replace(result, "{}", args[idx].String(), 1)
		}
		return StringVal(result), nil
	}}, false)

	// printf(template, args...) — formatted print
	i.globals.Define("printf", &Value{Type: VAL_BUILTIN, BuiltinVal: func(args []*Value) (*Value, error) {
		if len(args) == 0 {
			return NullValue(), nil
		}
		result := args[0].String()
		for idx := 1; idx < len(args); idx++ {
			result = strings.Replace(result, "{}", args[idx].String(), 1)
		}
		fmt.Print(result)
		i.output = append(i.output, result)
		return NullValue(), nil
	}}, false)

	// --- HTTP Server built-ins ---

	// http_serve(addr, handler) — start HTTP server. handler is a function that receives (path, method) and returns response string
	i.globals.Define("http_serve", &Value{Type: VAL_BUILTIN, BuiltinVal: func(args []*Value) (*Value, error) {
		if len(args) != 2 || args[0].Type != VAL_STRING || (args[1].Type != VAL_FUNCTION && args[1].Type != VAL_BUILTIN) {
			return nil, fmt.Errorf("http_serve() takes address string and handler function")
		}
		addr := args[0].StringVal
		handler := args[1]

		http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			// Call the handler with path and method
			handlerArgs := []*Value{StringVal(r.URL.Path), StringVal(r.Method)}
			var result *Value
			var err error
			if handler.Type == VAL_BUILTIN {
				result, err = handler.BuiltinVal(handlerArgs)
			} else {
				fn := handler.FuncVal
				funcEnv := NewEnclosedEnvironment(fn.Closure)
				for idx, pName := range fn.ParamNames {
					if idx < len(handlerArgs) {
						funcEnv.Define(pName, handlerArgs[idx], true)
					}
				}
				body := fn.Body.(*parser.BlockStatement)
				val, execErr := i.execBlock(body, funcEnv)
				if execErr != nil {
					http.Error(w, execErr.Error(), 500)
					return
				}
				if val != nil && val.Type == VAL_RETURN {
					result = val.ReturnVal
				} else {
					result = val
				}
				err = execErr
			}
			if err != nil {
				http.Error(w, err.Error(), 500)
				return
			}
			if result != nil {
				w.Header().Set("Content-Type", "text/plain")
				fmt.Fprint(w, result.String())
			}
		})

		line := fmt.Sprintf("Server listening on %s", addr)
		fmt.Println(line)
		i.output = append(i.output, line)
		return NullValue(), http.ListenAndServe(addr, nil)
	}}, false)

	// http_post(url, body, content_type) — HTTP POST request
	i.globals.Define("http_post", &Value{Type: VAL_BUILTIN, BuiltinVal: func(args []*Value) (*Value, error) {
		if len(args) < 2 || args[0].Type != VAL_STRING || args[1].Type != VAL_STRING {
			return nil, fmt.Errorf("http_post() takes url, body, and optional content_type")
		}
		contentType := "application/json"
		if len(args) > 2 && args[2].Type == VAL_STRING {
			contentType = args[2].StringVal
		}
		resp, err := http.Post(args[0].StringVal, contentType, strings.NewReader(args[1].StringVal))
		if err != nil {
			return nil, fmt.Errorf("http_post: %s", err)
		}
		defer resp.Body.Close()
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("http_post: %s", err)
		}
		return StringVal(string(body)), nil
	}}, false)

	// --- TCP Socket built-ins ---

	// tcp_listen(addr) — start TCP listener, returns listener ID
	i.globals.Define("tcp_listen", &Value{Type: VAL_BUILTIN, BuiltinVal: func(args []*Value) (*Value, error) {
		if len(args) != 1 || args[0].Type != VAL_STRING {
			return nil, fmt.Errorf("tcp_listen() takes 1 address string")
		}
		ln, err := net.Listen("tcp", args[0].StringVal)
		if err != nil {
			return nil, fmt.Errorf("tcp_listen: %s", err)
		}
		tcpListenerCounter++
		tcpListeners[tcpListenerCounter] = ln
		return IntVal(tcpListenerCounter), nil
	}}, false)

	// tcp_accept(listener_id) — accept connection, returns conn ID
	i.globals.Define("tcp_accept", &Value{Type: VAL_BUILTIN, BuiltinVal: func(args []*Value) (*Value, error) {
		if len(args) != 1 || args[0].Type != VAL_INT {
			return nil, fmt.Errorf("tcp_accept() takes listener ID")
		}
		ln, ok := tcpListeners[args[0].IntVal]
		if !ok {
			return nil, fmt.Errorf("invalid listener")
		}
		conn, err := ln.Accept()
		if err != nil {
			return nil, fmt.Errorf("tcp_accept: %s", err)
		}
		tcpConnCounter++
		tcpConns[tcpConnCounter] = conn
		return IntVal(tcpConnCounter), nil
	}}, false)

	// tcp_connect(addr) — connect to TCP server
	i.globals.Define("tcp_connect", &Value{Type: VAL_BUILTIN, BuiltinVal: func(args []*Value) (*Value, error) {
		if len(args) != 1 || args[0].Type != VAL_STRING {
			return nil, fmt.Errorf("tcp_connect() takes 1 address string")
		}
		conn, err := net.Dial("tcp", args[0].StringVal)
		if err != nil {
			return nil, fmt.Errorf("tcp_connect: %s", err)
		}
		tcpConnCounter++
		tcpConns[tcpConnCounter] = conn
		return IntVal(tcpConnCounter), nil
	}}, false)

	// tcp_send(conn_id, data) — send data
	i.globals.Define("tcp_send", &Value{Type: VAL_BUILTIN, BuiltinVal: func(args []*Value) (*Value, error) {
		if len(args) != 2 || args[0].Type != VAL_INT || args[1].Type != VAL_STRING {
			return nil, fmt.Errorf("tcp_send() takes conn ID and string data")
		}
		conn, ok := tcpConns[args[0].IntVal]
		if !ok {
			return nil, fmt.Errorf("invalid connection")
		}
		_, err := conn.Write([]byte(args[1].StringVal))
		if err != nil {
			return nil, fmt.Errorf("tcp_send: %s", err)
		}
		return NullValue(), nil
	}}, false)

	// tcp_recv(conn_id, max_bytes) — receive data
	i.globals.Define("tcp_recv", &Value{Type: VAL_BUILTIN, BuiltinVal: func(args []*Value) (*Value, error) {
		if len(args) < 1 || args[0].Type != VAL_INT {
			return nil, fmt.Errorf("tcp_recv() takes conn ID and optional max bytes")
		}
		conn, ok := tcpConns[args[0].IntVal]
		if !ok {
			return nil, fmt.Errorf("invalid connection")
		}
		maxBytes := 4096
		if len(args) > 1 && args[1].Type == VAL_INT {
			maxBytes = int(args[1].IntVal)
		}
		buf := make([]byte, maxBytes)
		n, err := conn.Read(buf)
		if err != nil && n == 0 {
			return nil, fmt.Errorf("tcp_recv: %s", err)
		}
		return StringVal(string(buf[:n])), nil
	}}, false)

	// tcp_close(conn_or_listener_id) — close connection or listener
	i.globals.Define("tcp_close", &Value{Type: VAL_BUILTIN, BuiltinVal: func(args []*Value) (*Value, error) {
		if len(args) != 1 || args[0].Type != VAL_INT {
			return nil, fmt.Errorf("tcp_close() takes 1 ID")
		}
		id := args[0].IntVal
		if conn, ok := tcpConns[id]; ok {
			conn.Close()
			delete(tcpConns, id)
		}
		if ln, ok := tcpListeners[id]; ok {
			ln.Close()
			delete(tcpListeners, id)
		}
		return NullValue(), nil
	}}, false)

	// --- Timers / Tickers ---

	// set_timeout(func, ms) — run function after delay (non-blocking)
	i.globals.Define("set_timeout", &Value{Type: VAL_BUILTIN, BuiltinVal: func(args []*Value) (*Value, error) {
		if len(args) != 2 || args[1].Type != VAL_INT {
			return nil, fmt.Errorf("set_timeout() takes function and milliseconds")
		}
		fn := args[0]
		ms := args[1].IntVal
		go func() {
			time.Sleep(time.Duration(ms) * time.Millisecond)
			if fn.Type == VAL_BUILTIN {
				fn.BuiltinVal(nil)
			} else if fn.Type == VAL_FUNCTION {
				body := fn.FuncVal.Body.(*parser.BlockStatement)
				env := NewEnclosedEnvironment(fn.FuncVal.Closure)
				i.execBlock(body, env)
			}
		}()
		return NullValue(), nil
	}}, false)

	// set_interval(func, ms) — run function repeatedly every ms milliseconds
	i.globals.Define("set_interval", &Value{Type: VAL_BUILTIN, BuiltinVal: func(args []*Value) (*Value, error) {
		if len(args) != 2 || args[1].Type != VAL_INT {
			return nil, fmt.Errorf("set_interval() takes function and milliseconds")
		}
		fn := args[0]
		ms := args[1].IntVal
		stop := make(chan *Value, 1)
		go func() {
			ticker := time.NewTicker(time.Duration(ms) * time.Millisecond)
			defer ticker.Stop()
			for {
				select {
				case <-ticker.C:
					if fn.Type == VAL_BUILTIN {
						fn.BuiltinVal(nil)
					} else if fn.Type == VAL_FUNCTION {
						body := fn.FuncVal.Body.(*parser.BlockStatement)
						env := NewEnclosedEnvironment(fn.FuncVal.Closure)
						i.execBlock(body, env)
					}
				case <-stop:
					return
				}
			}
		}()
		return &Value{Type: VAL_CHANNEL, ChannelVal: stop}, nil
	}}, false)

	// clear_interval(ch) — stop an interval by sending to its stop channel
	i.globals.Define("clear_interval", &Value{Type: VAL_BUILTIN, BuiltinVal: func(args []*Value) (*Value, error) {
		if len(args) != 1 || args[0].Type != VAL_CHANNEL {
			return nil, fmt.Errorf("clear_interval() takes a channel (from set_interval)")
		}
		args[0].ChannelVal <- NullValue()
		return NullValue(), nil
	}}, false)

	// --- Sort with custom comparator ---

	// sort_by(arr, comparator) — sort array using custom comparator function
	i.globals.Define("sort_by", &Value{Type: VAL_BUILTIN, BuiltinVal: func(args []*Value) (*Value, error) {
		if len(args) != 2 || args[0].Type != VAL_ARRAY {
			return nil, fmt.Errorf("sort_by() takes array and comparator function")
		}
		arr := make([]*Value, len(args[0].ArrayVal))
		copy(arr, args[0].ArrayVal)
		cmp := args[1]

		// Bubble sort with custom comparator
		for idx := 0; idx < len(arr); idx++ {
			for j := 0; j < len(arr)-1-idx; j++ {
				var result *Value
				var err error
				cmpArgs := []*Value{arr[j], arr[j+1]}
				if cmp.Type == VAL_BUILTIN {
					result, err = cmp.BuiltinVal(cmpArgs)
				} else {
					fn := cmp.FuncVal
					funcEnv := NewEnclosedEnvironment(fn.Closure)
					for k, pName := range fn.ParamNames {
						if k < len(cmpArgs) {
							funcEnv.Define(pName, cmpArgs[k], true)
						}
					}
					body := fn.Body.(*parser.BlockStatement)
					val, execErr := i.execBlock(body, funcEnv)
					if execErr != nil {
						return nil, execErr
					}
					if val != nil && val.Type == VAL_RETURN {
						result = val.ReturnVal
					} else {
						result = val
					}
					err = nil
				}
				if err != nil {
					return nil, err
				}
				if result != nil && result.Type == VAL_INT && result.IntVal > 0 {
					arr[j], arr[j+1] = arr[j+1], arr[j]
				}
			}
		}
		return ArrayValue(arr), nil
	}}, false)

	// --- Error wrapping ---

	// error_new(message) — create an error value
	i.globals.Define("error_new", &Value{Type: VAL_BUILTIN, BuiltinVal: func(args []*Value) (*Value, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("error_new() takes 1 argument")
		}
		pairs := make(map[string]*Value)
		keys := []string{"message", "type"}
		pairs["message"] = StringVal(args[0].String())
		pairs["type"] = StringVal("error")
		return MapVal(pairs, keys), nil
	}}, false)

	// error_wrap(err, message) — wrap an error with additional context
	i.globals.Define("error_wrap", &Value{Type: VAL_BUILTIN, BuiltinVal: func(args []*Value) (*Value, error) {
		if len(args) != 2 {
			return nil, fmt.Errorf("error_wrap() takes error and message")
		}
		innerMsg := ""
		if args[0].Type == VAL_MAP {
			if m, ok := args[0].MapVal.Pairs["message"]; ok {
				innerMsg = m.String()
			}
		} else {
			innerMsg = args[0].String()
		}
		wrapped := args[1].String() + ": " + innerMsg
		pairs := make(map[string]*Value)
		keys := []string{"message", "type", "cause"}
		pairs["message"] = StringVal(wrapped)
		pairs["type"] = StringVal("error")
		pairs["cause"] = args[0]
		return MapVal(pairs, keys), nil
	}}, false)

	// is_err(value) — check if value is an error (map with type="error")
	i.globals.Define("is_err", &Value{Type: VAL_BUILTIN, BuiltinVal: func(args []*Value) (*Value, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("is_err() takes 1 argument")
		}
		if args[0].Type == VAL_MAP {
			if t, ok := args[0].MapVal.Pairs["type"]; ok {
				return BoolValue(t.StringVal == "error"), nil
			}
		}
		return BoolValue(false), nil
	}}, false)

	// template(tmpl, data) — render template string with {{key}} placeholders from a map
	i.globals.Define("template", &Value{Type: VAL_BUILTIN, BuiltinVal: func(args []*Value) (*Value, error) {
		if len(args) != 2 || args[0].Type != VAL_STRING || args[1].Type != VAL_MAP {
			return nil, fmt.Errorf("template() takes template string and data map")
		}
		result := args[0].StringVal
		for _, key := range args[1].MapVal.Keys {
			val := args[1].MapVal.Pairs[key]
			placeholder := "{{" + key + "}}"
			result = strings.ReplaceAll(result, placeholder, val.String())
		}
		return StringVal(result), nil
	}}, false)

	// template_file(path, data) — load template from file and render
	i.globals.Define("template_file", &Value{Type: VAL_BUILTIN, BuiltinVal: func(args []*Value) (*Value, error) {
		if len(args) != 2 || args[0].Type != VAL_STRING || args[1].Type != VAL_MAP {
			return nil, fmt.Errorf("template_file() takes file path and data map")
		}
		content, err := os.ReadFile(args[0].StringVal)
		if err != nil {
			return nil, fmt.Errorf("template_file: %s", err)
		}
		result := string(content)
		for _, key := range args[1].MapVal.Keys {
			val := args[1].MapVal.Pairs[key]
			placeholder := "{{" + key + "}}"
			result = strings.ReplaceAll(result, placeholder, val.String())
		}
		return StringVal(result), nil
	}}, false)

	// fields(struct_val) — returns array of field names for a struct
	i.globals.Define("fields", &Value{Type: VAL_BUILTIN, BuiltinVal: func(args []*Value) (*Value, error) {
		if len(args) != 1 || args[0].Type != VAL_STRUCT {
			return nil, fmt.Errorf("fields() takes 1 struct argument")
		}
		var names []*Value
		for name := range args[0].StructVal.Fields {
			names = append(names, StringVal(name))
		}
		return ArrayValue(names), nil
	}}, false)

	// get_field(struct, name) — get field value by name string
	i.globals.Define("get_field", &Value{Type: VAL_BUILTIN, BuiltinVal: func(args []*Value) (*Value, error) {
		if len(args) != 2 || args[0].Type != VAL_STRUCT || args[1].Type != VAL_STRING {
			return nil, fmt.Errorf("get_field() takes struct and field name string")
		}
		val, ok := args[0].StructVal.Fields[args[1].StringVal]
		if !ok {
			return NullValue(), nil
		}
		return val, nil
	}}, false)

	// set_field(struct, name, value) — set field value by name string
	i.globals.Define("set_field", &Value{Type: VAL_BUILTIN, BuiltinVal: func(args []*Value) (*Value, error) {
		if len(args) != 3 || args[0].Type != VAL_STRUCT || args[1].Type != VAL_STRING {
			return nil, fmt.Errorf("set_field() takes struct, field name, and value")
		}
		args[0].StructVal.Fields[args[1].StringVal] = args[2]
		return NullValue(), nil
	}}, false)

	// type_name(val) — returns the struct type name or primitive type
	i.globals.Define("type_name", &Value{Type: VAL_BUILTIN, BuiltinVal: func(args []*Value) (*Value, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("type_name() takes 1 argument")
		}
		if args[0].Type == VAL_STRUCT {
			return StringVal(args[0].StructVal.TypeName), nil
		}
		return StringVal(args[0].Type.String()), nil
	}}, false)

	// is_null(val) — check if value is null
	i.globals.Define("is_null", &Value{Type: VAL_BUILTIN, BuiltinVal: func(args []*Value) (*Value, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("is_null() takes 1 argument")
		}
		return BoolValue(args[0].Type == VAL_NULL), nil
	}}, false)

	// is_int(val), is_float(val), is_string(val), is_bool(val), is_array(val), is_map(val), is_func(val)
	for _, check := range []struct {
		name string
		vt   ValueType
	}{
		{"is_int", VAL_INT}, {"is_float", VAL_FLOAT}, {"is_string", VAL_STRING},
		{"is_bool", VAL_BOOL}, {"is_array", VAL_ARRAY}, {"is_map", VAL_MAP},
		{"is_func", VAL_FUNCTION},
	} {
		vt := check.vt
		i.globals.Define(check.name, &Value{Type: VAL_BUILTIN, BuiltinVal: func(args []*Value) (*Value, error) {
			if len(args) != 1 {
				return nil, fmt.Errorf("type check function takes 1 argument")
			}
			return BoolValue(args[0].Type == vt), nil
		}}, false)
	}

	// to_map(struct) — convert struct to map
	i.globals.Define("to_map", &Value{Type: VAL_BUILTIN, BuiltinVal: func(args []*Value) (*Value, error) {
		if len(args) != 1 || args[0].Type != VAL_STRUCT {
			return nil, fmt.Errorf("to_map() takes 1 struct argument")
		}
		pairs := make(map[string]*Value)
		var keys []string
		for k, v := range args[0].StructVal.Fields {
			pairs[k] = v
			keys = append(keys, k)
		}
		return MapVal(pairs, keys), nil
	}}, false)

	// copy_val(val) — shallow copy of array/map/struct
	i.globals.Define("copy_val", &Value{Type: VAL_BUILTIN, BuiltinVal: func(args []*Value) (*Value, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("copy_val() takes 1 argument")
		}
		switch args[0].Type {
		case VAL_ARRAY:
			newArr := make([]*Value, len(args[0].ArrayVal))
			copy(newArr, args[0].ArrayVal)
			return ArrayValue(newArr), nil
		case VAL_MAP:
			newPairs := make(map[string]*Value)
			newKeys := make([]string, len(args[0].MapVal.Keys))
			copy(newKeys, args[0].MapVal.Keys)
			for k, v := range args[0].MapVal.Pairs {
				newPairs[k] = v
			}
			return MapVal(newPairs, newKeys), nil
		case VAL_STRUCT:
			newFields := make(map[string]*Value)
			for k, v := range args[0].StructVal.Fields {
				newFields[k] = v
			}
			return &Value{Type: VAL_STRUCT, StructVal: &StructValue{TypeName: args[0].StructVal.TypeName, Fields: newFields}}, nil
		default:
			return args[0], nil // primitives are already values
		}
	}}, false)
}

// AddTestBuiltins adds assert functions for test files.
func (i *Interpreter) AddTestBuiltins() {
	i.globals.Define("assert", &Value{Type: VAL_BUILTIN, BuiltinVal: func(args []*Value) (*Value, error) {
		if len(args) < 1 {
			return nil, fmt.Errorf("assert() takes at least 1 argument")
		}
		if !args[0].IsTruthy() {
			msg := "assertion failed"
			if len(args) > 1 {
				msg = args[1].String()
			}
			return nil, fmt.Errorf("%s", msg)
		}
		return NullValue(), nil
	}}, false)

	i.globals.Define("assert_eq", &Value{Type: VAL_BUILTIN, BuiltinVal: func(args []*Value) (*Value, error) {
		if len(args) != 2 {
			return nil, fmt.Errorf("assert_eq() takes 2 arguments")
		}
		if !valuesEqual(args[0], args[1]) {
			return nil, fmt.Errorf("assert_eq failed: %s != %s", args[0].Inspect(), args[1].Inspect())
		}
		return NullValue(), nil
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
	case *parser.XuiorClassicStatement:
		return i.execXuiorClassic(s, env)
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
	case *parser.XudeferStatement:
		i.deferStack = append(i.deferStack, deferredCall{expr: s.Call, env: env})
		return nil, nil
	case *parser.XuselectStatement:
		return i.execXuselect(s, env)
	case *parser.ExpressionStatement:
		return i.evalExpression(s.Expr, env)
	case *parser.BlockStatement:
		return i.execBlock(s, env)
	default:
		return nil, fmt.Errorf("unknown statement type: %T", stmt)
	}
}

func (i *Interpreter) execXuselect(s *parser.XuselectStatement, env *Environment) (*Value, error) {
	var selectCases []reflect.SelectCase
	var bodyMap []int // maps reflect select case index to our case index

	for idx, c := range s.Cases {
		if c.IsDefault {
			selectCases = append(selectCases, reflect.SelectCase{
				Dir: reflect.SelectDefault,
			})
			bodyMap = append(bodyMap, idx)
		} else {
			chVal, err := i.evalExpression(c.Channel, env)
			if err != nil {
				return nil, err
			}
			if chVal.Type != VAL_CHANNEL {
				return nil, fmt.Errorf("xuselect case must be a channel, got %s", chVal.Type)
			}
			selectCases = append(selectCases, reflect.SelectCase{
				Dir:  reflect.SelectRecv,
				Chan: reflect.ValueOf(chVal.ChannelVal),
			})
			bodyMap = append(bodyMap, idx)
		}
	}

	chosen, recvVal, _ := reflect.Select(selectCases)
	caseIdx := bodyMap[chosen]
	c := s.Cases[caseIdx]

	blockEnv := NewEnclosedEnvironment(env)

	// If we received a value from a channel, bind it as "it"
	if !c.IsDefault && recvVal.IsValid() {
		if val, ok := recvVal.Interface().(*Value); ok {
			blockEnv.Define("it", val, false)
		}
	}

	return i.execBlock(c.Body, blockEnv)
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

func (i *Interpreter) execXuiorClassic(s *parser.XuiorClassicStatement, env *Environment) (*Value, error) {
	loopEnv := NewEnclosedEnvironment(env)

	// Execute init
	if s.Init != nil {
		_, err := i.execStatement(s.Init, loopEnv)
		if err != nil {
			return nil, err
		}
	}

	for {
		// Check condition
		cond, err := i.evalExpression(s.Condition, loopEnv)
		if err != nil {
			return nil, err
		}
		if !cond.IsTruthy() {
			break
		}

		// Execute body
		bodyEnv := NewEnclosedEnvironment(loopEnv)
		val, err := i.execBlock(s.Body, bodyEnv)
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

		// Execute post
		if s.Post != nil {
			_, err := i.execStatement(s.Post, loopEnv)
			if err != nil {
				return nil, err
			}
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
		if ok {
			return val, nil
		}
		// Struct embedding: search embedded struct fields
		for _, field := range obj.StructVal.Fields {
			if field.Type == VAL_STRUCT {
				if embedded, ok := field.StructVal.Fields[e.Member]; ok {
					return embedded, nil
				}
			}
		}
		return nil, fmt.Errorf("%s: struct %q has no field %q", e.TokenPos(), obj.StructVal.TypeName, e.Member)
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
