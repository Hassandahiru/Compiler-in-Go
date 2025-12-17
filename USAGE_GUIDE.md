# Compiler Usage Guide

## Quick Start

### 1. Build the Compiler

```bash
cd /Users/hassan/Desktop/Projects_DevFiles/go\ programs/Compiler-project

# Build the compiler
go build -o compiler ./cmd/compiler
```

This creates a binary called `compiler` in your project directory.

### 2. Run the Compiler

```bash
# Compile a program
./compiler testdata/valid/fibonacci.src
```

### 3. What You'll See

The compiler will show:
- ✅ Parsing successful
- ✅ Semantic analysis successful
- ✅ IR generation successful
- **Unoptimized IR** (the raw intermediate representation)
- ✅ Optimization successful
- **Optimized IR** (after constant folding and dead code elimination)
- Compilation summary

## Writing Your Own Programs

### Basic Syntax

Create a file called `myprogram.src`:

```go
package main

func main() {
    var x int = 10;
    var y int = 20;
    var sum int = x + y;
}
```

Compile it:
```bash
./compiler myprogram.src
```

### Language Features

#### 1. Variables

```go
var x int = 5;           // Explicitly typed
var y = 10;              // Type inferred (int)
var name string = "Alice";
var flag bool = true;
```

#### 2. Functions

```go
func add(x int, y int) int {
    return x + y;
}

func greet(name string) void {
    // void functions don't return a value
}

func main() {
    var result int = add(5, 3);
}
```

#### 3. Control Flow

**If statements:**
```go
func max(a int, b int) int {
    if (a > b) {
        return a;
    } else {
        return b;
    }
}
```

**While loops:**
```go
func countdown(n int) void {
    while (n > 0) {
        n = n - 1;
    }
}
```

**For loops:**
```go
func sum(n int) int {
    var total int = 0;
    for (var i int = 1; i <= n; i = i + 1) {
        total = total + i;
    }
    return total;
}
```

**Break and Continue:**
```go
func findFirst(n int) int {
    for (var i int = 0; i < n; i = i + 1) {
        if (i % 7 == 0) {
            return i;  // or use break/continue
        }
    }
    return -1;
}
```

#### 4. Structs

```go
struct Point {
    x int;
    y int;
}

func distance(p Point) int {
    return p.x * p.x + p.y * p.y;
}

func main() {
    var origin Point = Point{x: 0, y: 0};
    var point Point = Point{x: 3, y: 4};
    var dist int = distance(point);
}
```

#### 5. Arrays

```go
func arrayDemo() void {
    var numbers [5]int;           // Fixed-size array
    var first int = numbers[0];   // Array access
}
```

#### 6. Operators

**Arithmetic:**
```go
var sum int = 2 + 3;       // Addition
var diff int = 10 - 4;     // Subtraction
var prod int = 7 * 8;      // Multiplication
var quot int = 20 / 4;     // Division
var rem int = 17 % 5;      // Modulo
```

**Comparison:**
```go
var eq bool = 5 == 5;      // Equal
var neq bool = 5 != 3;     // Not equal
var lt bool = 3 < 5;       // Less than
var le bool = 3 <= 5;      // Less or equal
var gt bool = 5 > 3;       // Greater than
var ge bool = 5 >= 3;      // Greater or equal
```

**Logical:**
```go
var and bool = true && false;   // Logical AND
var or bool = true || false;    // Logical OR
var not bool = !true;           // Logical NOT
```

**Bitwise:**
```go
var band int = 12 & 10;    // Bitwise AND
var bor int = 12 | 10;     // Bitwise OR
var bxor int = 12 ^ 10;    // Bitwise XOR
var lshift int = 4 << 2;   // Left shift
var rshift int = 16 >> 2;  // Right shift
```

## Example Programs

### Example 1: Factorial

Create `factorial.src`:
```go
package main

func factorial(n int) int {
    if (n <= 1) {
        return 1;
    }
    return n * factorial(n - 1);
}

func main() {
    var result int = factorial(5);  // 120
}
```

Compile:
```bash
./compiler factorial.src
```

### Example 2: Fibonacci

Create `fib.src`:
```go
package main

func fibonacci(n int) int {
    if (n <= 1) {
        return n;
    }
    return fibonacci(n - 1) + fibonacci(n - 2);
}

func main() {
    var fib10 int = fibonacci(10);  // 55
}
```

### Example 3: Struct Usage

Create `geometry.src`:
```go
package main

struct Point {
    x int;
    y int;
}

struct Rectangle {
    topLeft Point;
    width int;
    height int;
}

func area(rect Rectangle) int {
    return rect.width * rect.height;
}

func main() {
    var origin Point = Point{x: 0, y: 0};
    var rect Rectangle = Rectangle{
        topLeft: origin,
        width: 10,
        height: 20
    };
    var a int = area(rect);  // 200
}
```

## Understanding the Output

### Unoptimized IR

This shows the raw intermediate representation before optimization:

```
func main() void {
entry:
  t1 = call fibonacci.-1([const(10)])
  result.0 = t1
  x.2 = const(5)
  t4 = x.2 * const(2)
  t5 = t4 + const(3)
  y.3 = t5
  return
}
```

Each line is an instruction:
- `t1`, `t2`, etc. are temporary variables
- `const(10)` is a constant value
- `call` invokes a function
- Operations are in three-address form (dest = src1 op src2)

### Optimized IR

After optimization, unused code is removed and constants are folded:

```
func main() void {
entry:
  t1 = call fibonacci.-1([const(10)])
  return
}
```

Notice how:
- Unused variables (`result`, `x`, `y`) were eliminated
- Only essential operations remain

## Testing the Compiler

### Run All Tests

```bash
# Run all unit tests
go test ./...

# Run with verbose output
go test ./... -v

# Run specific package tests
go test ./internal/lexer -v
go test ./internal/optimizer -v
```

### Test Programs Available

The project includes several test programs in `testdata/valid/`:

1. **fibonacci.src** - Recursive fibonacci
2. **structs.src** - Struct definitions
3. **constant_folding.src** - Tests constant folding optimization
4. **dead_code.src** - Tests dead code elimination
5. **combined_optimizations.src** - Tests multiple optimizations

Try them:
```bash
./compiler testdata/valid/fibonacci.src
./compiler testdata/valid/structs.src
./compiler testdata/valid/constant_folding.src
```

## Advanced Usage

### Verbose Optimization Output

To see optimization details, edit `cmd/compiler/main.go` and change:

```go
opt.SetVerbose(false) // Change to true
```

Then rebuild and run:
```bash
go build -o compiler ./cmd/compiler
./compiler your_program.src
```

### Output Only IR (No Summary)

Modify `cmd/compiler/main.go` to remove the summary sections if you only want IR output.

### Running Tests on Your Program

Create test cases for your program:

1. Write a valid program
2. Run the compiler
3. Check that all phases complete successfully
4. Verify the optimized IR makes sense

## Common Issues

### Issue: "undefined: x"

**Problem**: Using a variable before declaring it.

**Fix**:
```go
// Wrong
var y int = x + 5;  // x not declared yet

// Right
var x int = 10;
var y int = x + 5;
```

### Issue: "type mismatch"

**Problem**: Assigning incompatible types.

**Fix**:
```go
// Wrong
var x int = true;  // bool assigned to int

// Right
var x int = 10;
var flag bool = true;
```

### Issue: "break/continue outside loop"

**Problem**: Using break/continue outside a loop.

**Fix**:
```go
// Wrong
func foo() void {
    break;  // Not in a loop!
}

// Right
func foo() void {
    for (var i int = 0; i < 10; i = i + 1) {
        if (i > 5) {
            break;  // OK, inside loop
        }
    }
}
```

## Development Workflow

### Typical workflow:

```bash
# 1. Write your program
nano myprogram.src

# 2. Compile it
./compiler myprogram.src

# 3. Check for errors in the output
# If there are errors, fix them and repeat

# 4. Once it compiles, examine the IR
# Look at both unoptimized and optimized versions

# 5. Verify optimizations worked
# Check that unused code was eliminated
# Check that constants were folded
```

## What's Next?

Currently, the compiler produces optimized IR but doesn't generate executable code. To actually run your programs, you would need to:

1. **Add a Code Generator** (future enhancement)
   - Generate x86-64 assembly, or
   - Generate bytecode for a virtual machine

2. **Add an Interpreter** (alternative)
   - Interpret the IR directly
   - Useful for rapid prototyping

3. **Use LLVM Backend** (advanced)
   - Convert IR to LLVM IR
   - Use LLVM's code generator

For now, the compiler:
- ✅ Validates your code (syntax and semantics)
- ✅ Shows you the optimized IR
- ✅ Demonstrates compiler construction principles
- ✅ Serves as a learning tool

## Command Reference

```bash
# Build
go build -o compiler ./cmd/compiler

# Compile a program
./compiler <filename.src>

# Run tests
go test ./...
go test ./internal/lexer -v
go test ./internal/optimizer -v

# Run a specific test
go test ./internal/lexer -run TestLexer_Numbers

# Check test coverage
go test ./internal/lexer -cover
go test ./internal/optimizer -cover

# Format code
go fmt ./...

# Check for issues
go vet ./...
```

## Example Session

```bash
$ cd Compiler-project

$ cat > test.src << 'EOF'
package main

func add(x int, y int) int {
    return x + y;
}

func main() {
    var result int = add(5, 10);
}
EOF

$ ./compiler test.src
✓ Parsing successful
✓ Semantic analysis successful
✓ IR generation successful

=== Unoptimized IR ===

; Module: main

func add(param(x.0): int, param(y.1): int) int {
entry:
  t1 = param(x.0) + param(y.1)
  return t1
}

func main() void {
entry:
  t1 = call add.-1([const(5), const(10)])
  result.0 = t1
  return
}

✓ Optimization successful

=== Optimized IR ===

; Module: main

func add(param(x.0): int, param(y.1): int) int {
entry:
  t1 = param(x.0) + param(y.1)
  return t1
}

func main() void {
entry:
  t1 = call add.-1([const(5), const(10)])
  return
}

=== Compilation Summary ===
File: test.src
Package: main
Declarations: 2
```

## Summary

Your compiler is a **fully functional frontend** that:
- Parses source code
- Checks for errors
- Generates optimized IR
- Demonstrates professional compiler construction

Use it to:
- Learn compiler construction principles
- Experiment with language features
- See how optimizations work
- Build a foundation for adding code generation

Enjoy exploring compiler construction! 🚀
