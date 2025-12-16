package ast

import (
	"github.com/hassan/compiler/internal/lexer"
)

// Expression nodes represent values and computations.

// BinaryExpr represents a binary operation: left op right
// Examples: 2 + 3, x * y, a == b, foo && bar
//
// DESIGN CHOICE: Single node type for all binary operators because:
// - They all have the same structure (left, operator, right)
// - The operator token distinguishes them
// - Simpler than having separate nodes for each operator
//
// Alternative: Separate nodes (AddExpr, MulExpr, etc.) would:
// - Be more type-safe
// - Require more code
// - Make the visitor interface huge
// - Not provide much benefit (operator is already strongly typed)
type BinaryExpr struct {
	Left     Expr
	Operator lexer.Token // The operator token (+, -, *, /, etc.)
	Right    Expr
}

func (b *BinaryExpr) Pos() lexer.Position { return b.Left.Pos() }
func (b *BinaryExpr) End() lexer.Position { return b.Right.End() }
func (b *BinaryExpr) exprNode()           {}
func (b *BinaryExpr) Accept(v Visitor) (interface{}, error) {
	return v.VisitBinaryExpr(b)
}

// UnaryExpr represents a unary operation: op operand
// Examples: -x, !flag, ~bits, ++i, i++
//
// DESIGN CHOICE: We use a single field for the operator rather than separate
// prefix/postfix operators because:
// - The token type (TokenPlusPlus vs TokenMinusMinus) already distinguishes them
// - We store whether it's prefix or postfix in a boolean
// - Simpler visitor interface
type UnaryExpr struct {
	Operator  lexer.Token
	Operand   Expr
	IsPostfix bool // true for i++, false for ++i
}

func (u *UnaryExpr) Pos() lexer.Position {
	if u.IsPostfix {
		return u.Operand.Pos()
	}
	return u.Operator.Position
}
func (u *UnaryExpr) End() lexer.Position {
	if u.IsPostfix {
		// For postfix (i++), the operator comes after the operand
		return u.Operator.Position
	}
	return u.Operand.End()
}
func (u *UnaryExpr) exprNode() {}
func (u *UnaryExpr) Accept(v Visitor) (interface{}, error) {
	return v.VisitUnaryExpr(u)
}

// LiteralExpr represents a literal value: numbers, strings, booleans, nil
// Examples: 42, 3.14, "hello", true, nil
//
// DESIGN CHOICE: Store the value as interface{} rather than separate fields because:
// - Simpler node structure
// - The lexer has already parsed the value
// - Type can be determined from Value's runtime type
//
// The Value field contains:
// - int64 for integer literals
// - float64 for floating-point literals
// - string for string literals
// - bool for boolean literals
// - nil for nil literal
type LiteralExpr struct {
	Token lexer.Token
	Value interface{} // The actual value
}

func (l *LiteralExpr) Pos() lexer.Position { return l.Token.Position }
func (l *LiteralExpr) End() lexer.Position {
	return lexer.Position{
		Filename: l.Token.Position.Filename,
		Line:     l.Token.Position.Line,
		Column:   l.Token.Position.Column + len(l.Token.Lexeme),
		Offset:   l.Token.Position.Offset + l.Token.Length,
	}
}
func (l *LiteralExpr) exprNode() {}
func (l *LiteralExpr) Accept(v Visitor) (interface{}, error) {
	return v.VisitLiteralExpr(l)
}

// IdentifierExpr represents a variable or function name: foo, bar, _temp
//
// DESIGN CHOICE: Separate from LiteralExpr even though it's also a "leaf" node because:
// - Identifiers need name resolution (lookup in symbol table)
// - Literals don't need resolution
// - Type checking is different
// - Makes semantic analysis clearer
type IdentifierExpr struct {
	Token lexer.Token
	Name  string // The identifier name
}

func (i *IdentifierExpr) Pos() lexer.Position { return i.Token.Position }
func (i *IdentifierExpr) End() lexer.Position {
	return lexer.Position{
		Filename: i.Token.Position.Filename,
		Line:     i.Token.Position.Line,
		Column:   i.Token.Position.Column + len(i.Name),
		Offset:   i.Token.Position.Offset + len(i.Name),
	}
}
func (i *IdentifierExpr) exprNode() {}
func (i *IdentifierExpr) Accept(v Visitor) (interface{}, error) {
	return v.VisitIdentifierExpr(i)
}

// CallExpr represents a function call: foo(1, 2, 3)
//
// COMPONENTS:
// - Callee: the function being called (can be any expression: foo, obj.method, functions[0])
// - Args: the arguments
// - LeftParen/RightParen: for position tracking
//
// DESIGN CHOICE: Callee is an Expr rather than just an identifier because:
// - Supports method calls: obj.method()
// - Supports function pointers: functionArray[0]()
// - Supports higher-order functions: getFunction()()
type CallExpr struct {
	Callee     Expr
	LeftParen  lexer.Token // Position of '('
	Args       []Expr
	RightParen lexer.Token // Position of ')'
}

func (c *CallExpr) Pos() lexer.Position { return c.Callee.Pos() }
func (c *CallExpr) End() lexer.Position { return c.RightParen.Position }
func (c *CallExpr) exprNode()           {}
func (c *CallExpr) Accept(v Visitor) (interface{}, error) {
	return v.VisitCallExpr(c)
}

// IndexExpr represents array/map indexing: arr[i], map[key]
//
// DESIGN CHOICE: Single node for both array and map access because:
// - Same syntax
// - Semantic analyzer can distinguish based on type
// - Simpler AST
type IndexExpr struct {
	Object       Expr
	LeftBracket  lexer.Token // Position of '['
	Index        Expr
	RightBracket lexer.Token // Position of ']'
}

func (i *IndexExpr) Pos() lexer.Position { return i.Object.Pos() }
func (i *IndexExpr) End() lexer.Position { return i.RightBracket.Position }
func (i *IndexExpr) exprNode()           {}
func (i *IndexExpr) Accept(v Visitor) (interface{}, error) {
	return v.VisitIndexExpr(i)
}

// MemberExpr represents member access: obj.field, point.x
//
// COMPONENTS:
// - Object: the thing we're accessing a member of
// - Dot: position of '.'
// - Member: the field/method name
//
// DESIGN CHOICE: Store Member as IdentifierExpr rather than just string because:
// - Preserves position information (useful for "go to definition")
// - Consistent with how we handle identifiers elsewhere
// - Could support computed member access later (obj[expr])
type MemberExpr struct {
	Object Expr
	Dot    lexer.Token
	Member *IdentifierExpr
}

func (m *MemberExpr) Pos() lexer.Position { return m.Object.Pos() }
func (m *MemberExpr) End() lexer.Position { return m.Member.End() }
func (m *MemberExpr) exprNode()           {}
func (m *MemberExpr) Accept(v Visitor) (interface{}, error) {
	return v.VisitMemberExpr(m)
}

// AssignmentExpr represents assignment: x = 5, arr[i] = 10
//
// COMPONENTS:
// - Target: what we're assigning to (identifier, index, member access)
// - Operator: the assignment operator (=, +=, -=, etc.)
// - Value: the value being assigned
//
// DESIGN CHOICE: Assignment is an expression (not a statement) because:
// - Allows chaining: x = y = 5
// - Can be used in conditions: if (x = foo()) != nil
// - Matches C/Java/Go semantics
//
// However, we could make it a statement for a simpler language.
type AssignmentExpr struct {
	Target   Expr
	Operator lexer.Token
	Value    Expr
}

func (a *AssignmentExpr) Pos() lexer.Position { return a.Target.Pos() }
func (a *AssignmentExpr) End() lexer.Position { return a.Value.End() }
func (a *AssignmentExpr) exprNode()           {}
func (a *AssignmentExpr) Accept(v Visitor) (interface{}, error) {
	return v.VisitAssignmentExpr(a)
}

// LogicalExpr represents logical operations: x && y, a || b
//
// DESIGN CHOICE: Separate from BinaryExpr because logical operators:
// - Have short-circuit evaluation (don't evaluate right if not needed)
// - Need different code generation (branch instructions)
// - Have different type requirements (both operands must be boolean)
//
// This makes semantic analysis and code generation clearer.
type LogicalExpr struct {
	Left     Expr
	Operator lexer.Token // && or ||
	Right    Expr
}

func (l *LogicalExpr) Pos() lexer.Position { return l.Left.Pos() }
func (l *LogicalExpr) End() lexer.Position { return l.Right.End() }
func (l *LogicalExpr) exprNode()           {}
func (l *LogicalExpr) Accept(v Visitor) (interface{}, error) {
	return v.VisitLogicalExpr(l)
}

// GroupingExpr represents parenthesized expressions: (2 + 3) * 4
//
// DESIGN CHOICE: Keep grouping in the AST even though it doesn't affect semantics because:
// - Code formatters need to preserve it
// - It's useful for debugging (see exactly what user wrote)
// - No significant cost
//
// Alternative: Don't store grouping (just store the inner expression).
// - Simpler AST
// - Loses information about user's intent
type GroupingExpr struct {
	LeftParen  lexer.Token
	Expression Expr
	RightParen lexer.Token
}

func (g *GroupingExpr) Pos() lexer.Position { return g.LeftParen.Position }
func (g *GroupingExpr) End() lexer.Position { return g.RightParen.Position }
func (g *GroupingExpr) exprNode()           {}
func (g *GroupingExpr) Accept(v Visitor) (interface{}, error) {
	return v.VisitGroupingExpr(g)
}

// ArrayLiteralExpr represents array literals: [1, 2, 3], []int{1, 2, 3}
//
// COMPONENTS:
// - ElementType: optional type annotation (for []int{...})
// - Elements: the array elements
// - LeftBracket/RightBrace: for position tracking
//
// DESIGN CHOICE: Support both [1, 2, 3] and []int{1, 2, 3} syntax because:
// - First is convenient (type inference)
// - Second is explicit (useful when type can't be inferred)
// - Matches Go's approach
type ArrayLiteralExpr struct {
	LeftBracket lexer.Token
	ElementType Expr // Optional type (nil if not specified)
	Elements    []Expr
	RightBrace  lexer.Token
}

func (a *ArrayLiteralExpr) Pos() lexer.Position { return a.LeftBracket.Position }
func (a *ArrayLiteralExpr) End() lexer.Position { return a.RightBrace.Position }
func (a *ArrayLiteralExpr) exprNode()           {}
func (a *ArrayLiteralExpr) Accept(v Visitor) (interface{}, error) {
	return v.VisitArrayLiteralExpr(a)
}

// StructLiteralExpr represents struct literals: Point{x: 1, y: 2}
//
// COMPONENTS:
// - Type: the struct type name
// - Fields: field initializers (name: value pairs)
// - LeftBrace/RightBrace: for position tracking
//
// DESIGN CHOICE: Store fields as a slice of FieldInit rather than a map because:
// - Preserves order (useful for error messages and formatting)
// - Allows duplicate field checking (error if same field appears twice)
// - Simpler to iterate over
type StructLiteralExpr struct {
	TypeName   *IdentifierExpr
	LeftBrace  lexer.Token
	Fields     []*FieldInit
	RightBrace lexer.Token
}

func (s *StructLiteralExpr) Pos() lexer.Position { return s.TypeName.Pos() }
func (s *StructLiteralExpr) End() lexer.Position { return s.RightBrace.Position }
func (s *StructLiteralExpr) exprNode()           {}
func (s *StructLiteralExpr) Accept(v Visitor) (interface{}, error) {
	return v.VisitStructLiteralExpr(s)
}

// FieldInit represents a field initializer in a struct literal: name: value
type FieldInit struct {
	Name  *IdentifierExpr
	Colon lexer.Token
	Value Expr
}

func (f *FieldInit) Pos() lexer.Position { return f.Name.Pos() }
func (f *FieldInit) End() lexer.Position { return f.Value.End() }
