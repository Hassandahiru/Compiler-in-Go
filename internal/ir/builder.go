package ir

import (
	"fmt"

	"github.com/hassan/compiler/internal/lexer"
	"github.com/hassan/compiler/internal/parser/ast"
	"github.com/hassan/compiler/internal/semantic"
	"github.com/hassan/compiler/internal/semantic/types"
	"github.com/hassan/compiler/internal/symtab"
)

// Builder constructs IR from a typed AST.
//
// DESIGN PHILOSOPHY:
// The builder is a visitor that traverses the AST and emits IR instructions.
// It maintains:
// - Current function and basic block
// - Variable mappings (AST symbols to IR values)
// - Control flow context (break/continue targets)
//
// DESIGN CHOICE: Separate builder from analyzer because:
// - Clean separation of concerns
// - Can run on already-analyzed AST
// - Easier to test independently
type Builder struct {
	// module is the IR module being built
	module *Module

	// analyzer provides type information
	analyzer *semantic.Analyzer

	// currentFunc is the function being built
	currentFunc *Function

	// currentBlock is the basic block being built
	currentBlock *BasicBlock

	// variables maps symbols to their IR values
	variables map[*symtab.Symbol]*Value

	// namedValues maps variable names to their IR values (for local lookup)
	namedValues map[string]*Value

	// breakTarget is the block to jump to on break
	breakTarget *BasicBlock

	// continueTarget is the block to jump to on continue
	continueTarget *BasicBlock

	// errors accumulates IR generation errors
	errors []error
}

// NewBuilder creates a new IR builder.
func NewBuilder(analyzer *semantic.Analyzer) *Builder {
	return &Builder{
		analyzer:    analyzer,
		variables:   make(map[*symtab.Symbol]*Value),
		namedValues: make(map[string]*Value),
		errors:      make([]error, 0),
	}
}

// Build generates IR for a file.
func (b *Builder) Build(file *ast.File) (*Module, []error) {
	// Create module
	b.module = NewModule(file.Package.Name.Name)

	// Generate IR for each declaration
	for _, decl := range file.Decls {
		b.buildDecl(decl)
	}

	return b.module, b.errors
}

// buildDecl generates IR for a declaration.
func (b *Builder) buildDecl(decl ast.Decl) {
	switch d := decl.(type) {
	case *ast.FuncDecl:
		b.buildFunction(d)
	case *ast.VarDecl:
		b.buildGlobalVar(d)
	// Struct and type declarations don't generate IR
	// They're just type information used by the semantic analyzer
	}
}

// buildFunction generates IR for a function.
func (b *Builder) buildFunction(decl *ast.FuncDecl) {
	// Look up function symbol to get type
	scope := b.analyzer.GetScope()
	symbol := scope.Lookup(decl.Name.Name)
	if symbol == nil {
		b.error(decl.Pos(), "function symbol not found")
		return
	}

	funcType := symbol.Type.(*types.FunctionType)

	// Create parameter values
	params := make([]*Value, len(decl.Params))
	for i, param := range decl.Params {
		params[i] = &Value{
			ID:   i,
			Name: param.Name.Name,
			Type: funcType.Parameters[i],
			Kind: ValueParameter,
		}
	}

	// Create function
	b.currentFunc = NewFunction(decl.Name.Name, params, funcType.ReturnType)
	b.currentBlock = b.currentFunc.Entry

	// Reset named values for this function
	b.namedValues = make(map[string]*Value)

	// Map parameters to values by name
	for i, param := range decl.Params {
		b.namedValues[param.Name.Name] = params[i]
	}

	// Generate body
	if decl.Body != nil {
		b.buildStmt(decl.Body)

		// Add implicit return for void functions if needed
		if funcType.ReturnType.Equals(types.Void) && !b.currentBlock.IsTerminated() {
			b.currentBlock.AddInstruction(&Return{Value: nil})
		}
	}

	// Add function to module
	b.module.AddFunction(b.currentFunc)

	// Clean up
	b.currentFunc = nil
	b.currentBlock = nil
}

// buildGlobalVar generates IR for a global variable.
func (b *Builder) buildGlobalVar(decl *ast.VarDecl) {
	// For now, just create the global value
	// Initialization will be handled specially
	scope := b.analyzer.GetScope()
	for _, name := range decl.Names {
		symbol := scope.Lookup(name.Name)
		if symbol != nil {
			global := &Value{
				ID:   len(b.module.Globals),
				Name: name.Name,
				Type: symbol.Type,
				Kind: ValueVariable,
			}
			b.module.Globals = append(b.module.Globals, global)
			b.variables[symbol] = global
		}
	}
}

// buildStmt generates IR for a statement.
func (b *Builder) buildStmt(stmt ast.Stmt) {
	switch s := stmt.(type) {
	case *ast.ExprStmt:
		b.buildExpr(s.Expression)

	case *ast.BlockStmt:
		for _, inner := range s.Statements {
			b.buildStmt(inner)
		}

	case *ast.IfStmt:
		b.buildIf(s)

	case *ast.WhileStmt:
		b.buildWhile(s)

	case *ast.ForStmt:
		b.buildFor(s)

	case *ast.ReturnStmt:
		b.buildReturn(s)

	case *ast.BreakStmt:
		if b.breakTarget != nil {
			b.currentBlock.AddInstruction(&Jump{Target: b.breakTarget})
		}

	case *ast.ContinueStmt:
		if b.continueTarget != nil {
			b.currentBlock.AddInstruction(&Jump{Target: b.continueTarget})
		}

	case *ast.VarDecl:
		b.buildLocalVar(s)
	}
}

// buildIf generates IR for an if statement.
func (b *Builder) buildIf(stmt *ast.IfStmt) {
	// Evaluate condition
	cond := b.buildExpr(stmt.Condition)

	// Create blocks
	thenBlock := b.currentFunc.NewBasicBlockInFunc("if.then")
	endBlock := b.currentFunc.NewBasicBlockInFunc("if.end")

	var elseBlock *BasicBlock
	if stmt.ElseBranch != nil {
		elseBlock = b.currentFunc.NewBasicBlockInFunc("if.else")
	} else {
		elseBlock = endBlock
	}

	// Branch
	b.currentBlock.AddInstruction(&Branch{
		Condition:  cond,
		TrueBlock:  thenBlock,
		FalseBlock: elseBlock,
	})
	b.currentBlock.AddSuccessor(thenBlock)
	b.currentBlock.AddSuccessor(elseBlock)

	// Then block
	b.currentBlock = thenBlock
	b.buildStmt(stmt.ThenBranch)
	if !b.currentBlock.IsTerminated() {
		b.currentBlock.AddInstruction(&Jump{Target: endBlock})
		b.currentBlock.AddSuccessor(endBlock)
	}

	// Else block
	if stmt.ElseBranch != nil {
		b.currentBlock = elseBlock
		b.buildStmt(stmt.ElseBranch)
		if !b.currentBlock.IsTerminated() {
			b.currentBlock.AddInstruction(&Jump{Target: endBlock})
			b.currentBlock.AddSuccessor(endBlock)
		}
	}

	b.currentBlock = endBlock
}

// buildWhile generates IR for a while loop.
func (b *Builder) buildWhile(stmt *ast.WhileStmt) {
	condBlock := b.currentFunc.NewBasicBlockInFunc("while.cond")
	bodyBlock := b.currentFunc.NewBasicBlockInFunc("while.body")
	endBlock := b.currentFunc.NewBasicBlockInFunc("while.end")

	// Save break/continue targets
	oldBreak := b.breakTarget
	oldContinue := b.continueTarget
	b.breakTarget = endBlock
	b.continueTarget = condBlock

	// Jump to condition
	b.currentBlock.AddInstruction(&Jump{Target: condBlock})
	b.currentBlock.AddSuccessor(condBlock)

	// Condition block
	b.currentBlock = condBlock
	cond := b.buildExpr(stmt.Condition)
	b.currentBlock.AddInstruction(&Branch{
		Condition:  cond,
		TrueBlock:  bodyBlock,
		FalseBlock: endBlock,
	})
	b.currentBlock.AddSuccessor(bodyBlock)
	b.currentBlock.AddSuccessor(endBlock)

	// Body block
	b.currentBlock = bodyBlock
	b.buildStmt(stmt.Body)
	if !b.currentBlock.IsTerminated() {
		b.currentBlock.AddInstruction(&Jump{Target: condBlock})
		b.currentBlock.AddSuccessor(condBlock)
	}

	// Restore break/continue targets
	b.breakTarget = oldBreak
	b.continueTarget = oldContinue

	b.currentBlock = endBlock
}

// buildFor generates IR for a for loop.
func (b *Builder) buildFor(stmt *ast.ForStmt) {
	// Init
	if stmt.Init != nil {
		b.buildStmt(stmt.Init)
	}

	condBlock := b.currentFunc.NewBasicBlockInFunc("for.cond")
	bodyBlock := b.currentFunc.NewBasicBlockInFunc("for.body")
	postBlock := b.currentFunc.NewBasicBlockInFunc("for.post")
	endBlock := b.currentFunc.NewBasicBlockInFunc("for.end")

	// Save break/continue targets
	oldBreak := b.breakTarget
	oldContinue := b.continueTarget
	b.breakTarget = endBlock
	b.continueTarget = postBlock

	// Jump to condition
	b.currentBlock.AddInstruction(&Jump{Target: condBlock})
	b.currentBlock.AddSuccessor(condBlock)

	// Condition block
	b.currentBlock = condBlock
	if stmt.Condition != nil {
		cond := b.buildExpr(stmt.Condition)
		b.currentBlock.AddInstruction(&Branch{
			Condition:  cond,
			TrueBlock:  bodyBlock,
			FalseBlock: endBlock,
		})
	} else {
		// Infinite loop
		b.currentBlock.AddInstruction(&Jump{Target: bodyBlock})
	}
	b.currentBlock.AddSuccessor(bodyBlock)
	b.currentBlock.AddSuccessor(endBlock)

	// Body block
	b.currentBlock = bodyBlock
	b.buildStmt(stmt.Body)
	if !b.currentBlock.IsTerminated() {
		b.currentBlock.AddInstruction(&Jump{Target: postBlock})
		b.currentBlock.AddSuccessor(postBlock)
	}

	// Post block
	b.currentBlock = postBlock
	if stmt.Post != nil {
		b.buildStmt(stmt.Post)
	}
	b.currentBlock.AddInstruction(&Jump{Target: condBlock})
	b.currentBlock.AddSuccessor(condBlock)

	// Restore break/continue targets
	b.breakTarget = oldBreak
	b.continueTarget = oldContinue

	b.currentBlock = endBlock
}

// buildReturn generates IR for a return statement.
func (b *Builder) buildReturn(stmt *ast.ReturnStmt) {
	var value *Value
	if stmt.Value != nil {
		value = b.buildExpr(stmt.Value)
	}
	b.currentBlock.AddInstruction(&Return{Value: value})
}

// buildLocalVar generates IR for a local variable declaration.
func (b *Builder) buildLocalVar(decl *ast.VarDecl) {
	for _, name := range decl.Names {
		// Get type from analyzer
		varType := types.Int // Default, should get from semantic analysis
		if decl.Type != nil {
			// Type is specified - would resolve this properly
			varType = types.Int
		}

		// Allocate space for the variable
		alloca := b.currentFunc.NewValue(name.Name, varType, ValueVariable)
		b.currentFunc.Locals = append(b.currentFunc.Locals, alloca)
		b.namedValues[name.Name] = alloca

		// Initialize if there's an initializer
		if decl.Initializer != nil {
			initValue := b.buildExpr(decl.Initializer)
			// For now, just copy (simplified - real version would use store)
			b.currentBlock.AddInstruction(&Copy{
				Dest:  alloca,
				Value: initValue,
			})
		}
	}
}

// buildExpr generates IR for an expression and returns the resulting value.
func (b *Builder) buildExpr(expr ast.Expr) *Value {
	exprType := b.analyzer.GetExprType(expr)

	switch e := expr.(type) {
	case *ast.BinaryExpr:
		return b.buildBinary(e, exprType)

	case *ast.UnaryExpr:
		return b.buildUnary(e, exprType)

	case *ast.LiteralExpr:
		return b.buildLiteral(e, exprType)

	case *ast.IdentifierExpr:
		return b.buildIdentifier(e)

	case *ast.CallExpr:
		return b.buildCall(e, exprType)

	case *ast.AssignmentExpr:
		return b.buildAssignment(e)

	default:
		b.error(expr.Pos(), fmt.Sprintf("unsupported expression type: %T", expr))
		return b.currentFunc.NewTemp(types.Invalid)
	}
}

// buildBinary generates IR for a binary expression.
func (b *Builder) buildBinary(expr *ast.BinaryExpr, resultType types.Type) *Value {
	left := b.buildExpr(expr.Left)
	right := b.buildExpr(expr.Right)

	result := b.currentFunc.NewTemp(resultType)

	// Map token to IR operator
	var op BinaryOperator
	switch expr.Operator.Type {
	case lexer.TokenPlus:
		op = OpAdd
	case lexer.TokenMinus:
		op = OpSub
	case lexer.TokenStar:
		op = OpMul
	case lexer.TokenSlash:
		op = OpDiv
	case lexer.TokenPercent:
		op = OpMod
	case lexer.TokenEqual:
		op = OpEq
	case lexer.TokenNotEqual:
		op = OpNeq
	case lexer.TokenLess:
		op = OpLt
	case lexer.TokenLessEqual:
		op = OpLe
	case lexer.TokenGreater:
		op = OpGt
	case lexer.TokenGreaterEqual:
		op = OpGe
	case lexer.TokenBitAnd:
		op = OpBitAnd
	case lexer.TokenBitOr:
		op = OpBitOr
	case lexer.TokenBitXor:
		op = OpBitXor
	case lexer.TokenShl:
		op = OpShl
	case lexer.TokenShr:
		op = OpShr
	default:
		b.error(expr.Operator.Position, "unsupported binary operator")
		return result
	}

	b.currentBlock.AddInstruction(&BinaryOp{
		Op:    op,
		Dest:  result,
		Left:  left,
		Right: right,
	})

	return result
}

// buildUnary generates IR for a unary expression.
func (b *Builder) buildUnary(expr *ast.UnaryExpr, resultType types.Type) *Value {
	operand := b.buildExpr(expr.Operand)
	result := b.currentFunc.NewTemp(resultType)

	var op UnaryOperator
	switch expr.Operator.Type {
	case lexer.TokenMinus:
		op = OpNeg
	case lexer.TokenNot:
		op = OpNot
	case lexer.TokenBitNot:
		op = OpBitNot
	default:
		b.error(expr.Operator.Position, "unsupported unary operator")
		return result
	}

	b.currentBlock.AddInstruction(&UnaryOp{
		Op:      op,
		Dest:    result,
		Operand: operand,
	})

	return result
}

// buildLiteral generates IR for a literal.
func (b *Builder) buildLiteral(expr *ast.LiteralExpr, exprType types.Type) *Value {
	return &Value{
		ID:       -1, // Constants don't need IDs
		Type:     exprType,
		Kind:     ValueConstant,
		Constant: expr.Value,
	}
}

// buildIdentifier generates IR for an identifier reference.
func (b *Builder) buildIdentifier(expr *ast.IdentifierExpr) *Value {
	// Try named values first (local variables and parameters)
	if val, ok := b.namedValues[expr.Name]; ok {
		return val
	}

	// Check if it's a function - create a function reference
	scope := b.analyzer.GetScope()
	symbol := scope.Lookup(expr.Name)
	if symbol != nil && symbol.Kind == symtab.SymbolFunction {
		// Create a function value reference
		return &Value{
			ID:   -1, // Functions don't need IDs
			Name: expr.Name,
			Type: symbol.Type,
			Kind: ValueVariable, // Treat as variable for now
		}
	}

	// Try symbol-based lookup for globals
	if symbol == nil {
		b.error(expr.Pos(), "undefined variable")
		return b.currentFunc.NewTemp(types.Invalid)
	}

	if val, ok := b.variables[symbol]; ok {
		return val
	}

	b.error(expr.Pos(), "variable not mapped to IR value")
	return b.currentFunc.NewTemp(types.Invalid)
}

// buildCall generates IR for a function call.
func (b *Builder) buildCall(expr *ast.CallExpr, resultType types.Type) *Value {
	function := b.buildExpr(expr.Callee)

	args := make([]*Value, len(expr.Args))
	for i, arg := range expr.Args {
		args[i] = b.buildExpr(arg)
	}

	var result *Value
	if !resultType.Equals(types.Void) {
		result = b.currentFunc.NewTemp(resultType)
	}

	b.currentBlock.AddInstruction(&Call{
		Dest:     result,
		Function: function,
		Args:     args,
	})

	return result
}

// buildAssignment generates IR for an assignment.
func (b *Builder) buildAssignment(expr *ast.AssignmentExpr) *Value {
	value := b.buildExpr(expr.Value)

	// Get target
	if ident, ok := expr.Target.(*ast.IdentifierExpr); ok {
		// Try named values first
		if target, ok := b.namedValues[ident.Name]; ok {
			b.currentBlock.AddInstruction(&Copy{
				Dest:  target,
				Value: value,
			})
			return target
		}

		// Try symbol-based lookup
		scope := b.analyzer.GetScope()
		symbol := scope.Lookup(ident.Name)
		if symbol != nil {
			if target, ok := b.variables[symbol]; ok {
				b.currentBlock.AddInstruction(&Copy{
					Dest:  target,
					Value: value,
				})
				return target
			}
		}
	}

	return value
}

// error records an IR generation error.
func (b *Builder) error(pos lexer.Position, message string) {
	b.errors = append(b.errors, fmt.Errorf("%s: %s", pos.String(), message))
}
