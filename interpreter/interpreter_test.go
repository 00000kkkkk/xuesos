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
