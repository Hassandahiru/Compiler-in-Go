package symtab

import (
	"fmt"
)

// ScopeKind represents the kind of scope.
//
// DESIGN CHOICE: Distinguish different scope kinds because:
// - Different scoping rules (e.g., function scope allows returns, loop scope allows break/continue)
// - Different lifetime rules (global vs local storage)
// - Better error messages ("break outside loop" vs "break outside scope")
type ScopeKind int

const (
	// ScopeGlobal is the top-level scope of a package
	ScopeGlobal ScopeKind = iota

	// ScopeFunction is a function's scope (for parameters and local variables)
	ScopeFunction

	// ScopeBlock is a block scope (like { ... })
	ScopeBlock

	// ScopeLoop is a loop scope (allows break/continue)
	ScopeLoop

	// ScopeSwitch is a switch scope (allows break)
	ScopeSwitch

	// ScopeStruct is a struct scope (for fields)
	ScopeStruct
)

// String returns a human-readable representation of the scope kind.
func (sk ScopeKind) String() string {
	switch sk {
	case ScopeGlobal:
		return "global"
	case ScopeFunction:
		return "function"
	case ScopeBlock:
		return "block"
	case ScopeLoop:
		return "loop"
	case ScopeSwitch:
		return "switch"
	case ScopeStruct:
		return "struct"
	default:
		return "unknown"
	}
}

// Scope represents a lexical scope in the program.
//
// WHAT IS A SCOPE?
// A scope is a region of code where names can be declared and resolved.
// Scopes are nested: inner scopes can see names from outer scopes.
//
// EXAMPLE:
//   var x = 1;          // global scope
//   func foo() {        // function scope (can see x)
//       var y = 2;      // can see x and y
//       if (true) {     // block scope
//           var z = 3;  // can see x, y, and z
//       }
//       // can see x and y, but NOT z
//   }
//
// DESIGN CHOICE: Use a tree structure (parent pointers) rather than a stack because:
// - Natural representation of nested scopes
// - Easy to traverse up to find enclosing function/loop
// - Supports multiple children (for if/else branches)
// - No need to explicitly push/pop
type Scope struct {
	// Kind is the kind of scope
	Kind ScopeKind

	// Parent is the enclosing scope (nil for global scope)
	Parent *Scope

	// Symbols maps names to their symbols in this scope
	// We use a map for O(1) lookup
	// DESIGN CHOICE: Don't use sync.Map because:
	// - Symbol tables are typically built in one pass (no concurrency)
	// - Regular maps are faster for our use case
	// - If we add concurrency later, we can add locks
	Symbols map[string]*Symbol

	// Children are the scopes nested inside this one
	// We track these for:
	// - Debugging (visualizing scope tree)
	// - Lifetime analysis (finding all variables in a function)
	// - Optimization (inlining, escape analysis)
	Children []*Scope

	// Function is the enclosing function (nil if not in a function)
	// This is useful for:
	// - Return type checking (does return value match function signature?)
	// - Closure analysis (which variables are captured?)
	Function *Symbol

	// Depth is the nesting depth (0 for global, 1 for top-level function, etc.)
	// Used for:
	// - Debugging and visualization
	// - Optimization heuristics (deeply nested code is less performance-critical)
	Depth int
}

// NewScope creates a new scope with the given kind and parent.
//
// USAGE:
//   global := NewScope(ScopeGlobal, nil)
//   funcScope := NewScope(ScopeFunction, global)
//   blockScope := NewScope(ScopeBlock, funcScope)
func NewScope(kind ScopeKind, parent *Scope) *Scope {
	depth := 0
	if parent != nil {
		depth = parent.Depth + 1
	}

	scope := &Scope{
		Kind:     kind,
		Parent:   parent,
		Symbols:  make(map[string]*Symbol),
		Children: make([]*Scope, 0),
		Depth:    depth,
	}

	// Link to parent
	if parent != nil {
		parent.Children = append(parent.Children, scope)
		// Inherit function from parent (unless this is a function scope)
		if kind != ScopeFunction {
			scope.Function = parent.Function
		}
	}

	return scope
}

// Define adds a symbol to this scope.
//
// RETURNS:
// - nil if successful
// - error if a symbol with the same name already exists
//
// DESIGN CHOICE: Return error rather than panic because:
// - Caller can decide how to handle (report error and continue, or stop)
// - Consistent with Go error handling philosophy
// - Allows collecting multiple errors in one pass
//
// NOTE: This does NOT check parent scopes. Shadowing is allowed:
//   var x = 1;
//   func foo() {
//       var x = 2;  // This is OK - shadows outer x
//   }
func (s *Scope) Define(symbol *Symbol) error {
	if existing, ok := s.Symbols[symbol.Name]; ok {
		return fmt.Errorf("symbol %s already declared at %s",
			symbol.Name, existing.Pos.String())
	}

	s.Symbols[symbol.Name] = symbol
	symbol.Scope = s
	symbol.Index = len(s.Symbols) - 1 // 0-based index

	return nil
}

// Lookup finds a symbol by name in this scope or any parent scope.
//
// RETURNS:
// - The symbol if found
// - nil if not found
//
// LOOKUP PROCESS:
// 1. Check this scope
// 2. If not found, check parent scope
// 3. Repeat until found or reach global scope
//
// This implements lexical scoping.
//
// DESIGN CHOICE: Mark symbols as used during lookup because:
// - Natural place to track usage
// - Avoids needing a separate pass
// - Matches how developers think ("looking up a symbol = using it")
func (s *Scope) Lookup(name string) *Symbol {
	// Check this scope first
	if symbol, ok := s.Symbols[name]; ok {
		symbol.MarkUsed()
		return symbol
	}

	// Not found - check parent scope
	if s.Parent != nil {
		return s.Parent.Lookup(name)
	}

	// Not found anywhere
	return nil
}

// LookupLocal finds a symbol by name only in this scope (not parent scopes).
//
// This is useful for:
// - Detecting redeclarations in the same scope
// - Finding parameters in a function signature
// - Looking up fields in a struct (without checking outer scopes)
func (s *Scope) LookupLocal(name string) *Symbol {
	return s.Symbols[name]
}

// IsGlobal returns true if this is the global scope.
func (s *Scope) IsGlobal() bool {
	return s.Kind == ScopeGlobal
}

// IsFunction returns true if this is a function scope.
func (s *Scope) IsFunction() bool {
	return s.Kind == ScopeFunction
}

// IsLoop returns true if this is a loop scope.
func (s *Scope) IsLoop() bool {
	return s.Kind == ScopeLoop
}

// IsSwitch returns true if this is a switch scope.
func (s *Scope) IsSwitch() bool {
	return s.Kind == ScopeSwitch
}

// FindEnclosingFunction finds the nearest enclosing function scope.
// Returns nil if not inside a function.
//
// This is useful for:
// - Return statements (check return type matches function)
// - Closure analysis (which function do we belong to?)
func (s *Scope) FindEnclosingFunction() *Scope {
	if s.IsFunction() {
		return s
	}
	if s.Parent != nil {
		return s.Parent.FindEnclosingFunction()
	}
	return nil
}

// FindEnclosingLoop finds the nearest enclosing loop scope.
// Returns nil if not inside a loop.
//
// This is useful for:
// - Break/continue statements (only valid inside loops)
// - Optimization (loop invariant code motion)
func (s *Scope) FindEnclosingLoop() *Scope {
	if s.IsLoop() {
		return s
	}
	if s.Parent != nil {
		return s.Parent.FindEnclosingLoop()
	}
	return nil
}

// FindEnclosingLoopOrSwitch finds the nearest enclosing loop or switch scope.
// Returns nil if not inside a loop or switch.
//
// This is useful for:
// - Break statements (valid in both loops and switches)
func (s *Scope) FindEnclosingLoopOrSwitch() *Scope {
	if s.IsLoop() || s.IsSwitch() {
		return s
	}
	if s.Parent != nil {
		return s.Parent.FindEnclosingLoopOrSwitch()
	}
	return nil
}

// AllSymbols returns all symbols in this scope and all parent scopes.
// The symbols are returned in order from innermost to outermost scope.
//
// This is useful for:
// - Debugging (showing all visible names)
// - IDE features (autocomplete)
// - Closure analysis (finding all captured variables)
func (s *Scope) AllSymbols() []*Symbol {
	symbols := make([]*Symbol, 0)

	// Add symbols from this scope
	for _, symbol := range s.Symbols {
		symbols = append(symbols, symbol)
	}

	// Add symbols from parent scopes
	if s.Parent != nil {
		symbols = append(symbols, s.Parent.AllSymbols()...)
	}

	return symbols
}

// LocalSymbols returns all symbols declared in this scope only.
func (s *Scope) LocalSymbols() []*Symbol {
	symbols := make([]*Symbol, 0, len(s.Symbols))
	for _, symbol := range s.Symbols {
		symbols = append(symbols, symbol)
	}
	return symbols
}

// UnusedSymbols returns all symbols in this scope that were never used.
//
// This is useful for:
// - Warning about unused variables
// - Dead code elimination
// - Code quality checks
//
// DESIGN CHOICE: Only check local scope because:
// - Variables in parent scopes might be used elsewhere
// - We only want to warn about variables we declared
func (s *Scope) UnusedSymbols() []*Symbol {
	unused := make([]*Symbol, 0)
	for _, symbol := range s.Symbols {
		if !symbol.Used {
			unused = append(unused, symbol)
		}
	}
	return unused
}

// String returns a human-readable representation of the scope.
// Shows the scope kind, depth, and number of symbols.
func (s *Scope) String() string {
	return fmt.Sprintf("%s scope (depth %d, %d symbols)",
		s.Kind.String(), s.Depth, len(s.Symbols))
}

// DebugString returns a detailed representation of the scope tree.
// This recursively prints the scope and all children, indented by depth.
//
// EXAMPLE OUTPUT:
//   global scope (2 symbols)
//     variable x: int
//     function foo: func() int
//       function scope (2 symbols)
//         parameter n: int
//         variable result: int
//         block scope (1 symbol)
//           variable temp: int
func (s *Scope) DebugString() string {
	return s.debugStringIndent(0)
}

func (s *Scope) debugStringIndent(indent int) string {
	prefix := ""
	for i := 0; i < indent; i++ {
		prefix += "  "
	}

	result := prefix + s.String() + "\n"

	// Print symbols
	for _, symbol := range s.Symbols {
		result += prefix + "  " + symbol.String() + "\n"
	}

	// Print children
	for _, child := range s.Children {
		result += child.debugStringIndent(indent + 1)
	}

	return result
}
