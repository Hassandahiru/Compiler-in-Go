package symtab

import (
	"testing"

	"github.com/hassan/compiler/internal/lexer"
	"github.com/hassan/compiler/internal/semantic/types"
)

// Test Symbol

func TestSymbol_String(t *testing.T) {
	symbol := &Symbol{
		Name: "x",
		Kind: SymbolVariable,
		Type: types.Int,
		Pos:  lexer.Position{Filename: "test.go", Line: 1, Column: 5},
	}

	expected := "variable x: int at test.go:1:5"
	result := symbol.String()
	if result != expected {
		t.Errorf("Symbol.String() = %q, want %q", result, expected)
	}
}

func TestSymbol_IsGlobal(t *testing.T) {
	globalScope := NewScope(ScopeGlobal, nil)
	localScope := NewScope(ScopeBlock, globalScope)

	globalSymbol := &Symbol{
		Name:  "x",
		Scope: globalScope,
	}

	localSymbol := &Symbol{
		Name:  "y",
		Scope: localScope,
	}

	if !globalSymbol.IsGlobal() {
		t.Error("Expected globalSymbol.IsGlobal() to be true")
	}

	if localSymbol.IsGlobal() {
		t.Error("Expected localSymbol.IsGlobal() to be false")
	}
}

func TestSymbol_CanAssign(t *testing.T) {
	tests := []struct {
		name     string
		symbol   *Symbol
		expected bool
	}{
		{
			name: "variable can be assigned",
			symbol: &Symbol{
				Kind:     SymbolVariable,
				Constant: false,
			},
			expected: true,
		},
		{
			name: "parameter can be assigned",
			symbol: &Symbol{
				Kind:     SymbolParameter,
				Constant: false,
			},
			expected: true,
		},
		{
			name: "constant cannot be assigned",
			symbol: &Symbol{
				Kind:     SymbolVariable,
				Constant: true,
			},
			expected: false,
		},
		{
			name: "function cannot be assigned",
			symbol: &Symbol{
				Kind:     SymbolFunction,
				Constant: false,
			},
			expected: false,
		},
		{
			name: "type cannot be assigned",
			symbol: &Symbol{
				Kind:     SymbolType,
				Constant: false,
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.symbol.CanAssign()
			if result != tt.expected {
				t.Errorf("Symbol.CanAssign() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestSymbol_LookupField(t *testing.T) {
	// Create a struct symbol with fields
	structSymbol := &Symbol{
		Kind: SymbolStruct,
		Fields: map[string]*Symbol{
			"x": {Name: "x", Type: types.Int},
			"y": {Name: "y", Type: types.Int},
		},
	}

	// Test looking up existing field
	field := structSymbol.LookupField("x")
	if field == nil {
		t.Error("Expected to find field 'x'")
	} else if field.Name != "x" {
		t.Errorf("Found field with name %q, want 'x'", field.Name)
	}

	// Test looking up non-existent field
	field = structSymbol.LookupField("z")
	if field != nil {
		t.Error("Expected nil for non-existent field 'z'")
	}

	// Test looking up field on non-struct
	varSymbol := &Symbol{Kind: SymbolVariable}
	field = varSymbol.LookupField("x")
	if field != nil {
		t.Error("Expected nil for field lookup on non-struct")
	}
}

// Test Scope

func TestNewScope(t *testing.T) {
	parent := NewScope(ScopeGlobal, nil)
	child := NewScope(ScopeBlock, parent)

	if child.Parent != parent {
		t.Error("Expected child scope to have correct parent")
	}

	if child.Depth != 1 {
		t.Errorf("Expected child depth = 1, got %d", child.Depth)
	}

	if len(parent.Children) != 1 || parent.Children[0] != child {
		t.Error("Expected parent to contain child in Children slice")
	}
}

func TestScope_Define(t *testing.T) {
	scope := NewScope(ScopeGlobal, nil)
	symbol := &Symbol{
		Name: "x",
		Type: types.Int,
	}

	// First definition should succeed
	err := scope.Define(symbol)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if symbol.Scope != scope {
		t.Error("Expected symbol scope to be set")
	}

	// Duplicate definition should fail
	duplicate := &Symbol{
		Name: "x",
		Type: types.Float,
	}
	err = scope.Define(duplicate)
	if err == nil {
		t.Error("Expected error for duplicate definition")
	}
}

func TestScope_Lookup(t *testing.T) {
	global := NewScope(ScopeGlobal, nil)
	local := NewScope(ScopeBlock, global)

	globalSymbol := &Symbol{Name: "x", Type: types.Int}
	localSymbol := &Symbol{Name: "y", Type: types.Float}

	global.Define(globalSymbol)
	local.Define(localSymbol)

	// Look up local symbol in local scope
	found := local.Lookup("y")
	if found == nil {
		t.Error("Expected to find local symbol 'y'")
	} else if found.Name != "y" {
		t.Errorf("Found symbol with name %q, want 'y'", found.Name)
	}

	// Look up global symbol from local scope
	found = local.Lookup("x")
	if found == nil {
		t.Error("Expected to find global symbol 'x' from local scope")
	} else if found.Name != "x" {
		t.Errorf("Found symbol with name %q, want 'x'", found.Name)
	}

	// Look up non-existent symbol
	found = local.Lookup("z")
	if found != nil {
		t.Error("Expected nil for non-existent symbol 'z'")
	}

	// Verify symbols are marked as used
	if !globalSymbol.Used {
		t.Error("Expected global symbol to be marked as used")
	}
	if !localSymbol.Used {
		t.Error("Expected local symbol to be marked as used")
	}
}

func TestScope_LookupLocal(t *testing.T) {
	global := NewScope(ScopeGlobal, nil)
	local := NewScope(ScopeBlock, global)

	globalSymbol := &Symbol{Name: "x", Type: types.Int}
	localSymbol := &Symbol{Name: "y", Type: types.Float}

	global.Define(globalSymbol)
	local.Define(localSymbol)

	// Look up local symbol
	found := local.LookupLocal("y")
	if found == nil {
		t.Error("Expected to find local symbol 'y'")
	}

	// Should NOT find global symbol
	found = local.LookupLocal("x")
	if found != nil {
		t.Error("Expected nil when looking up parent symbol with LookupLocal")
	}
}

func TestScope_FindEnclosingFunction(t *testing.T) {
	global := NewScope(ScopeGlobal, nil)
	funcScope := NewScope(ScopeFunction, global)
	blockScope := NewScope(ScopeBlock, funcScope)

	// Block scope should find enclosing function
	found := blockScope.FindEnclosingFunction()
	if found != funcScope {
		t.Error("Expected to find function scope from block scope")
	}

	// Global scope should not find function
	found = global.FindEnclosingFunction()
	if found != nil {
		t.Error("Expected nil for enclosing function from global scope")
	}
}

func TestScope_FindEnclosingLoop(t *testing.T) {
	funcScope := NewScope(ScopeFunction, nil)
	loopScope := NewScope(ScopeLoop, funcScope)
	blockScope := NewScope(ScopeBlock, loopScope)

	// Block inside loop should find loop
	found := blockScope.FindEnclosingLoop()
	if found != loopScope {
		t.Error("Expected to find loop scope from block scope")
	}

	// Function scope should not find loop
	found = funcScope.FindEnclosingLoop()
	if found != nil {
		t.Error("Expected nil for enclosing loop from function scope")
	}
}

func TestScope_UnusedSymbols(t *testing.T) {
	scope := NewScope(ScopeGlobal, nil)

	usedSymbol := &Symbol{Name: "x", Type: types.Int, Used: true}
	unusedSymbol := &Symbol{Name: "y", Type: types.Float, Used: false}

	scope.Define(usedSymbol)
	scope.Define(unusedSymbol)

	unused := scope.UnusedSymbols()
	if len(unused) != 1 {
		t.Errorf("Expected 1 unused symbol, got %d", len(unused))
	}

	if unused[0].Name != "y" {
		t.Errorf("Expected unused symbol 'y', got %q", unused[0].Name)
	}
}

func TestSymbolKind_String(t *testing.T) {
	tests := []struct {
		kind     SymbolKind
		expected string
	}{
		{SymbolVariable, "variable"},
		{SymbolFunction, "function"},
		{SymbolParameter, "parameter"},
		{SymbolType, "type"},
		{SymbolStruct, "struct"},
		{SymbolField, "field"},
		{SymbolPackage, "package"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := tt.kind.String()
			if result != tt.expected {
				t.Errorf("SymbolKind.String() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestScopeKind_String(t *testing.T) {
	tests := []struct {
		kind     ScopeKind
		expected string
	}{
		{ScopeGlobal, "global"},
		{ScopeFunction, "function"},
		{ScopeBlock, "block"},
		{ScopeLoop, "loop"},
		{ScopeSwitch, "switch"},
		{ScopeStruct, "struct"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := tt.kind.String()
			if result != tt.expected {
				t.Errorf("ScopeKind.String() = %q, want %q", result, tt.expected)
			}
		})
	}
}
