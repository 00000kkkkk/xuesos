# Xuesos++ Programming Language

A compiled programming language with unique xui-style syntax, inspired by Rust, Go, and C.

**Speed of C. Simplicity of Go. Xui-style of Xuesos++.**

## Install

### Download binary (no dependencies)

**Linux/macOS:**
```bash
curl -L https://github.com/00000kkkkk/xusesosplusplus/releases/latest/download/xuesos-linux-amd64 -o xuesos
chmod +x xuesos
sudo mv xuesos /usr/local/bin/
```

**Windows:** download `xuesos-windows-amd64.exe` from [Releases](https://github.com/00000kkkkk/xusesosplusplus/releases) and add to PATH.

**macOS (Apple Silicon):**
```bash
curl -L https://github.com/00000kkkkk/xusesosplusplus/releases/latest/download/xuesos-darwin-arm64 -o xuesos
chmod +x xuesos
sudo mv xuesos /usr/local/bin/
```

### Build from source (requires Go 1.24+)
```bash
git clone https://github.com/00000kkkkk/xusesosplusplus.git
cd xusesosplusplus
go build -o xuesos .
```

## Quick Start

```bash
# Run a program
xuesos run examples/hello.xpp

# Compile to native binary (via C, requires gcc)
xuesos build examples/fibonacci.xpp
./fibonacci

# Interactive REPL
xuesos repl

# Format code
xuesos fmt myfile.xpp

# Show version
xuesos version
```

## Syntax

```xuesos
// Variables
xuet name = "Xuesos++"          // immutable
xuiar counter = 0                // mutable

// Functions
xuen add(a int, b int) int {
    xueturn a + b
}

// Structs
xuiruct Player {
    name str
    health int
}

xuimpl Player {
    xuen take_damage(xuiar self, dmg int) {
        self.health = self.health - dmg
    }
}

// Control flow
xuif (x > 10) {
    print("big")
} xuelse {
    print("small")
}

// Loops
xuior (i xuin 0..10) {
    print(i)
}

xuile (counter < 100) {
    counter = counter + 1
}

// Pattern matching
xuiatch (status) {
    "ok" => print("success")
    "error" => print("failure")
    _ => print("unknown")
}

// Arrays
xuet nums = [1, 2, 3, 4, 5]

// Maps
xuet scores = {"alice": 90, "bob": 85}
print(scores["alice"])

// String interpolation
xuet name = "World"
print("Hello {name}! 2+2={2+2}")

// Lambda functions
xuet add = (a, b) => a + b
xuet square = (n) => n * n

// Error handling
xutry {
    xuthrow "something went wrong"
} xucatch (e) {
    print("caught: " + e)
}

// Multi-file imports
xuimport "mylib"

// Pointers
xuiar x = 42
xuet ptr = &x
print(*ptr)        // 42

// Memory allocation
xuet buf = alloc(1024)

// Concurrency (goroutines + channels)
xuet ch = channel(10)
spawn(() => {
    send(ch, "hello from goroutine")
})
xuet msg = recv(ch)

// Higher-order functions (generics pattern)
xuet doubled = map_arr(nums, (x) => x * 2)
xuet evens = filter_arr(nums, (x) => x % 2 == 0)

// Enums
xuenum Direction {
    Up
    Down
    Left
    Right
}
```

## Keywords

| Xuesos++ | Meaning | Xuesos++ | Meaning |
|----------|---------|----------|---------|
| `xuet` | let (immutable) | `xueturn` | return |
| `xuiar` | var (mutable) | `xuieak` | break |
| `xuen` | function | `xuitinue` | continue |
| `xuif` | if | `xuiruct` | struct |
| `xuelse` | else | `xuimpl` | impl |
| `xuior` | for | `xuenum` | enum |
| `xuile` | while | `xuiatch` | match |
| `xuin` | in | `xuimport` | import |
| `xuitru` | true | `xuiub` | pub |
| `xuinia` | false | `xuinull` | null |
| `xutry` | try | `xucatch` | catch |
| `xuthrow` | throw | | |

## Types

| Type | Description | Example |
|------|-------------|---------|
| `int` | 64-bit integer | `42` |
| `float` | 64-bit float | `3.14` |
| `bool` | Boolean | `xuitru`, `xuinia` |
| `str` | UTF-8 string | `"hello"` |
| `char` | Unicode char | `'a'` |
| `[]T` | Array | `[1, 2, 3]` |
| `?T` | Nullable | `?int` |

## Built-in Functions

| Function | Description |
|----------|-------------|
| `print(args...)` | Print to stdout |
| `len(x)` | Length of string/array |
| `type(x)` | Type name as string |
| `sqrt(x)` | Square root |
| `abs(x)` | Absolute value |
| `max(a, b)` | Maximum of two numbers |
| `min(a, b)` | Minimum of two numbers |
| `append(arr, val)` | Append to array (returns new) |
| `to_int(x)` | Convert to int |
| `to_float(x)` | Convert to float |
| `to_str(x)` | Convert to string |
| `input(prompt)` | Read line from stdin |
| `exit(code)` | Exit program |
| `contains(str, sub)` | Check substring |
| `split(str, sep)` | Split string |
| `trim(str)` | Trim whitespace |
| `replace(str, old, new)` | Replace in string |
| `upper(str)` | Uppercase |
| `lower(str)` | Lowercase |
| `join(arr, sep)` | Join array to string |

## Architecture

```
Xuesos++ Source (.xpp)
        |
    [ Lexer ] --- tokenize
        |
    [ Parser ] --- build AST (Pratt parsing)
        |
    [ Type Checker ] --- validate types
        |
   /          \
[ Interpreter ]  [ C Codegen ]
  (xuesos run)   (xuesos build)
       |              |
    Execute      .c file -> gcc -> binary
```

## Examples

### Hello World
```xuesos
xuen main() {
    print("Hello from Xuesos++!")
}
```

### Fibonacci
```xuesos
xuen fibonacci(n int) int {
    xuif (n <= 1) {
        xueturn n
    }
    xueturn fibonacci(n - 1) + fibonacci(n - 2)
}

xuen main() {
    xuior (i xuin 0..20) {
        print(fibonacci(i))
    }
}
```

### FizzBuzz
```xuesos
xuen main() {
    xuior (i xuin 1..101) {
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
}
```

### Structs with Methods
```xuesos
xuiruct Vector2 {
    x float
    y float
}

xuimpl Vector2 {
    xuen magnitude(self) float {
        xueturn sqrt(self.x * self.x + self.y * self.y)
    }
}

xuen main() {
    xuet v = Vector2 { x = 3.0, y = 4.0 }
    print(v.magnitude())
}
```

## Development

```bash
# Run tests
go test ./...

# Build compiler
go build -o xuesos .

# Install
go install .
```

## License

Apache 2.0
