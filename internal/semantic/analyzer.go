// Package semantic implements semantic analysis for the compiler.
//
// SEMANTIC ANALYSIS:
// After parsing, we have a syntactically correct AST, but it might not be semantically valid.
// Semantic analysis checks:
// 1. Name resolution - are all names defined before use?
// 2. Type checking - do operations use compatible types?
// 3. Control flow - are break/continue/return used correctly?
// 4. Definite assignment - are variables initialized before use?
//
// DESIGN PHILOSOPHY:
// - Collect all errors, don't stop at the first one
// - Use the visitor pattern to traverse the AST
// - Build symbol table while checking
// - Annotate AST with type information (stored separately)
//
// PASSES:
// We do semantic analysis in one pass (unlike some compilers that use multiple passes).
// This is possible because:
// - We require forward declarations (or process in order)
// - No complex type inference
// - Simpler implementation
package semantic

import (
	"fmt"

	"github.com/hassan/compiler/internal/lexer"
	"github.com/hassan/compiler/internal/parser/ast"
	"github.com/hassan/compiler/internal/semantic/types"
	"github.com/hassan/compiler/internal/symtab"
)

// Analyzer performs semantic analysis on an AST.
//
// DESIGN CHOICE: Implement the visitor pattern to traverse the AST because:
// - Separation of concerns (AST structure vs analysis)
// - Can be reused for other analyses
// - Standard pattern in compilers
type Analyzer struct {
	// currentScope tracks the current scope during traversal
	currentScope *symtab.Scope

	// globalScope is the top-level scope
	globalScope *symtab.Scope

	// errors accumulates all semantic errors
	errors []error

	// exprTypes maps expressions to their computed types
	// We store this separately rather than modifying the AST because:
	// - AST is immutable (good for concurrent access)
	// - Can run analysis multiple times
	// - Cleaner separation of concerns
	exprTypes map[ast.Expr]types.Type

	// currentFunction tracks the function we're currently analyzing
	// Used for:
	// - Checking return types
	// - Determining if we're in a function (for return statements)
	currentFunction *symtab.Symbol
}

// New creates a new semantic analyzer.
func New() *Analyzer {
	globalScope := symtab.NewScope(symtab.ScopeGlobal, nil)
	return &Analyzer{
		currentScope: globalScope,
		globalScope:  globalScope,
		errors:       make([]error, 0),
		exprTypes:    make(map[ast.Expr]types.Type),
	}
}

// Analyze performs semantic analysis on a file.
// Returns the list of errors found (empty if no errors).
func (a *Analyzer) Analyze(file *ast.File) []error {
	// Reset state
	a.errors = make([]error, 0)
	a.exprTypes = make(map[ast.Expr]types.Type)
	a.currentScope = a.globalScope

	// Process package declaration
	if file.Package == nil {
		a.error(lexer.Position{}, "missing package declaration")
		return a.errors
	}

	// Process imports
	for _, imp := range file.Imports {
		a.processImport(imp)
	}

	// Process declarations
	// We do this in two passes:
	// 1. Declare all names (to allow forward references)
	// 2. Check all bodies
	for _, decl := range file.Decls {
		a.declareDecl(decl)
	}

	for _, decl := range file.Decls {
		_ = decl.Accept(a)
	}

	return a.errors
}

// processImport processes an import declaration
func (a *Analyzer) processImport(imp *ast.ImportDecl) {
	name := imp.Path.Value.(string)
	if imp.Name != nil {
		name = imp.Name.Name
	}

	symbol := &symtab.Symbol{
		Name: name,
		Kind: symtab.SymbolPackage,
		Type: types.Invalid, // Packages don't have a type
		Pos:  imp.Pos(),
	}

	if err := a.currentScope.Define(symbol); err != nil {
		a.error(imp.Pos(), err.Error())
	}
}

// declareDecl declares a top-level declaration without checking its body
func (a *Analyzer) declareDecl(decl ast.Decl) {
	switch d := decl.(type) {
	case *ast.VarDecl:
		// Declare variables
		for _, name := range d.Names {
			// Type will be determined later
			symbol := &symtab.Symbol{
				Name:     name.Name,
				Kind:     symtab.SymbolVariable,
				Type:     types.Invalid, // Will be set during checking
				Pos:      name.Pos(),
				Constant: false,
			}
			if err := a.currentScope.Define(symbol); err != nil {
				a.error(name.Pos(), err.Error())
			}
		}

	case *ast.FuncDecl:
		// Declare function
		symbol := &symtab.Symbol{
			Name: d.Name.Name,
			Kind: symtab.SymbolFunction,
			Type: types.Invalid, // Will be set during checking
			Pos:  d.Pos(),
		}
		if err := a.currentScope.Define(symbol); err != nil {
			a.error(d.Name.Pos(), err.Error())
		}

	case *ast.StructDecl:
		// Declare struct type
		symbol := &symtab.Symbol{
			Name:   d.Name.Name,
			Kind:   symtab.SymbolStruct,
			Type:   types.Invalid, // Will be set during checking
			Pos:    d.Pos(),
			Fields: make(map[string]*symtab.Symbol),
		}
		if err := a.currentScope.Define(symbol); err != nil {
			a.error(d.Name.Pos(), err.Error())
		}

	case *ast.TypeDecl:
		// Declare type alias
		symbol := &symtab.Symbol{
			Name: d.Name.Name,
			Kind: symtab.SymbolType,
			Type: types.Invalid, // Will be set during checking
			Pos:  d.Pos(),
		}
		if err := a.currentScope.Define(symbol); err != nil {
			a.error(d.Name.Pos(), err.Error())
		}
	}
}

// Visitor implementation for declarations

func (a *Analyzer) VisitVarDecl(decl *ast.VarDecl) error {
	// Determine the type
	var varType types.Type
	var initType types.Type

	// Evaluate initializer if present
	if decl.Initializer != nil {
		result, _ := decl.Initializer.Accept(a)
		initType = result.(types.Type)
	}

	if decl.Type != nil {
		// Explicit type
		varType = a.resolveType(decl.Type)

		// Check initializer type matches declared type (if both present)
		if decl.Initializer != nil {
			if !a.assignable(initType, varType, decl.Initializer.Pos()) {
				// Error already reported by assignable
			}
		}
	} else if decl.Initializer != nil {
		// Infer from initializer
		varType = initType
	} else {
		a.error(decl.Pos(), "variable declaration must have type or initializer")
		varType = types.Invalid
	}

	// Declare or update symbols
	for _, name := range decl.Names {
		symbol := a.currentScope.LookupLocal(name.Name)
		if symbol != nil {
			// Update existing symbol (global scope)
			symbol.Type = varType
		} else {
			// Declare new symbol (local scope)
			symbol = &symtab.Symbol{
				Name:     name.Name,
				Kind:     symtab.SymbolVariable,
				Type:     varType,
				Pos:      name.Pos(),
				Constant: false,
			}
			if err := a.currentScope.Define(symbol); err != nil {
				a.error(name.Pos(), err.Error())
			}
		}
	}

	return nil
}

func (a *Analyzer) VisitFuncDecl(decl *ast.FuncDecl) error {
	// Build parameter types
	paramTypes := make([]types.Type, len(decl.Params))
	for i, param := range decl.Params {
		paramTypes[i] = a.resolveType(param.Type)
	}

	// Determine return type
	var returnType types.Type
	if decl.ReturnType != nil {
		returnType = a.resolveType(decl.ReturnType)
	} else {
		returnType = types.Void
	}

	// Create function type
	funcType := types.NewFunction(paramTypes, returnType)

	// Update the function symbol
	symbol := a.globalScope.LookupLocal(decl.Name.Name)
	if symbol != nil {
		symbol.Type = funcType
	}

	// Create function scope
	a.enterScope(symtab.ScopeFunction)
	a.currentScope.Function = symbol
	a.currentFunction = symbol

	// Add parameters to scope
	for i, param := range decl.Params {
		paramSymbol := &symtab.Symbol{
			Name:  param.Name.Name,
			Kind:  symtab.SymbolParameter,
			Type:  paramTypes[i],
			Pos:   param.Pos(),
			Index: i,
		}
		if err := a.currentScope.Define(paramSymbol); err != nil {
			a.error(param.Pos(), err.Error())
		}
	}

	// Check function body
	if decl.Body != nil {
		_ = decl.Body.Accept(a)
	}

	a.exitScope()
	a.currentFunction = nil

	return nil
}

func (a *Analyzer) VisitStructDecl(decl *ast.StructDecl) error {
	// Build struct fields
	structFields := make([]types.StructField, len(decl.Fields))
	fieldSymbols := make(map[string]*symtab.Symbol)

	for i, field := range decl.Fields {
		fieldType := a.resolveType(field.Type)
		structFields[i] = types.StructField{
			Name: field.Name.Name,
			Type: fieldType,
		}

		// Create field symbol
		fieldSymbol := &symtab.Symbol{
			Name:  field.Name.Name,
			Kind:  symtab.SymbolField,
			Type:  fieldType,
			Pos:   field.Pos(),
			Index: i,
		}
		fieldSymbols[field.Name.Name] = fieldSymbol
	}

	// Create struct type
	structType := types.NewStruct(decl.Name.Name, structFields)

	// Update the struct symbol
	symbol := a.globalScope.LookupLocal(decl.Name.Name)
	if symbol != nil {
		symbol.Type = structType
		symbol.Fields = fieldSymbols
	}

	return nil
}

func (a *Analyzer) VisitTypeDecl(decl *ast.TypeDecl) error {
	// Resolve the aliased type
	aliasedType := a.resolveType(decl.Type)

	// Update the type symbol
	symbol := a.globalScope.LookupLocal(decl.Name.Name)
	if symbol != nil {
		symbol.Type = aliasedType
	}

	return nil
}

// Visitor implementation for statements

func (a *Analyzer) VisitExprStmt(stmt *ast.ExprStmt) error {
	_, err := stmt.Expression.Accept(a)
	return err
}

func (a *Analyzer) VisitBlockStmt(stmt *ast.BlockStmt) error {
	a.enterScope(symtab.ScopeBlock)
	for _, s := range stmt.Statements {
		_ = s.Accept(a)
	}
	a.exitScope()
	return nil
}

func (a *Analyzer) VisitIfStmt(stmt *ast.IfStmt) error {
	// Check condition
	condType, _ := stmt.Condition.Accept(a)
	if !types.IsBooleanType(condType.(types.Type)) {
		a.error(stmt.Condition.Pos(), "condition must be boolean")
	}

	// Check branches
	_ = stmt.ThenBranch.Accept(a)
	if stmt.ElseBranch != nil {
		_ = stmt.ElseBranch.Accept(a)
	}

	return nil
}

func (a *Analyzer) VisitWhileStmt(stmt *ast.WhileStmt) error {
	// Check condition
	condType, _ := stmt.Condition.Accept(a)
	if !types.IsBooleanType(condType.(types.Type)) {
		a.error(stmt.Condition.Pos(), "condition must be boolean")
	}

	// Check body
	a.enterScope(symtab.ScopeLoop)
	_ = stmt.Body.Accept(a)
	a.exitScope()

	return nil
}

func (a *Analyzer) VisitForStmt(stmt *ast.ForStmt) error {
	a.enterScope(symtab.ScopeLoop)

	// Check init
	if stmt.Init != nil {
		_ = stmt.Init.Accept(a)
	}

	// Check condition
	if stmt.Condition != nil {
		condType, _ := stmt.Condition.Accept(a)
		if !types.IsBooleanType(condType.(types.Type)) {
			a.error(stmt.Condition.Pos(), "condition must be boolean")
		}
	}

	// Check post
	if stmt.Post != nil {
		_ = stmt.Post.Accept(a)
	}

	// Check body
	_ = stmt.Body.Accept(a)

	a.exitScope()
	return nil
}

func (a *Analyzer) VisitReturnStmt(stmt *ast.ReturnStmt) error {
	// Check if we're in a function
	if a.currentFunction == nil {
		a.error(stmt.Pos(), "return outside function")
		return nil
	}

	// Get expected return type
	funcType := a.currentFunction.Type.(*types.FunctionType)
	expectedType := funcType.ReturnType

	// Check return value
	if stmt.Value != nil {
		returnType, _ := stmt.Value.Accept(a)
		if !a.assignable(returnType.(types.Type), expectedType, stmt.Value.Pos()) {
			// Error already reported
		}
	} else {
		// Void return
		if !expectedType.Equals(types.Void) {
			a.error(stmt.Pos(), fmt.Sprintf("expected return value of type %s", expectedType))
		}
	}

	return nil
}

func (a *Analyzer) VisitBreakStmt(stmt *ast.BreakStmt) error {
	if a.currentScope.FindEnclosingLoopOrSwitch() == nil {
		a.error(stmt.Pos(), "break outside loop or switch")
	}
	return nil
}

func (a *Analyzer) VisitContinueStmt(stmt *ast.ContinueStmt) error {
	if a.currentScope.FindEnclosingLoop() == nil {
		a.error(stmt.Pos(), "continue outside loop")
	}
	return nil
}

func (a *Analyzer) VisitSwitchStmt(stmt *ast.SwitchStmt) error {
	// Check value
	valueType, _ := stmt.Value.Accept(a)

	a.enterScope(symtab.ScopeSwitch)

	// Check cases
	for _, c := range stmt.Cases {
		if !c.IsDefault {
			for _, val := range c.Values {
				caseType, _ := val.Accept(a)
				if !a.assignable(caseType.(types.Type), valueType.(types.Type), val.Pos()) {
					// Error already reported
				}
			}
		}

		// Check body
		for _, s := range c.Body {
			_ = s.Accept(a)
		}
	}

	a.exitScope()
	return nil
}

// Visitor implementation for expressions (continued in next part...)

// Helper functions

// enterScope creates a new scope
func (a *Analyzer) enterScope(kind symtab.ScopeKind) {
	a.currentScope = symtab.NewScope(kind, a.currentScope)
}

// exitScope returns to the parent scope
func (a *Analyzer) exitScope() {
	if a.currentScope.Parent != nil {
		a.currentScope = a.currentScope.Parent
	}
}

// error records a semantic error
func (a *Analyzer) error(pos lexer.Position, message string) {
	if pos.IsValid() {
		a.errors = append(a.errors, fmt.Errorf("%s: %s", pos.String(), message))
	} else {
		a.errors = append(a.errors, fmt.Errorf("%s", message))
	}
}

// resolveType converts an AST type expression to a Type
func (a *Analyzer) resolveType(typeExpr ast.Expr) types.Type {
	// For now, we only support identifier types
	if ident, ok := typeExpr.(*ast.IdentifierExpr); ok {
		// Check built-in types
		switch ident.Name {
		case "int":
			return types.Int
		case "float":
			return types.Float
		case "bool":
			return types.Bool
		case "string":
			return types.String
		case "char":
			return types.Char
		case "void":
			return types.Void
		}

		// Look up user-defined type
		symbol := a.currentScope.Lookup(ident.Name)
		if symbol == nil {
			a.error(ident.Pos(), fmt.Sprintf("undefined type: %s", ident.Name))
			return types.Invalid
		}

		if symbol.Kind != symtab.SymbolType && symbol.Kind != symtab.SymbolStruct {
			a.error(ident.Pos(), fmt.Sprintf("%s is not a type", ident.Name))
			return types.Invalid
		}

		return symbol.Type
	}

	a.error(typeExpr.Pos(), "invalid type expression")
	return types.Invalid
}

// assignable checks if valueType can be assigned to targetType
// Reports an error if not assignable
func (a *Analyzer) assignable(valueType, targetType types.Type, pos lexer.Position) bool {
	if valueType.AssignableTo(targetType) {
		return true
	}

	a.error(pos, fmt.Sprintf("cannot assign %s to %s", valueType, targetType))
	return false
}

// GetExprType returns the type of an expression (after analysis)
func (a *Analyzer) GetExprType(expr ast.Expr) types.Type {
	if t, ok := a.exprTypes[expr]; ok {
		return t
	}
	return types.Invalid
}

// GetScope returns the global scope (for inspection)
func (a *Analyzer) GetScope() *symtab.Scope {
	return a.globalScope
}
