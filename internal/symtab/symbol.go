// Package symtab implements symbol table management for name resolution and scoping.
//
// DESIGN PHILOSOPHY:
// The symbol table tracks all named entities (variables, functions, types, etc.) and
// their scopes. It's used by the semantic analyzer to:
// 1. Resolve names to their declarations
// 2. Detect redeclarations and undefined names
// 3. Check that names are used in the correct context
// 4. Support nested scopes (blocks, functions, etc.)
//
// KEY DESIGN CHOICES:
// - Lexical scoping (like C, Java, Go) - inner scopes can shadow outer scopes
// - Separate namespaces for types vs values (like Go) - type Foo and var Foo can coexist
// - Symbols are immutable once created (simplifies concurrent access if needed)
package symtab

import (
	"github.com/hassan/compiler/internal/lexer"
	"github.com/hassan/compiler/internal/semantic/types"
)

// SymbolKind represents the kind of symbol.
//
// DESIGN CHOICE: Use an enum rather than using the type system (interfaces) because:
// - Simple and efficient
// - Easy to switch on
// - Clear in error messages ("expected variable, got function")
type SymbolKind int

const (
	// SymbolVariable represents a variable (var x int)
	SymbolVariable SymbolKind = iota

	// SymbolFunction represents a function (func foo() {})
	SymbolFunction

	// SymbolParameter represents a function parameter
	// We distinguish this from variables because:
	// - Parameters are read-only in some languages
	// - Different calling conventions
	// - Clearer error messages
	SymbolParameter

	// SymbolType represents a type name (struct Point {}, type MyInt = int)
	SymbolType

	// SymbolStruct represents a struct type specifically
	// We track this separately because structs have fields we need to look up
	SymbolStruct

	// SymbolField represents a struct field
	SymbolField

	// SymbolPackage represents an imported package
	SymbolPackage
)

// String returns a human-readable representation of the symbol kind.
func (sk SymbolKind) String() string {
	switch sk {
	case SymbolVariable:
		return "variable"
	case SymbolFunction:
		return "function"
	case SymbolParameter:
		return "parameter"
	case SymbolType:
		return "type"
	case SymbolStruct:
		return "struct"
	case SymbolField:
		return "field"
	case SymbolPackage:
		return "package"
	default:
		return "unknown"
	}
}

// Symbol represents a named entity in the program.
//
// DESIGN CHOICE: Store all symbol information in one struct rather than having
// separate structs for each kind because:
// - Simpler code (no type assertions)
// - All symbols have similar information
// - Easy to add new fields that apply to all symbols
//
// The downside is some fields are unused for some symbol kinds, but the memory
// overhead is minimal and the simplicity is worth it.
type Symbol struct {
	// Name is the symbol's identifier
	Name string

	// Kind is what kind of symbol this is
	Kind SymbolKind

	// Type is the symbol's type (variable type, function signature, etc.)
	Type types.Type

	// Pos is where this symbol was declared
	// This is crucial for error messages ("x already declared at line 10")
	Pos lexer.Position

	// Scope is the scope where this symbol was declared
	// We store this for:
	// - Determining symbol visibility
	// - Finding enclosing function (for return type checking)
	// - Lifetime analysis (stack vs heap allocation)
	Scope *Scope

	// Constant indicates if this is a constant (const x = 5)
	// Constants can't be reassigned and may be optimized differently
	Constant bool

	// Used tracks if this symbol has been referenced
	// This is useful for:
	// - Warning about unused variables
	// - Dead code elimination
	// - Import optimization (removing unused imports)
	Used bool

	// Value stores the constant value for compile-time constants
	// Only meaningful when Constant is true
	// Used by the optimizer for constant folding
	Value interface{}

	// Fields stores struct fields (only for SymbolStruct)
	// We store this here rather than in the Type because:
	// - Simpler lookups (no need to cast Type to StructType)
	// - Symbol table is the natural place for name -> symbol mappings
	Fields map[string]*Symbol

	// Index is the index of this symbol in its scope
	// Used for:
	// - Stack frame offsets (local variables)
	// - Parameter positions in function calls
	// - Field offsets in structs
	Index int
}

// String returns a human-readable representation of the symbol.
// Format: "kind name: type at position"
// Example: "variable x: int at main.go:42:15"
func (s *Symbol) String() string {
	return s.Kind.String() + " " + s.Name + ": " + s.Type.String() + " at " + s.Pos.String()
}

// IsGlobal returns true if this symbol is declared at global scope.
// Global symbols have different:
// - Lifetime (exist for entire program execution)
// - Visibility (may be exported to other packages)
// - Storage (in data segment, not on stack)
func (s *Symbol) IsGlobal() bool {
	return s.Scope != nil && s.Scope.IsGlobal()
}

// IsLocal returns true if this symbol is declared in a local scope.
// Local symbols are:
// - Allocated on the stack (or promoted to heap if escaped)
// - Only visible within their scope
// - Cleaned up when scope exits
func (s *Symbol) IsLocal() bool {
	return !s.IsGlobal()
}

// CanAssign returns true if this symbol can be assigned to.
//
// RULES:
// - Constants cannot be assigned
// - Functions cannot be assigned (in our language, though some allow it)
// - Types cannot be assigned
// - Variables and parameters can be assigned
//
// This is used by the semantic analyzer to check assignments like "x = 5"
func (s *Symbol) CanAssign() bool {
	if s.Constant {
		return false
	}

	switch s.Kind {
	case SymbolVariable, SymbolParameter:
		return true
	default:
		return false
	}
}

// MarkUsed marks this symbol as used.
// This is called when the symbol is referenced (read or written).
func (s *Symbol) MarkUsed() {
	s.Used = true
}

// LookupField looks up a field in a struct symbol.
// Returns nil if this is not a struct or the field doesn't exist.
//
// USAGE: When analyzing "point.x", we:
// 1. Resolve "point" to a symbol
// 2. Get its type (should be a struct)
// 3. Call LookupField("x") to find the x field
func (s *Symbol) LookupField(name string) *Symbol {
	if s.Kind != SymbolStruct {
		return nil
	}
	return s.Fields[name]
}
