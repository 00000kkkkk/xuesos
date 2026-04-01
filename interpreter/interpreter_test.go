package interpreter

import (
	"strings"
	"testing"

	"github.com/00000kkkkk/xusesosplusplus/lexer"
	"github.com/00000kkkkk/xusesosplusplus/parser"
)

func run(t *testing.T, src string) *Interpreter {
	t.Helper()
	l := lexer.New("test.xpp", src)
	tokens, lexErrs := l.ScanAll()
	if len(lexErrs) > 0 {
		t.Fatalf("lexer errors: %v", lexErrs)
	}
	p := parser.New(tokens)
	prog, parseErrs := p.Parse()
	if len(parseErrs) > 0 {
		t.Fatalf("parser errors: %v", parseErrs)
	}
	interp := New()
	if err := interp.Run(prog); err != nil {
		t.Fatalf("runtime error: %v", err)
	}
	return interp
}

func runExpectError(t *testing.T, src string) error {
	t.Helper()
	l := lexer.New("test.xpp", src)
	tokens, _ := l.ScanAll()
	p := parser.New(tokens)
	prog, _ := p.Parse()
	interp := New()
	return interp.Run(prog)
}

func expectOutput(t *testing.T, interp *Interpreter, expected ...string) {
	t.Helper()
	output := interp.Output()
	if len(output) != len(expected) {
		t.Fatalf("expected %d output lines, got %d:\n%v", len(expected), len(output), output)
	}
	for i, exp := range expected {
		if output[i] != exp {
			t.Errorf("output[%d]: expected %q, got %q", i, exp, output[i])
		}
	}
}

// --- Basic tests ---

func TestPrintHello(t *testing.T) {
	interp := run(t, `print("Hello, Xuesos++!")`)
	expectOutput(t, interp, "Hello, Xuesos++!")
}

func TestPrintMultipleArgs(t *testing.T) {
	interp := run(t, `print("hello", "world")`)
	expectOutput(t, interp, "hello world")
}

func TestXuetVariable(t *testing.T) {
	interp := run(t, `
		xuet x = 42
		print(x)
	`)
	expectOutput(t, interp, "42")
}

func TestXuiarVariable(t *testing.T) {
	interp := run(t, `
		xuiar x = 10
		x = 20
		print(x)
	`)
	expectOutput(t, interp, "20")
}

func TestImmutableError(t *testing.T) {
	err := runExpectError(t, `
		xuet x = 10
		x = 20
	`)
	if err == nil {
		t.Fatal("expected error for immutable reassignment")
	}
	if !strings.Contains(err.Error(), "immutable") {
		t.Errorf("expected immutable error, got: %v", err)
	}
}

// --- Arithmetic ---

func TestArithmetic(t *testing.T) {
	interp := run(t, `
		print(2 + 3)
		print(10 - 4)
		print(3 * 7)
		print(15 / 3)
		print(17 % 5)
	`)
	expectOutput(t, interp, "5", "6", "21", "5", "2")
}

func TestFloatArithmetic(t *testing.T) {
	interp := run(t, `
		print(2.5 + 1.5)
		print(2.5 * 2.0)
	`)
	expectOutput(t, interp, "4", "5")
}

func TestIntFloatMix(t *testing.T) {
	interp := run(t, `print(5 + 2.5)`)
	expectOutput(t, interp, "7.5")
}

func TestNegation(t *testing.T) {
	interp := run(t, `print(-42)`)
	expectOutput(t, interp, "-42")
}

func TestStringConcat(t *testing.T) {
	interp := run(t, `print("hello" + " " + "world")`)
	expectOutput(t, interp, "hello world")
}

func TestStringIntConcat(t *testing.T) {
	interp := run(t, `print("age: " + 25)`)
	expectOutput(t, interp, "age: 25")
}

// --- Comparison ---

func TestComparison(t *testing.T) {
	interp := run(t, `
		print(5 == 5)
		print(5 != 3)
		print(3 < 5)
		print(5 > 3)
		print(5 <= 5)
		print(5 >= 6)
	`)
	expectOutput(t, interp, "xuitru", "xuitru", "xuitru", "xuitru", "xuitru", "xuinia")
}

// --- Logical ---

func TestLogical(t *testing.T) {
	interp := run(t, `
		print(xuitru && xuitru)
		print(xuitru && xuinia)
		print(xuinia || xuitru)
		print(xuinia || xuinia)
		print(!xuitru)
		print(!xuinia)
	`)
	expectOutput(t, interp, "xuitru", "xuinia", "xuitru", "xuinia", "xuinia", "xuitru")
}

func TestShortCircuit(t *testing.T) {
	// If short-circuit works, this shouldn't crash on undefined variable
	interp := run(t, `
		print(xuinia && undefined_var)
		print(xuitru || undefined_var)
	`)
	expectOutput(t, interp, "xuinia", "xuitru")
}

// --- If/Else ---

func TestXuifTrue(t *testing.T) {
	interp := run(t, `
		xuif (5 > 3) {
			print("yes")
		}
	`)
	expectOutput(t, interp, "yes")
}

func TestXuifFalse(t *testing.T) {
	interp := run(t, `
		xuif (5 < 3) {
			print("yes")
		}
	`)
	expectOutput(t, interp)
}

func TestXuifXuelse(t *testing.T) {
	interp := run(t, `
		xuif (5 < 3) {
			print("yes")
		} xuelse {
			print("no")
		}
	`)
	expectOutput(t, interp, "no")
}

func TestXuifElseIf(t *testing.T) {
	interp := run(t, `
		xuet x = 15
		xuif (x > 20) {
			print("big")
		} xuelse xuif (x > 10) {
			print("medium")
		} xuelse {
			print("small")
		}
	`)
	expectOutput(t, interp, "medium")
}

// --- Loops ---

func TestXuiorRange(t *testing.T) {
	interp := run(t, `
		xuior (i xuin 0..5) {
			print(i)
		}
	`)
	expectOutput(t, interp, "0", "1", "2", "3", "4")
}

func TestXuiorArray(t *testing.T) {
	interp := run(t, `
		xuet nums = [10, 20, 30]
		xuior (n xuin nums) {
			print(n)
		}
	`)
	expectOutput(t, interp, "10", "20", "30")
}

func TestXuiorString(t *testing.T) {
	interp := run(t, `
		xuior (ch xuin "abc") {
			print(ch)
		}
	`)
	expectOutput(t, interp, "a", "b", "c")
}

func TestXuile(t *testing.T) {
	interp := run(t, `
		xuiar i = 0
		xuile (i < 3) {
			print(i)
			i = i + 1
		}
	`)
	expectOutput(t, interp, "0", "1", "2")
}

func TestXuieak(t *testing.T) {
	interp := run(t, `
		xuior (i xuin 0..10) {
			xuif (i == 3) {
				xuieak
			}
			print(i)
		}
	`)
	expectOutput(t, interp, "0", "1", "2")
}

func TestXuitinue(t *testing.T) {
	interp := run(t, `
		xuior (i xuin 0..5) {
			xuif (i == 2) {
				xuitinue
			}
			print(i)
		}
	`)
	expectOutput(t, interp, "0", "1", "3", "4")
}

// --- Functions ---

func TestXuenBasic(t *testing.T) {
	interp := run(t, `
		xuen greet() {
			print("hello")
		}
		greet()
	`)
	expectOutput(t, interp, "hello")
}

func TestXuenWithParams(t *testing.T) {
	interp := run(t, `
		xuen add(a int, b int) int {
			xueturn a + b
		}
		print(add(3, 4))
	`)
	expectOutput(t, interp, "7")
}

func TestRecursion(t *testing.T) {
	interp := run(t, `
		xuen fib(n int) int {
			xuif (n <= 1) {
				xueturn n
			}
			xueturn fib(n - 1) + fib(n - 2)
		}
		print(fib(10))
	`)
	expectOutput(t, interp, "55")
}

func TestXueturnNoValue(t *testing.T) {
	interp := run(t, `
		xuen doNothing() {
			xueturn
		}
		xuet result = doNothing()
		print(result)
	`)
	expectOutput(t, interp, "xuinull")
}

func TestClosures(t *testing.T) {
	interp := run(t, `
		xuen makeCounter() {
			xuiar count = 0
			xuen increment() int {
				count = count + 1
				xueturn count
			}
			xueturn increment
		}
		xuet counter = makeCounter()
		print(counter())
		print(counter())
		print(counter())
	`)
	expectOutput(t, interp, "1", "2", "3")
}

// --- Arrays ---

func TestArrayLiteral(t *testing.T) {
	interp := run(t, `
		xuet arr = [1, 2, 3]
		print(arr)
	`)
	expectOutput(t, interp, "[1, 2, 3]")
}

func TestArrayIndex(t *testing.T) {
	interp := run(t, `
		xuet arr = [10, 20, 30]
		print(arr[0])
		print(arr[2])
	`)
	expectOutput(t, interp, "10", "30")
}

func TestArrayAssign(t *testing.T) {
	interp := run(t, `
		xuiar arr = [1, 2, 3]
		arr[1] = 99
		print(arr)
	`)
	expectOutput(t, interp, "[1, 99, 3]")
}

func TestArrayLength(t *testing.T) {
	interp := run(t, `
		xuet arr = [1, 2, 3, 4, 5]
		print(len(arr))
		print(arr.length)
	`)
	expectOutput(t, interp, "5", "5")
}

func TestAppend(t *testing.T) {
	interp := run(t, `
		xuiar arr = [1, 2]
		arr = append(arr, 3)
		print(arr)
	`)
	expectOutput(t, interp, "[1, 2, 3]")
}

// --- Structs ---

func TestStruct(t *testing.T) {
	interp := run(t, `
		xuiruct Point {
			x int
			y int
		}
		xuet p = Point { x = 10, y = 20 }
		print(p.x)
		print(p.y)
	`)
	expectOutput(t, interp, "10", "20")
}

func TestStructMethod(t *testing.T) {
	interp := run(t, `
		xuiruct Counter {
			value int
		}
		xuimpl Counter {
			xuen get(self) int {
				xueturn self.value
			}
			xuen increment(xuiar self) {
				self.value = self.value + 1
			}
		}
		xuiar c = Counter { value = 0 }
		c.increment()
		c.increment()
		c.increment()
		print(c.get())
	`)
	expectOutput(t, interp, "3")
}

// --- Enum ---

func TestEnum(t *testing.T) {
	interp := run(t, `
		xuenum Color {
			Red
			Green
			Blue
		}
		xuet c = Red
		print(c)
	`)
	expectOutput(t, interp, "Color.Red")
}

// --- Match ---

func TestXuiatch(t *testing.T) {
	interp := run(t, `
		xuet status = "ok"
		xuiatch (status) {
			"ok" => print("success")
			"error" => print("failure")
			_ => print("unknown")
		}
	`)
	expectOutput(t, interp, "success")
}

func TestXuiatchWildcard(t *testing.T) {
	interp := run(t, `
		xuet x = 42
		xuiatch (x) {
			1 => print("one")
			_ => print("other")
		}
	`)
	expectOutput(t, interp, "other")
}

// --- Built-in functions ---

func TestLen(t *testing.T) {
	interp := run(t, `
		print(len("hello"))
		print(len([1, 2, 3]))
	`)
	expectOutput(t, interp, "5", "3")
}

func TestTypeFunc(t *testing.T) {
	interp := run(t, `
		print(type(42))
		print(type("hello"))
		print(type(xuitru))
		print(type(xuinull))
	`)
	expectOutput(t, interp, "int", "str", "bool", "null")
}

func TestSqrt(t *testing.T) {
	interp := run(t, `print(sqrt(16.0))`)
	expectOutput(t, interp, "4")
}

func TestConversions(t *testing.T) {
	interp := run(t, `
		print(to_int(3.14))
		print(to_float(42))
		print(to_str(123))
	`)
	expectOutput(t, interp, "3", "42", "123")
}

// --- Null ---

func TestNull(t *testing.T) {
	interp := run(t, `
		xuet x = xuinull
		print(x)
		print(x == xuinull)
	`)
	expectOutput(t, interp, "xuinull", "xuitru")
}

// --- Division by zero ---

func TestDivisionByZero(t *testing.T) {
	err := runExpectError(t, `xuet x = 10 / 0`)
	if err == nil {
		t.Fatal("expected division by zero error")
	}
}

// --- Scope ---

func TestScope(t *testing.T) {
	interp := run(t, `
		xuet x = "outer"
		xuif (xuitru) {
			xuet x = "inner"
			print(x)
		}
		print(x)
	`)
	expectOutput(t, interp, "inner", "outer")
}

// --- Full programs ---

func TestHelloWorld(t *testing.T) {
	interp := run(t, `
		xuen main() {
			xuet message = "Hello, Xuesos++!"
			print(message)
		}
	`)
	expectOutput(t, interp, "Hello, Xuesos++!")
}

func TestFibonacci(t *testing.T) {
	interp := run(t, `
		xuen fibonacci(n int) int {
			xuif (n <= 1) {
				xueturn n
			}
			xueturn fibonacci(n - 1) + fibonacci(n - 2)
		}
		xuior (i xuin 0..10) {
			print(fibonacci(i))
		}
	`)
	expectOutput(t, interp, "0", "1", "1", "2", "3", "5", "8", "13", "21", "34")
}

func TestFizzBuzz(t *testing.T) {
	interp := run(t, `
		xuior (i xuin 1..16) {
			xuif (i % 15 == 0) {
				print("FizzBuzz")
			} xuelse xuif (i % 3 == 0) {
				print("Fizz")
			} xuelse xuif (i % 5 == 0) {
				print("Buzz")
			} xuelse {
				print(i)
			}
		}
	`)
	expectOutput(t, interp,
		"1", "2", "Fizz", "4", "Buzz", "Fizz", "7", "8", "Fizz", "Buzz",
		"11", "Fizz", "13", "14", "FizzBuzz",
	)
}

// --- New built-in function tests ---

func TestAbs(t *testing.T) {
	interp := run(t, `
		print(abs(-42))
		print(abs(10))
		print(abs(-3.14))
	`)
	expectOutput(t, interp, "42", "10", "3.14")
}

func TestMaxMin(t *testing.T) {
	interp := run(t, `
		print(max(5, 10))
		print(min(5, 10))
		print(max(-1, -5))
		print(min(3.14, 2.71))
	`)
	expectOutput(t, interp, "10", "5", "-1", "2.71")
}

func TestContains(t *testing.T) {
	interp := run(t, `
		print(contains("hello world", "world"))
		print(contains("hello", "xyz"))
	`)
	expectOutput(t, interp, "xuitru", "xuinia")
}

func TestSplit(t *testing.T) {
	interp := run(t, `
		xuet parts = split("a,b,c", ",")
		print(len(parts))
		print(parts[0])
		print(parts[2])
	`)
	expectOutput(t, interp, "3", "a", "c")
}

func TestTrim(t *testing.T) {
	interp := run(t, `print(trim("  hello  "))`)
	expectOutput(t, interp, "hello")
}

func TestReplace(t *testing.T) {
	interp := run(t, `print(replace("hello world", "world", "xuesos"))`)
	expectOutput(t, interp, "hello xuesos")
}

func TestUpperLower(t *testing.T) {
	interp := run(t, `
		print(upper("hello"))
		print(lower("HELLO"))
	`)
	expectOutput(t, interp, "HELLO", "hello")
}

func TestStartsEndsWith(t *testing.T) {
	interp := run(t, `
		print(starts_with("hello world", "hello"))
		print(starts_with("hello", "world"))
		print(ends_with("hello.xpp", ".xpp"))
		print(ends_with("hello.xpp", ".go"))
	`)
	expectOutput(t, interp, "xuitru", "xuinia", "xuitru", "xuinia")
}

func TestJoin(t *testing.T) {
	interp := run(t, `
		xuet arr = ["a", "b", "c"]
		print(join(arr, ", "))
		print(join(arr, "-"))
	`)
	expectOutput(t, interp, "a, b, c", "a-b-c")
}

func TestSlice(t *testing.T) {
	interp := run(t, `
		xuet arr = [10, 20, 30, 40, 50]
		xuet sub = slice(arr, 1, 4)
		print(sub)
		print(len(sub))
	`)
	expectOutput(t, interp, "[20, 30, 40]", "3")
}

func TestPush(t *testing.T) {
	interp := run(t, `
		xuiar arr = [1, 2, 3]
		push(arr, 4)
		push(arr, 5)
		print(arr)
		print(len(arr))
	`)
	expectOutput(t, interp, "[1, 2, 3, 4, 5]", "5")
}

func TestPushError(t *testing.T) {
	err := runExpectError(t, `push("hello", 1)`)
	if err == nil {
		t.Fatal("expected error for push() with non-array")
	}
}

func TestRangeArr(t *testing.T) {
	interp := run(t, `
		xuet arr = range_arr(0, 5)
		print(arr)
		print(len(arr))
		print(arr[0])
		print(arr[4])
	`)
	expectOutput(t, interp, "[0, 1, 2, 3, 4]", "5", "0", "4")
}

func TestRangeArrNegative(t *testing.T) {
	interp := run(t, `
		xuet arr = range_arr(-2, 3)
		print(arr)
	`)
	expectOutput(t, interp, "[-2, -1, 0, 1, 2]")
}

func TestRangeArrEmpty(t *testing.T) {
	interp := run(t, `
		xuet arr = range_arr(5, 3)
		print(len(arr))
	`)
	expectOutput(t, interp, "0")
}

func TestSliceString(t *testing.T) {
	interp := run(t, `
		print(slice("hello world", 0, 5))
		print(slice("hello world", 6, 11))
	`)
	expectOutput(t, interp, "hello", "world")
}

func TestSliceArrayBounds(t *testing.T) {
	interp := run(t, `
		xuet arr = [1, 2, 3]
		print(slice(arr, 0, 100))
		print(slice(arr, 5, 10))
	`)
	expectOutput(t, interp, "[1, 2, 3]", "[]")
}

func TestSliceStringBounds(t *testing.T) {
	interp := run(t, `
		print(slice("abc", 0, 100))
		print(slice("abc", 5, 10))
	`)
	expectOutput(t, interp, "abc", "")
}

func TestAbsError(t *testing.T) {
	err := runExpectError(t, `abs("hello")`)
	if err == nil {
		t.Fatal("expected error for abs() with string")
	}
}

func TestMaxError(t *testing.T) {
	err := runExpectError(t, `max("a", "b")`)
	if err == nil {
		t.Fatal("expected error for max() with strings")
	}
}

func TestMinError(t *testing.T) {
	err := runExpectError(t, `min("a", "b")`)
	if err == nil {
		t.Fatal("expected error for min() with strings")
	}
}

func TestContainsError(t *testing.T) {
	err := runExpectError(t, `contains(123, "a")`)
	if err == nil {
		t.Fatal("expected error for contains() with non-string")
	}
}

func TestSplitError(t *testing.T) {
	err := runExpectError(t, `split(123, ",")`)
	if err == nil {
		t.Fatal("expected error for split() with non-string")
	}
}

func TestTrimError(t *testing.T) {
	err := runExpectError(t, `trim(123)`)
	if err == nil {
		t.Fatal("expected error for trim() with non-string")
	}
}

func TestUpperError(t *testing.T) {
	err := runExpectError(t, `upper(123)`)
	if err == nil {
		t.Fatal("expected error for upper() with non-string")
	}
}

func TestLowerError(t *testing.T) {
	err := runExpectError(t, `lower(123)`)
	if err == nil {
		t.Fatal("expected error for lower() with non-string")
	}
}

func TestSplitJoinRoundtrip(t *testing.T) {
	interp := run(t, `
		xuet original = "hello world foo"
		xuet parts = split(original, " ")
		xuet result = join(parts, " ")
		print(result == original)
	`)
	expectOutput(t, interp, "xuitru")
}

func TestStringPipeline(t *testing.T) {
	interp := run(t, `
		xuet s = "  Hello, World!  "
		xuet trimmed = trim(s)
		xuet lowered = lower(trimmed)
		xuet replaced = replace(lowered, "world", "xuesos++")
		print(replaced)
	`)
	expectOutput(t, interp, "hello, xuesos++!")
}

func TestPushInLoop(t *testing.T) {
	interp := run(t, `
		xuiar result = []
		xuior (i xuin 0..5) {
			push(result, i * i)
		}
		print(result)
	`)
	expectOutput(t, interp, "[0, 1, 4, 9, 16]")
}

func TestRangeArrWithSlice(t *testing.T) {
	interp := run(t, `
		xuet arr = range_arr(0, 10)
		xuet middle = slice(arr, 3, 7)
		print(middle)
	`)
	expectOutput(t, interp, "[3, 4, 5, 6]")
}

func TestJoinInts(t *testing.T) {
	interp := run(t, `
		xuet nums = [1, 2, 3]
		print(join(nums, "+"))
	`)
	expectOutput(t, interp, "1+2+3")
}

// --- Map tests ---

func TestMapLiteral(t *testing.T) {
	interp := run(t, `
		xuet m = {"name": "xuesos", "version": "1.0"}
		print(m["name"])
		print(m["version"])
	`)
	expectOutput(t, interp, "xuesos", "1.0")
}

func TestMapAssign(t *testing.T) {
	interp := run(t, `
		xuiar m = {"x": 1}
		m["y"] = 2
		m["x"] = 10
		print(m["x"])
		print(m["y"])
	`)
	expectOutput(t, interp, "10", "2")
}

func TestMapLen(t *testing.T) {
	interp := run(t, `
		xuet m = {"a": 1, "b": 2, "c": 3}
		print(len(m))
	`)
	expectOutput(t, interp, "3")
}

func TestEmptyMap(t *testing.T) {
	interp := run(t, `
		xuiar m = {}
		m["key"] = "value"
		print(m["key"])
		print(len(m))
	`)
	expectOutput(t, interp, "value", "1")
}

func TestMapContains(t *testing.T) {
	interp := run(t, `
		xuet m = {"a": 1, "b": 2}
		print(has_key(m, "a"))
		print(has_key(m, "z"))
	`)
	expectOutput(t, interp, "xuitru", "xuinia")
}

func TestMapKeys(t *testing.T) {
	interp := run(t, `
		xuet m = {"x": 1, "y": 2}
		xuet k = keys(m)
		print(len(k))
	`)
	expectOutput(t, interp, "2")
}

func TestMapValues(t *testing.T) {
	interp := run(t, `
		xuet m = {"a": 10, "b": 20}
		xuet v = values(m)
		print(len(v))
	`)
	expectOutput(t, interp, "2")
}

func TestMapInLoop(t *testing.T) {
	interp := run(t, `
		xuet scores = {"alice": 90, "bob": 85}
		xuiar total = 0
		xuior (k xuin keys(scores)) {
			total = total + scores[k]
		}
		print(total)
	`)
	expectOutput(t, interp, "175")
}

// --- String interpolation tests ---

func TestStringInterpolation(t *testing.T) {
	interp := run(t, `
		xuet name = "Xuesos"
		print("Hello {name}!")
	`)
	expectOutput(t, interp, "Hello Xuesos!")
}

func TestStringInterpolationExpr(t *testing.T) {
	interp := run(t, `
		print("2 + 2 = {2 + 2}")
	`)
	expectOutput(t, interp, "2 + 2 = 4")
}

func TestStringInterpolationMultiple(t *testing.T) {
	interp := run(t, `
		xuet a = 10
		xuet b = 20
		print("{a} + {b} = {a + b}")
	`)
	expectOutput(t, interp, "10 + 20 = 30")
}

func TestStringInterpolationNested(t *testing.T) {
	interp := run(t, `
		xuet arr = [1, 2, 3]
		print("length is {len(arr)}")
	`)
	expectOutput(t, interp, "length is 3")
}

func TestStringNoInterpolation(t *testing.T) {
	interp := run(t, `print("no braces here")`)
	expectOutput(t, interp, "no braces here")
}

// --- Stdlib tests ---

func TestMathStdlib(t *testing.T) {
	interp := run(t, `
		xuimport "math"
		print(math_pi())
		print(math_floor(3.7))
		print(math_ceil(3.2))
		print(math_pow(2, 10))
	`)
	expectOutput(t, interp, "3.141592653589793", "3", "4", "1024")
}

func TestOsStdlib(t *testing.T) {
	interp := run(t, `
		xuimport "os"
		xuet args = os_args()
		print(type(args))
	`)
	expectOutput(t, interp, "array")
}

// --- Lambda tests ---

func TestLambdaSimple(t *testing.T) {
	interp := run(t, `
		xuet add = (a, b) => a + b
		print(add(3, 4))
	`)
	expectOutput(t, interp, "7")
}

func TestLambdaNoParams(t *testing.T) {
	interp := run(t, `
		xuet greet = () => "hello"
		print(greet())
	`)
	expectOutput(t, interp, "hello")
}

func TestLambdaBlock(t *testing.T) {
	interp := run(t, `
		xuet factorial = (n) => {
			xuif (n <= 1) {
				xueturn 1
			}
			xueturn n * factorial(n - 1)
		}
		print(factorial(5))
	`)
	expectOutput(t, interp, "120")
}

func TestLambdaAsArgument(t *testing.T) {
	interp := run(t, `
		xuen apply(f, x int, y int) int {
			xueturn f(x, y)
		}
		xuet result = apply((a, b) => a * b, 6, 7)
		print(result)
	`)
	expectOutput(t, interp, "42")
}

func TestLambdaClosure(t *testing.T) {
	interp := run(t, `
		xuet multiplier = 10
		xuet times = (x) => x * multiplier
		print(times(5))
	`)
	expectOutput(t, interp, "50")
}

// --- Try/Catch tests ---

func TestTryCatch(t *testing.T) {
	interp := run(t, `
		xutry {
			xuet x = 10 / 0
		} xucatch (e) {
			print("caught: " + e)
		}
	`)
	output := interp.Output()
	if len(output) != 1 {
		t.Fatalf("expected 1 output, got %d: %v", len(output), output)
	}
	if !strings.Contains(output[0], "caught:") {
		t.Errorf("expected caught error, got %q", output[0])
	}
}

func TestTryCatchNoError(t *testing.T) {
	interp := run(t, `
		xutry {
			print("ok")
		} xucatch (e) {
			print("error: " + e)
		}
	`)
	expectOutput(t, interp, "ok")
}

func TestThrow(t *testing.T) {
	interp := run(t, `
		xutry {
			xuthrow "something went wrong"
		} xucatch (e) {
			print(e)
		}
	`)
	expectOutput(t, interp, "something went wrong")
}

func TestThrowInFunction(t *testing.T) {
	interp := run(t, `
		xuen divide(a int, b int) int {
			xuif (b == 0) {
				xuthrow "division by zero!"
			}
			xueturn a / b
		}
		xutry {
			print(divide(10, 0))
		} xucatch (e) {
			print("error: " + e)
		}
	`)
	expectOutput(t, interp, "error: division by zero!")
}

// --- Concurrency ---

func TestSleep(t *testing.T) {
	interp := run(t, `
		sleep(1)
		print("done")
	`)
	expectOutput(t, interp, "done")
}

func TestWait(t *testing.T) {
	interp := run(t, `
		wait(1)
		print("done")
	`)
	expectOutput(t, interp, "done")
}

func TestChannelBasic(t *testing.T) {
	interp := run(t, `
		xuet ch = channel(1)
		send(ch, 42)
		xuet val = recv(ch)
		print(val)
	`)
	expectOutput(t, interp, "42")
}

func TestChannelStringValue(t *testing.T) {
	interp := run(t, `
		xuet ch = channel(1)
		send(ch, "hello")
		xuet val = recv(ch)
		print(val)
	`)
	expectOutput(t, interp, "hello")
}

func TestSpawnWithChannel(t *testing.T) {
	interp := run(t, `
		xuet ch = channel(0)
		spawn((  ) => {
			send(ch, 99)
		})
		xuet val = recv(ch)
		print(val)
	`)
	expectOutput(t, interp, "99")
}

// --- Pointers and memory management ---

func TestPointerBasic(t *testing.T) {
	interp := run(t, `
		xuiar x = 42
		xuet ptr = &x
		print(*ptr)
	`)
	expectOutput(t, interp, "42")
}

func TestPointerModify(t *testing.T) {
	interp := run(t, `
		xuiar x = 10
		xuet ptr = &x
		x = 99
		print(*ptr)
	`)
	expectOutput(t, interp, "99")
}

func TestAlloc(t *testing.T) {
	interp := run(t, `
		xuet buf = alloc(5)
		print(len(buf))
		print(buf[0])
	`)
	expectOutput(t, interp, "5", "0")
}

func TestSizeof(t *testing.T) {
	interp := run(t, `
		print(sizeof(42))
		print(sizeof("hello"))
	`)
	expectOutput(t, interp, "8", "5")
}

// --- JSON built-in tests ---

func TestJsonParse(t *testing.T) {
	interp := run(t, `
		xuet data = json_parse("{\"name\": \"xuesos\", \"version\": 1}")
		print(data["name"])
		print(data["version"])
	`)
	expectOutput(t, interp, "xuesos", "1")
}

func TestJsonStringify(t *testing.T) {
	interp := run(t, `
		xuet m = {"a": 1, "b": 2}
		xuet s = json_stringify(m)
		print(type(s))
	`)
	expectOutput(t, interp, "str")
}

// --- Filesystem built-in tests ---

func TestFileExists(t *testing.T) {
	interp := run(t, `
		print(file_exists("interpreter.go"))
		print(file_exists("nonexistent.xyz"))
	`)
	expectOutput(t, interp, "xuitru", "xuinia")
}

func TestPathJoin(t *testing.T) {
	interp := run(t, `
		xuet p = path_join("a", "b", "c.txt")
		print(contains(p, "b"))
	`)
	expectOutput(t, interp, "xuitru")
}

// --- Interface (xuinterface) tests ---

func TestInterface(t *testing.T) {
	interp := run(t, `
		xuinterface Shape {
			area() float
		}
		xuiruct Circle {
			radius float
		}
		xuimpl Circle {
			xuen area(self) float {
				xueturn 3.14 * self.radius * self.radius
			}
		}
		xuet c = Circle { radius = 5.0 }
		print(c.area())
	`)
	expectOutput(t, interp, "78.5")
}

func TestImplementsBuiltin(t *testing.T) {
	interp := run(t, `
		xuinterface Drawable {
			draw()
		}
		xuiruct Box {
			width float
		}
		xuimpl Box {
			xuen draw(self) {
				print("drawing box")
			}
		}
		xuiruct Plain {
			x int
		}
		xuet b = Box { width = 10.0 }
		xuet p = Plain { x = 1 }
		print(implements(b, "Drawable"))
		print(implements(p, "Drawable"))
	`)
	expectOutput(t, interp, "xuitru", "xuinia")
}

func TestOperatorOverloading(t *testing.T) {
	interp := run(t, `
		xuiruct Vec2 {
			x float
			y float
		}
		xuimpl Vec2 {
			xuen __add(self, other Vec2) Vec2 {
				xueturn Vec2 { x = self.x + other.x, y = self.y + other.y }
			}
		}
		xuet a = Vec2 { x = 1.0, y = 2.0 }
		xuet b = Vec2 { x = 3.0, y = 4.0 }
		xuet c = a + b
		print(c.x)
		print(c.y)
	`)
	expectOutput(t, interp, "4", "6")
}

func TestCastInt(t *testing.T) {
	interp := run(t, `
		print(cast_int("42"))
		print(cast_int(3.14))
		print(cast_int(xuitru))
	`)
	expectOutput(t, interp, "42", "3", "1")
}

func TestCastStr(t *testing.T) {
	interp := run(t, `
		print(cast_str(42))
		print(cast_str(3.14))
	`)
	expectOutput(t, interp, "42", "3.14")
}

// --- Tuple (multiple return values) tests ---

func TestTuple(t *testing.T) {
	interp := run(t, `
		xuen divide(a int, b int) {
			xuif (b == 0) {
				xueturn tuple(0, "division by zero")
			}
			xueturn tuple(a / b, "")
		}
		xuet result = divide(10, 3)
		print(first(result))
		print(second(result))
	`)
	expectOutput(t, interp, "3", "")
}

func TestTupleUnpack(t *testing.T) {
	interp := run(t, `
		xuet t = tuple(1, "hello", xuitru)
		print(unpack(t, 0))
		print(unpack(t, 1))
		print(unpack(t, 2))
	`)
	expectOutput(t, interp, "1", "hello", "xuitru")
}

func TestTupleIsError(t *testing.T) {
	interp := run(t, `
		xuet t1 = tuple(42, "")
		xuet t2 = tuple(0, "something went wrong")
		print(is_error(second(t1)))
		print(is_error(second(t2)))
	`)
	expectOutput(t, interp, "xuinia", "xuitru")
}

// --- Defer tests ---

func TestDefer(t *testing.T) {
	interp := run(t, `
		print("start")
		xudefer print("deferred 1")
		xudefer print("deferred 2")
		print("end")
	`)
	expectOutput(t, interp, "start", "end", "deferred 2", "deferred 1")
}

// --- WaitGroup tests ---

func TestWaitGroup(t *testing.T) {
	interp := run(t, `
		xuet wg = wg_new()
		wg_add(wg, 1)
		spawn(() => {
			sleep(1)
			wg_done(wg)
		})
		wg_wait(wg)
		print("done")
	`)
	expectOutput(t, interp, "done")
}

// --- Mutex tests ---

func TestMutex(t *testing.T) {
	interp := run(t, `
		xuet mu = mutex_new()
		mutex_lock(mu)
		mutex_unlock(mu)
		print("ok")
	`)
	expectOutput(t, interp, "ok")
}

func TestTimeNow(t *testing.T) {
	interp := run(t, `
		xuet start = time_now()
		xuet elapsed = time_since(start)
		print(elapsed >= 0)
	`)
	expectOutput(t, interp, "xuitru")
}

func TestBenchmarkBuiltin(t *testing.T) {
	interp := run(t, `
		xuiar counter = 0
		xuen inc() {
			counter = counter + 1
		}
		benchmark("inc", inc, 10)
	`)
	output := interp.Output()
	if len(output) != 1 {
		t.Fatalf("expected 1 output line, got %d: %v", len(output), output)
	}
	if !strings.Contains(output[0], "benchmark inc:") {
		t.Errorf("expected benchmark output, got %q", output[0])
	}
	if !strings.Contains(output[0], "10 iterations") {
		t.Errorf("expected 10 iterations in output, got %q", output[0])
	}
}

// --- Standard library expansion tests ---

func TestRepeat(t *testing.T) {
	interp := run(t, `print(repeat("ab", 3))`)
	expectOutput(t, interp, "ababab")
}

func TestReverse(t *testing.T) {
	interp := run(t, `
		print(reverse("hello"))
		print(reverse([1, 2, 3]))
	`)
	expectOutput(t, interp, "olleh", "[3, 2, 1]")
}

func TestSortArr(t *testing.T) {
	interp := run(t, `
		print(sort_arr([3, 1, 4, 1, 5, 9]))
	`)
	expectOutput(t, interp, "[1, 1, 3, 4, 5, 9]")
}

func TestUnique(t *testing.T) {
	interp := run(t, `print(unique([1, 2, 2, 3, 3, 3]))`)
	expectOutput(t, interp, "[1, 2, 3]")
}

func TestZip(t *testing.T) {
	interp := run(t, `
		xuet pairs = zip([1, 2, 3], ["a", "b", "c"])
		print(len(pairs))
	`)
	expectOutput(t, interp, "3")
}

func TestEnumerate(t *testing.T) {
	interp := run(t, `
		xuet items = enumerate(["a", "b", "c"])
		print(len(items))
	`)
	expectOutput(t, interp, "3")
}

func TestCount(t *testing.T) {
	interp := run(t, `print(count("hello world hello", "hello"))`)
	expectOutput(t, interp, "2")
}

func TestOsPlatform(t *testing.T) {
	interp := run(t, `
		xuet p = os_platform()
		print(len(p) > 0)
	`)
	expectOutput(t, interp, "xuitru")
}

func TestMathRandom(t *testing.T) {
	interp := run(t, `
		xuet r = math_random()
		print(r >= 0.0 && r < 1.0)
	`)
	expectOutput(t, interp, "xuitru")
}

// --- Select statement ---

func TestSelectDefault(t *testing.T) {
	interp := run(t, `
		xuet ch = channel(0)
		xuselect {
			ch => { print("received") }
			_ => { print("default") }
		}
	`)
	expectOutput(t, interp, "default")
}

func TestSelectRecv(t *testing.T) {
	interp := run(t, `
		xuet ch = channel(1)
		send(ch, 42)
		xuselect {
			ch => { print(it) }
			_ => { print("default") }
		}
	`)
	expectOutput(t, interp, "42")
}

// --- Type Switch ---

func TestTypeSwitch(t *testing.T) {
	interp := run(t, `
		xuen describe(x) {
			xuiatch (type(x)) {
				"int" => print("integer")
				"str" => print("string")
				_ => print("other")
			}
		}
		describe(42)
		describe("hello")
		describe(xuitru)
	`)
	expectOutput(t, interp, "integer", "string", "other")
}

func TestTypeOfBuiltin(t *testing.T) {
	interp := run(t, `
		print(type_of(42))
		print(type_of("hello"))
		print(type_of(xuitru))
		print(type_of(xuinull))
	`)
	expectOutput(t, interp, "int", "str", "bool", "null")
}

// --- C-style For Loop ---

func TestClassicFor(t *testing.T) {
	interp := run(t, `
		xuiar sum = 0
		xuior (xuiar i = 1 : i <= 10 : i = i + 1) {
			sum = sum + i
		}
		print(sum)
	`)
	expectOutput(t, interp, "55")
}

func TestClassicForBreak(t *testing.T) {
	interp := run(t, `
		xuior (xuiar i = 0 : i < 100 : i = i + 1) {
			xuif (i == 5) { xuieak }
			print(i)
		}
	`)
	expectOutput(t, interp, "0", "1", "2", "3", "4")
}

func TestClassicForContinue(t *testing.T) {
	interp := run(t, `
		xuior (xuiar i = 0 : i < 5 : i = i + 1) {
			xuif (i == 2) { xuitinue }
			print(i)
		}
	`)
	expectOutput(t, interp, "0", "1", "3", "4")
}

func TestClassicForNested(t *testing.T) {
	interp := run(t, `
		xuiar result = 0
		xuior (xuiar i = 0 : i < 3 : i = i + 1) {
			xuior (xuiar j = 0 : j < 3 : j = j + 1) {
				result = result + 1
			}
		}
		print(result)
	`)
	expectOutput(t, interp, "9")
}

// --- Variadic Functions (format, printf) ---

func TestFormat(t *testing.T) {
	interp := run(t, `
		xuet s = format("Hello {} you are {} years old", "Alice", 30)
		print(s)
	`)
	expectOutput(t, interp, "Hello Alice you are 30 years old")
}

func TestFormatNoArgs(t *testing.T) {
	interp := run(t, `
		xuet s = format("no placeholders")
		print(s)
	`)
	expectOutput(t, interp, "no placeholders")
}

func TestPrintf(t *testing.T) {
	interp := run(t, `
		printf("Hello {}", "World")
	`)
	expectOutput(t, interp, "Hello World")
}

// --- Regex built-in tests ---

func TestRegexMatch(t *testing.T) {
	interp := run(t, `
		print(regex_match("^hello", "hello world"))
		print(regex_match("^world", "hello world"))
	`)
	expectOutput(t, interp, "xuitru", "xuinia")
}

func TestRegexFindAll(t *testing.T) {
	interp := run(t, `
		xuet matches = regex_find_all("[0-9]+", "abc123def456")
		print(len(matches))
		print(matches[0])
		print(matches[1])
	`)
	expectOutput(t, interp, "2", "123", "456")
}

func TestRegexReplace(t *testing.T) {
	interp := run(t, `
		xuet result = regex_replace("[0-9]+", "abc123def456", "NUM")
		print(result)
	`)
	expectOutput(t, interp, "abcNUMdefNUM")
}

// --- Crypto built-in tests ---

func TestSha256(t *testing.T) {
	interp := run(t, `
		xuet hash = sha256("hello")
		print(len(hash))
	`)
	expectOutput(t, interp, "64")
}

// --- Encoding built-in tests ---

func TestBase64(t *testing.T) {
	interp := run(t, `
		xuet encoded = base64_encode("Hello, World!")
		xuet decoded = base64_decode(encoded)
		print(decoded)
	`)
	expectOutput(t, interp, "Hello, World!")
}

func TestUrlEncode(t *testing.T) {
	interp := run(t, `
		xuet encoded = url_encode("hello world&foo=bar")
		print(contains(encoded, "+") || contains(encoded, "%20"))
	`)
	expectOutput(t, interp, "xuitru")
}

func TestTcpFunctionsExist(t *testing.T) {
	interp := run(t, `
		print(type(tcp_listen))
		print(type(tcp_connect))
	`)
	expectOutput(t, interp, "builtin", "builtin")
}

func TestHttpPostExists(t *testing.T) {
	interp := run(t, `print(type(http_post))`)
	expectOutput(t, interp, "builtin")
}

// --- Timer / Ticker / Sort / Error built-in tests ---

func TestSetTimeout(t *testing.T) {
	interp := run(t, `
		xuet ch = channel(1)
		set_timeout(() => { send(ch, "done") }, 10)
		xuet result = recv(ch)
		print(result)
	`)
	expectOutput(t, interp, "done")
}

func TestSortBy(t *testing.T) {
	interp := run(t, `
		xuet arr = [3, 1, 4, 1, 5]
		xuet sorted = sort_by(arr, (a, b) => a - b)
		print(sorted)
	`)
	expectOutput(t, interp, "[1, 1, 3, 4, 5]")
}

func TestSortByDesc(t *testing.T) {
	interp := run(t, `
		xuet arr = [3, 1, 4, 1, 5]
		xuet sorted = sort_by(arr, (a, b) => b - a)
		print(sorted)
	`)
	expectOutput(t, interp, "[5, 4, 3, 1, 1]")
}

func TestErrorNew(t *testing.T) {
	interp := run(t, `
		xuet err = error_new("something failed")
		print(is_err(err))
		print(err["message"])
	`)
	expectOutput(t, interp, "xuitru", "something failed")
}

func TestErrorWrap(t *testing.T) {
	interp := run(t, `
		xuet inner = error_new("file not found")
		xuet wrapped = error_wrap(inner, "config loading failed")
		print(wrapped["message"])
	`)
	expectOutput(t, interp, "config loading failed: file not found")
}

func TestTemplate(t *testing.T) {
	interp := run(t, `
		xuet lb = from_char_code(123)
		xuet rb = from_char_code(125)
		xuet tmpl = "Hello " + lb + lb + "name" + rb + rb + ", age " + lb + lb + "age" + rb + rb + "!"
		xuet result = template(tmpl, {"name": "Xuesos", "age": "1"})
		print(result)
	`)
	expectOutput(t, interp, "Hello Xuesos, age 1!")
}

func TestFields(t *testing.T) {
	interp := run(t, `
		xuiruct Point { x int
			y int }
		xuet p = Point { x = 10, y = 20 }
		xuet f = fields(p)
		print(len(f))
	`)
	expectOutput(t, interp, "2")
}

func TestGetSetField(t *testing.T) {
	interp := run(t, `
		xuiruct Box { value int }
		xuiar b = Box { value = 42 }
		print(get_field(b, "value"))
		set_field(b, "value", 99)
		print(get_field(b, "value"))
	`)
	expectOutput(t, interp, "42", "99")
}

func TestTypeName(t *testing.T) {
	interp := run(t, `
		xuiruct Dog { name str }
		xuet d = Dog { name = "Rex" }
		print(type_name(d))
		print(type_name(42))
	`)
	expectOutput(t, interp, "Dog", "int")
}

func TestIsTypeChecks(t *testing.T) {
	interp := run(t, `
		print(is_int(42))
		print(is_string("hello"))
		print(is_null(xuinull))
		print(is_array([1, 2]))
		print(is_map({"a": 1}))
	`)
	expectOutput(t, interp, "xuitru", "xuitru", "xuitru", "xuitru", "xuitru")
}

func TestCopyVal(t *testing.T) {
	interp := run(t, `
		xuiar original = [1, 2, 3]
		xuet copied = copy_val(original)
		push(original, 4)
		print(len(original))
		print(len(copied))
	`)
	expectOutput(t, interp, "4", "3")
}
