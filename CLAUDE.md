# Compiler Project - Development Guide

## Project Overview

This is a multi-pass compiler written in Go that translates source code through lexical analysis, parsing, semantic analysis, optimization, and code generation phases. The compiler follows Go's design principles: simplicity, clarity, composition over inheritance, and explicit error handling.

## Architecture

### Core Components

**Lexer (lexer/)** - Tokenizes source code into a stream of tokens
**Parser (parser/)** - Builds an Abstract Syntax Tree (AST) from tokens
**Semantic Analyzer (semantic/)** - Performs type checking and semantic validation
**IR Generator (ir/)** - Produces intermediate representation
**Optimizer (optimizer/)** - Performs optimization passes on IR
**Code Generator (codegen/)** - Emits target machine code or bytecode
**Symbol Table (symtab/)** - Manages scopes and symbol resolution

### Project Structure

```
.
├── cmd/
│   └── compiler/
│       └── main.go           # Entry point
├── internal/
│   ├── lexer/
│   │   ├── lexer.go          # Tokenization logic
│   │   ├── token.go          # Token definitions
│   │   └── position.go       # Source position tracking
│   ├── parser/
│   │   ├── parser.go         # Recursive descent parser
│   │   ├── ast/
│   │   │   ├── ast.go        # AST node definitions
│   │   │   ├── expr.go       # Expression nodes
│   │   │   ├── stmt.go       # Statement nodes
│   │   │   └── visitor.go    # Visitor pattern interface
│   │   └── precedence.go     # Operator precedence
│   ├── semantic/
│   │   ├── analyzer.go       # Semantic analysis
│   │   ├── typechecker.go    # Type checking logic
│   │   └── types/
│   │       └── types.go      # Type system definitions
│   ├── symtab/
│   │   ├── symbol.go         # Symbol definitions
│   │   └── scope.go          # Scope management
│   ├── ir/
│   │   ├── ir.go             # IR instruction definitions
│   │   ├── builder.go        # IR construction
│   │   └── basicblock.go     # Control flow graph
│   ├── optimizer/
│   │   ├── optimizer.go      # Optimization coordinator
│   │   ├── constant.go       # Constant folding
│   │   ├── deadcode.go       # Dead code elimination
│   │   └── inline.go         # Function inlining
│   ├── codegen/
│   │   ├── codegen.go        # Code generation interface
│   │   ├── x86/              # x86-64 backend
│   │   └── bytecode/         # Bytecode backend
│   └── errors/
│       ├── errors.go         # Error types
│       └── reporter.go       # Error reporting
├── pkg/
│   └── compiler/
│       └── compiler.go       # Public compiler API
├── testdata/
│   ├── valid/                # Valid test programs
│   └── invalid/              # Programs with errors
└── go.mod
```

## Design Principles

### 1. Explicit Error Handling

Always return errors explicitly. Never panic except for programmer errors or truly unrecoverable situations.

```go
// Good
func (p *Parser) parseExpression() (ast.Expr, error) {
    token := p.current()
    if token.Type != TokenIdentifier {
        return nil, fmt.Errorf("expected identifier, got %v at %v", 
            token.Type, token.Position)
    }
    // ...
}

// Bad - using panic for control flow
func (p *Parser) parseExpression() ast.Expr {
    if p.current().Type != TokenIdentifier {
        panic("expected identifier")
    }
    // ...
}
```

### 2. Interface Composition

Define small, focused interfaces that can be composed.

```go
// Small, focused interfaces
type TokenStream interface {
    Next() Token
    Peek() Token
    HasMore() bool
}

type ErrorReporter interface {
    Report(err error)
    HasErrors() bool
    Errors() []error
}

// Compose them where needed
type Parser struct {
    tokens TokenStream
    errors ErrorReporter
    // ...
}
```

### 3. Value Types Over Pointers

Use value types when possible. Use pointers for:
- Large structs that should be shared
- Types that need to be mutated
- Types that need a nil state
- Interface implementations that require pointer receivers

```go
// Value type for small, immutable data
type Token struct {
    Type     TokenType
    Lexeme   string
    Position Position
}

// Pointer type for mutable, shared state
type Scope struct {
    parent  *Scope
    symbols map[string]*Symbol
}

func NewScope(parent *Scope) *Scope {
    return &Scope{
        parent:  parent,
        symbols: make(map[string]*Symbol),
    }
}
```

### 4. Clear Package Boundaries

Each package should have a single, clear responsibility. Avoid circular dependencies.

```go
// lexer package exposes only what's needed
package lexer

// Public API
func New(source string) *Lexer { /* ... */ }

type Lexer struct {
    // Unexported fields
    source string
    pos    int
}

func (l *Lexer) NextToken() (Token, error) { /* ... */ }

// Internal implementation details are unexported
func (l *Lexer) skipWhitespace() { /* ... */ }
func (l *Lexer) readIdentifier() string { /* ... */ }
```

### 5. Visitor Pattern for AST Traversal

Use the visitor pattern for operations on the AST to maintain separation of concerns.

```go
type Visitor interface {
    VisitBinaryExpr(expr *BinaryExpr) error
    VisitUnaryExpr(expr *UnaryExpr) error
    VisitLiteral(expr *Literal) error
    VisitIdentifier(expr *Identifier) error
    // ... other node types
}

type Expr interface {
    Accept(v Visitor) error
}

type BinaryExpr struct {
    Left     Expr
    Operator Token
    Right    Expr
}

func (b *BinaryExpr) Accept(v Visitor) error {
    return v.VisitBinaryExpr(b)
}
```

## Implementation Guidelines

### Lexer Implementation

The lexer should:
- Read source character by character
- Maintain position information for error reporting
- Handle all token types including keywords, operators, literals, identifiers
- Report lexical errors with precise location information

```go
type Lexer struct {
    source  string
    start   int  // Start of current lexeme
    current int  // Current position
    line    int  // Current line number
    column  int  // Current column number
}

func (l *Lexer) NextToken() (Token, error) {
    l.skipWhitespace()
    l.start = l.current
    
    if l.isAtEnd() {
        return l.makeToken(TokenEOF), nil
    }
    
    ch := l.advance()
    
    switch {
    case isLetter(ch):
        return l.identifier()
    case isDigit(ch):
        return l.number()
    case ch == '"':
        return l.string()
    // ... handle operators and punctuation
    default:
        return Token{}, l.error("unexpected character")
    }
}
```

### Parser Implementation

Use recursive descent parsing for clarity and maintainability:

```go
func (p *Parser) parseExpression() (ast.Expr, error) {
    return p.parseAssignment()
}

func (p *Parser) parseAssignment() (ast.Expr, error) {
    expr, err := p.parseEquality()
    if err != nil {
        return nil, err
    }
    
    if p.match(TokenEqual) {
        equals := p.previous()
        value, err := p.parseAssignment()
        if err != nil {
            return nil, err
        }
        
        if ident, ok := expr.(*ast.Identifier); ok {
            return &ast.Assignment{
                Name:  ident,
                Value: value,
            }, nil
        }
        
        return nil, p.errorAt(equals, "invalid assignment target")
    }
    
    return expr, nil
}

func (p *Parser) parseEquality() (ast.Expr, error) {
    left, err := p.parseComparison()
    if err != nil {
        return nil, err
    }
    
    for p.match(TokenEqualEqual, TokenBangEqual) {
        op := p.previous()
        right, err := p.parseComparison()
        if err != nil {
            return nil, err
        }
        left = &ast.BinaryExpr{
            Left:     left,
            Operator: op,
            Right:    right,
        }
    }
    
    return left, nil
}
```

### Symbol Table and Scoping

Implement lexical scoping with a symbol table that tracks nested scopes:

```go
type SymbolTable struct {
    currentScope *Scope
}

func NewSymbolTable() *SymbolTable {
    return &SymbolTable{
        currentScope: NewScope(nil),
    }
}

func (st *SymbolTable) EnterScope() {
    st.currentScope = NewScope(st.currentScope)
}

func (st *SymbolTable) ExitScope() error {
    if st.currentScope.parent == nil {
        return errors.New("cannot exit global scope")
    }
    st.currentScope = st.currentScope.parent
    return nil
}

func (st *SymbolTable) Define(name string, symbol *Symbol) error {
    if st.currentScope.symbols[name] != nil {
        return fmt.Errorf("symbol %s already defined in this scope", name)
    }
    st.currentScope.symbols[name] = symbol
    return nil
}

func (st *SymbolTable) Resolve(name string) (*Symbol, error) {
    scope := st.currentScope
    for scope != nil {
        if sym := scope.symbols[name]; sym != nil {
            return sym, nil
        }
        scope = scope.parent
    }
    return nil, fmt.Errorf("undefined symbol: %s", name)
}
```

### Type System

Define a clear type system with support for primitives and user-defined types:

```go
type Type interface {
    String() string
    Equals(other Type) bool
    IsAssignableTo(other Type) bool
}

type PrimitiveType struct {
    Kind TypeKind
}

type TypeKind int

const (
    TypeInt TypeKind = iota
    TypeFloat
    TypeBool
    TypeString
    TypeVoid
)

type FunctionType struct {
    Parameters []Type
    ReturnType Type
}

type ArrayType struct {
    ElementType Type
    Size        int  // -1 for dynamic arrays
}
```

### Error Handling and Reporting

Provide clear, actionable error messages with context:

```go
type CompilerError struct {
    Position Position
    Message  string
    Phase    string  // "lexer", "parser", "semantic", etc.
}

func (e *CompilerError) Error() string {
    return fmt.Sprintf("%s:%d:%d: %s error: %s",
        e.Position.Filename,
        e.Position.Line,
        e.Position.Column,
        e.Phase,
        e.Message)
}

type ErrorReporter struct {
    errors []error
    source string  // For showing source context
}

func (er *ErrorReporter) Report(err error) {
    er.errors = append(er.errors, err)
}

func (er *ErrorReporter) PrintErrors(w io.Writer) {
    for _, err := range er.errors {
        fmt.Fprintln(w, err)
        
        // Print source context if available
        if compErr, ok := err.(*CompilerError); ok {
            er.printSourceContext(w, compErr.Position)
        }
    }
}
```

### IR Design

Design an SSA-like intermediate representation:

```go
type Instruction interface {
    String() string
    Operands() []*Value
}

type Value struct {
    ID   int
    Type Type
    Name string
}

type BasicBlock struct {
    Label        string
    Instructions []Instruction
    Successors   []*BasicBlock
    Predecessors []*BasicBlock
}

type Function struct {
    Name       string
    Parameters []*Value
    ReturnType Type
    Blocks     []*BasicBlock
    Entry      *BasicBlock
}

// Example instructions
type BinaryOp struct {
    Result *Value
    Op     Operator
    Left   *Value
    Right  *Value
}

type LoadInstruction struct {
    Result *Value
    Addr   *Value
}

type StoreInstruction struct {
    Addr  *Value
    Value *Value
}
```

### Optimization Passes

Implement optimization as separate, composable passes:

```go
type Pass interface {
    Name() string
    Run(fn *ir.Function) error
}

type Optimizer struct {
    passes []Pass
}

func NewOptimizer() *Optimizer {
    return &Optimizer{
        passes: []Pass{
            &ConstantFoldingPass{},
            &DeadCodeEliminationPass{},
            &CommonSubexpressionEliminationPass{},
            &InliningPass{threshold: 10},
        },
    }
}

func (o *Optimizer) Optimize(fn *ir.Function) error {
    for _, pass := range o.passes {
        if err := pass.Run(fn); err != nil {
            return fmt.Errorf("optimization pass %s failed: %w", 
                pass.Name(), err)
        }
    }
    return nil
}
```

## Testing Strategy

### Unit Tests

Write focused unit tests for each component:

```go
func TestLexer_Identifiers(t *testing.T) {
    tests := []struct {
        name     string
        input    string
        expected []TokenType
    }{
        {
            name:     "simple identifier",
            input:    "foo",
            expected: []TokenType{TokenIdentifier, TokenEOF},
        },
        {
            name:     "keyword",
            input:    "if",
            expected: []TokenType{TokenIf, TokenEOF},
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            l := lexer.New(tt.input)
            for _, expectedType := range tt.expected {
                token, err := l.NextToken()
                if err != nil {
                    t.Fatalf("unexpected error: %v", err)
                }
                if token.Type != expectedType {
                    t.Errorf("expected %v, got %v", expectedType, token.Type)
                }
            }
        })
    }
}
```

### Integration Tests

Test the entire compilation pipeline with sample programs in testdata/:

```go
func TestCompiler_ValidPrograms(t *testing.T) {
    files, err := filepath.Glob("../../testdata/valid/*.src")
    if err != nil {
        t.Fatal(err)
    }
    
    for _, file := range files {
        t.Run(filepath.Base(file), func(t *testing.T) {
            source, err := os.ReadFile(file)
            if err != nil {
                t.Fatal(err)
            }
            
            comp := compiler.New()
            _, err = comp.Compile(string(source))
            if err != nil {
                t.Errorf("compilation failed: %v", err)
            }
        })
    }
}
```

## Performance Considerations

### Memory Allocation

Minimize allocations in hot paths:

```go
// Reuse buffers for string building
type Parser struct {
    tokens TokenStream
    buf    strings.Builder  // Reused buffer
}

// Use sync.Pool for frequently allocated objects
var valuePool = sync.Pool{
    New: func() interface{} {
        return &Value{}
    },
}

func newValue() *Value {
    return valuePool.Get().(*Value)
}

func releaseValue(v *Value) {
    *v = Value{}  // Clear
    valuePool.Put(v)
}
```

### Profiling

Use Go's built-in profiling tools:

```bash
# CPU profiling
go test -cpuprofile=cpu.prof -bench=.
go tool pprof cpu.prof

# Memory profiling
go test -memprofile=mem.prof -bench=.
go tool pprof mem.prof
```

## Common Patterns

### Error Recovery in Parser

Implement panic-mode error recovery to continue parsing after errors:

```go
func (p *Parser) synchronize() {
    p.advance()
    
    for !p.isAtEnd() {
        if p.previous().Type == TokenSemicolon {
            return
        }
        
        switch p.current().Type {
        case TokenClass, TokenFunc, TokenVar, TokenFor, 
             TokenIf, TokenWhile, TokenReturn:
            return
        }
        
        p.advance()
    }
}

func (p *Parser) parseStatement() (ast.Stmt, error) {
    stmt, err := p.tryParseStatement()
    if err != nil {
        p.errors.Report(err)
        p.synchronize()
        return nil, err
    }
    return stmt, nil
}
```

### AST Cloning for Transformations

When transforming the AST, clone nodes to avoid mutation issues:

```go
func CloneExpr(expr ast.Expr) ast.Expr {
    switch e := expr.(type) {
    case *ast.BinaryExpr:
        return &ast.BinaryExpr{
            Left:     CloneExpr(e.Left),
            Operator: e.Operator,
            Right:    CloneExpr(e.Right),
        }
    case *ast.Literal:
        return &ast.Literal{
            Value: e.Value,
            Type:  e.Type,
        }
    // ... other cases
    default:
        panic(fmt.Sprintf("unknown expression type: %T", expr))
    }
}
```

## Build and Run

```bash
# Build the compiler
go build -o compiler ./cmd/compiler

# Run the compiler
./compiler input.src -o output

# Run tests
go test ./...

# Run with race detector
go test -race ./...

# Benchmarks
go test -bench=. ./internal/lexer
go test -bench=. ./internal/parser
```

## Extending the Compiler

### Adding a New Statement Type

1. Define the AST node in `internal/parser/ast/stmt.go`
2. Add visitor method to `Visitor` interface
3. Implement parsing in `internal/parser/parser.go`
4. Add semantic analysis in `internal/semantic/analyzer.go`
5. Generate IR in `internal/ir/builder.go`
6. Update code generator in `internal/codegen/`

### Adding a New Optimization Pass

1. Create new file in `internal/optimizer/`
2. Implement the `Pass` interface
3. Add to optimizer's pass list
4. Write tests verifying correctness and improvement

## References

- Go design principles: https://go.dev/doc/effective_go
- Compiler construction: "Engineering a Compiler" by Cooper & Torczon
- SSA form: "Modern Compiler Implementation" by Appel
- Go performance: https://go.dev/doc/gc-guide
