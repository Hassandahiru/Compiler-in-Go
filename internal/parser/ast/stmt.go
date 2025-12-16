package ast

import (
	"github.com/hassan/compiler/internal/lexer"
)

// Statement nodes represent actions and control flow.

// ExprStmt represents an expression used as a statement: foo(); x = 5;
//
// DESIGN CHOICE: Wrap expressions in a statement node because:
// - Clear distinction between expression and statement contexts
// - Some expressions aren't valid statements (literals, binary ops)
// - Parser can validate this (e.g., disallow "2 + 3;" as a statement)
//
// In our language, valid expression statements are:
// - Function calls
// - Assignments
// - Increment/decrement
type ExprStmt struct {
	Expression Expr
}

func (e *ExprStmt) Pos() lexer.Position { return e.Expression.Pos() }
func (e *ExprStmt) End() lexer.Position { return e.Expression.End() }
func (e *ExprStmt) stmtNode()           {}
func (e *ExprStmt) Accept(v Visitor) error {
	return v.VisitExprStmt(e)
}

// BlockStmt represents a block of statements: { stmt1; stmt2; ... }
//
// DESIGN CHOICE: Blocks create a new scope for variables.
// This is enforced during semantic analysis, not parsing.
//
// Blocks are used in:
// - Function bodies
// - If/else branches
// - Loop bodies
// - Standalone blocks (for scoping)
type BlockStmt struct {
	LeftBrace  lexer.Token
	Statements []Stmt
	RightBrace lexer.Token
}

func (b *BlockStmt) Pos() lexer.Position { return b.LeftBrace.Position }
func (b *BlockStmt) End() lexer.Position { return b.RightBrace.Position }
func (b *BlockStmt) stmtNode()           {}
func (b *BlockStmt) Accept(v Visitor) error {
	return v.VisitBlockStmt(b)
}

// IfStmt represents an if statement: if (cond) { ... } else { ... }
//
// COMPONENTS:
// - Condition: boolean expression
// - ThenBranch: executed if condition is true
// - ElseBranch: optional, executed if condition is false
//
// DESIGN CHOICE: Store ElseBranch as Stmt (not *BlockStmt) because:
// - Allows if-else chains: if (a) {} else if (b) {} else {}
// - More flexible (could support single-statement else, though we require blocks)
type IfStmt struct {
	IfPos      lexer.Position
	Condition  Expr
	ThenBranch *BlockStmt
	ElseBranch Stmt // Can be nil, another IfStmt, or a BlockStmt
}

func (i *IfStmt) Pos() lexer.Position { return i.IfPos }
func (i *IfStmt) End() lexer.Position {
	if i.ElseBranch != nil {
		return i.ElseBranch.End()
	}
	return i.ThenBranch.End()
}
func (i *IfStmt) stmtNode() {}
func (i *IfStmt) Accept(v Visitor) error {
	return v.VisitIfStmt(i)
}

// WhileStmt represents a while loop: while (cond) { ... }
//
// DESIGN CHOICE: Separate While and For rather than unifying them because:
// - Clearer semantics (while is simpler)
// - Easier to understand in error messages
// - Code generation can optimize differently
//
// while (cond) { body } is semantically:
//   loop {
//     if (!cond) break
//     body
//   }
type WhileStmt struct {
	WhilePos  lexer.Position
	Condition Expr
	Body      *BlockStmt
}

func (w *WhileStmt) Pos() lexer.Position { return w.WhilePos }
func (w *WhileStmt) End() lexer.Position { return w.Body.End() }
func (w *WhileStmt) stmtNode()           {}
func (w *WhileStmt) Accept(v Visitor) error {
	return v.VisitWhileStmt(w)
}

// ForStmt represents a for loop: for (init; cond; post) { ... }
//
// COMPONENTS:
// - Init: optional initialization (var i = 0 or i = 0)
// - Condition: optional condition (i < 10)
// - Post: optional post-iteration statement (i++)
// - Body: loop body
//
// DESIGN CHOICE: All parts are optional because:
// - for (;;) {} is an infinite loop (like while (true))
// - for (i < 10;) {} is like while (i < 10)
// - Flexibility without adding more node types
//
// SEMANTIC NOTE: Init can declare variables that are scoped to the loop:
//   for (var i = 0; i < 10; i++) { ... }
// Variable i is not visible after the loop.
type ForStmt struct {
	ForPos    lexer.Position
	Init      Stmt // Can be nil, VarDecl, or ExprStmt
	Condition Expr // Can be nil (means infinite loop)
	Post      Stmt // Can be nil or ExprStmt
	Body      *BlockStmt
}

func (f *ForStmt) Pos() lexer.Position { return f.ForPos }
func (f *ForStmt) End() lexer.Position { return f.Body.End() }
func (f *ForStmt) stmtNode()           {}
func (f *ForStmt) Accept(v Visitor) error {
	return v.VisitForStmt(f)
}

// ReturnStmt represents a return statement: return expr;
//
// COMPONENTS:
// - ReturnPos: position of 'return' keyword
// - Value: optional return value (nil for void functions)
//
// DESIGN CHOICE: Value is optional (can be nil) because:
// - Void functions just use "return" without a value
// - Clearer than requiring a special "void" expression
type ReturnStmt struct {
	ReturnPos lexer.Position
	Value     Expr // Can be nil for void return
}

func (r *ReturnStmt) Pos() lexer.Position { return r.ReturnPos }
func (r *ReturnStmt) End() lexer.Position {
	if r.Value != nil {
		return r.Value.End()
	}
	// Return just the keyword position + length of "return"
	return lexer.Position{
		Filename: r.ReturnPos.Filename,
		Line:     r.ReturnPos.Line,
		Column:   r.ReturnPos.Column + 6, // len("return")
		Offset:   r.ReturnPos.Offset + 6,
	}
}
func (r *ReturnStmt) stmtNode() {}
func (r *ReturnStmt) Accept(v Visitor) error {
	return v.VisitReturnStmt(r)
}

// BreakStmt represents a break statement: break;
//
// SEMANTIC NOTE: Break must appear inside a loop or switch.
// This is validated during semantic analysis.
type BreakStmt struct {
	BreakPos lexer.Position
}

func (b *BreakStmt) Pos() lexer.Position { return b.BreakPos }
func (b *BreakStmt) End() lexer.Position {
	return lexer.Position{
		Filename: b.BreakPos.Filename,
		Line:     b.BreakPos.Line,
		Column:   b.BreakPos.Column + 5, // len("break")
		Offset:   b.BreakPos.Offset + 5,
	}
}
func (b *BreakStmt) stmtNode() {}
func (b *BreakStmt) Accept(v Visitor) error {
	return v.VisitBreakStmt(b)
}

// ContinueStmt represents a continue statement: continue;
//
// SEMANTIC NOTE: Continue must appear inside a loop (not switch).
// This is validated during semantic analysis.
type ContinueStmt struct {
	ContinuePos lexer.Position
}

func (c *ContinueStmt) Pos() lexer.Position { return c.ContinuePos }
func (c *ContinueStmt) End() lexer.Position {
	return lexer.Position{
		Filename: c.ContinuePos.Filename,
		Line:     c.ContinuePos.Line,
		Column:   c.ContinuePos.Column + 8, // len("continue")
		Offset:   c.ContinuePos.Offset + 8,
	}
}
func (c *ContinueStmt) stmtNode() {}
func (c *ContinueStmt) Accept(v Visitor) error {
	return v.VisitContinueStmt(c)
}

// SwitchStmt represents a switch statement:
//   switch (expr) {
//     case value1: stmts...
//     case value2: stmts...
//     default: stmts...
//   }
//
// DESIGN CHOICES:
// - No fallthrough (each case is independent, no need for break)
// - Default case is optional
// - Case values must be constants (validated during semantic analysis)
//
// This is simpler and safer than C-style switches.
type SwitchStmt struct {
	SwitchPos lexer.Position
	Value     Expr // The value being switched on
	Cases     []*CaseClause
}

func (s *SwitchStmt) Pos() lexer.Position { return s.SwitchPos }
func (s *SwitchStmt) End() lexer.Position {
	if len(s.Cases) > 0 {
		return s.Cases[len(s.Cases)-1].End()
	}
	// Just the switch keyword if no cases (error case)
	return lexer.Position{
		Filename: s.SwitchPos.Filename,
		Line:     s.SwitchPos.Line,
		Column:   s.SwitchPos.Column + 6, // len("switch")
		Offset:   s.SwitchPos.Offset + 6,
	}
}
func (s *SwitchStmt) stmtNode() {}
func (s *SwitchStmt) Accept(v Visitor) error {
	return v.VisitSwitchStmt(s)
}

// CaseClause represents a case in a switch statement.
//
// COMPONENTS:
// - Values: the case values (can be multiple: case 1, 2, 3:)
// - Body: statements to execute
// - IsDefault: true for default case
//
// DESIGN CHOICE: Allow multiple values per case (case 1, 2, 3:) because:
// - Common pattern in many languages
// - More concise than multiple cases
// - No semantic difference from multiple cases
type CaseClause struct {
	CasePos   lexer.Position
	Values    []Expr // Empty for default case
	Colon     lexer.Token
	Body      []Stmt
	IsDefault bool
}

func (c *CaseClause) Pos() lexer.Position { return c.CasePos }
func (c *CaseClause) End() lexer.Position {
	if len(c.Body) > 0 {
		return c.Body[len(c.Body)-1].End()
	}
	return c.Colon.Position
}

// Declaration nodes represent introducing new names.

// VarDecl represents a variable declaration: var x int = 5;
//
// COMPONENTS:
// - Names: variable names (can declare multiple: var x, y, z int)
// - Type: optional type annotation (nil if inferred)
// - Initializer: optional initial value (nil if not initialized)
//
// DESIGN CHOICES:
// - Support multiple declarations: var x, y int
// - Type is optional (inferred from initializer)
// - Initializer is optional (default to zero value)
// - If both Type and Initializer are nil, that's an error (validated during parsing/semantic analysis)
type VarDecl struct {
	VarPos      lexer.Position
	Names       []*IdentifierExpr
	Type        Expr // Can be nil (type inference)
	Initializer Expr // Can be nil (default initialization)
}

func (v *VarDecl) Pos() lexer.Position { return v.VarPos }
func (v *VarDecl) End() lexer.Position {
	if v.Initializer != nil {
		return v.Initializer.End()
	}
	if v.Type != nil {
		return v.Type.End()
	}
	return v.Names[len(v.Names)-1].End()
}
func (v *VarDecl) stmtNode() {}
func (v *VarDecl) declNode() {}
func (v *VarDecl) Accept(v2 Visitor) error {
	return v2.VisitVarDecl(v)
}

// FuncDecl represents a function declaration:
//   func name(param1 type1, param2 type2) returnType { body }
//
// COMPONENTS:
// - Name: function name
// - Params: parameter list
// - ReturnType: return type (nil for void)
// - Body: function body
//
// DESIGN CHOICES:
// - Body is optional (nil for function declarations without implementation)
// - ReturnType is optional (nil for void functions)
// - Params use the same structure as VarDecl (for consistency)
type FuncDecl struct {
	FuncPos    lexer.Position
	Name       *IdentifierExpr
	Params     []*Parameter
	ReturnType Expr // Can be nil for void
	Body       *BlockStmt
}

func (f *FuncDecl) Pos() lexer.Position { return f.FuncPos }
func (f *FuncDecl) End() lexer.Position {
	if f.Body != nil {
		return f.Body.End()
	}
	if f.ReturnType != nil {
		return f.ReturnType.End()
	}
	// End at the closing paren of parameters
	if len(f.Params) > 0 {
		return f.Params[len(f.Params)-1].End()
	}
	return f.Name.End()
}
func (f *FuncDecl) stmtNode() {}
func (f *FuncDecl) declNode() {}
func (f *FuncDecl) Accept(v Visitor) error {
	return v.VisitFuncDecl(f)
}

// Parameter represents a function parameter: name type
type Parameter struct {
	Name *IdentifierExpr
	Type Expr
}

func (p *Parameter) Pos() lexer.Position { return p.Name.Pos() }
func (p *Parameter) End() lexer.Position { return p.Type.End() }

// TypeDecl represents a type alias declaration: type Name = OtherType
//
// EXAMPLE: type StringMap = map[string]string
//
// DESIGN CHOICE: Type aliases (not new types) because:
// - Simpler to implement
// - Good enough for most use cases
// - Can add "new types" later if needed
type TypeDecl struct {
	TypePos lexer.Position
	Name    *IdentifierExpr
	Type    Expr
}

func (t *TypeDecl) Pos() lexer.Position { return t.TypePos }
func (t *TypeDecl) End() lexer.Position { return t.Type.End() }
func (t *TypeDecl) stmtNode()           {}
func (t *TypeDecl) declNode()           {}
func (t *TypeDecl) Accept(v Visitor) error {
	return v.VisitTypeDecl(t)
}

// StructDecl represents a struct type declaration:
//   struct Name {
//     field1 type1
//     field2 type2
//   }
//
// DESIGN CHOICE: Separate from TypeDecl because:
// - Structs are common and deserve special handling
// - Easier to extend (add methods, interfaces, etc.)
// - Clearer error messages
type StructDecl struct {
	StructPos  lexer.Position
	Name       *IdentifierExpr
	LeftBrace  lexer.Token
	Fields     []*FieldDecl
	RightBrace lexer.Token
}

func (s *StructDecl) Pos() lexer.Position { return s.StructPos }
func (s *StructDecl) End() lexer.Position { return s.RightBrace.Position }
func (s *StructDecl) stmtNode()           {}
func (s *StructDecl) declNode()           {}
func (s *StructDecl) Accept(v Visitor) error {
	return v.VisitStructDecl(s)
}

// FieldDecl represents a field in a struct: name type
type FieldDecl struct {
	Name *IdentifierExpr
	Type Expr
}

func (f *FieldDecl) Pos() lexer.Position { return f.Name.Pos() }
func (f *FieldDecl) End() lexer.Position { return f.Type.End() }
