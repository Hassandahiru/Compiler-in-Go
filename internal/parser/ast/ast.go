// Package ast defines the Abstract Syntax Tree node types for the compiler.
//
// DESIGN PHILOSOPHY:
// The AST is a tree representation of the program's structure. It:
// 1. Preserves program semantics (but not all syntax details)
// 2. Is easy to traverse and analyze
// 3. Supports the visitor pattern for operations
// 4. Maintains position information for error reporting
//
// KEY DESIGN CHOICES:
// - Use interfaces (Expr, Stmt) for polymorphism
// - Use the visitor pattern for operations (avoids type switches)
// - Store position info in every node (for errors and IDE features)
// - Use value types for small nodes, pointers for large/mutable ones
package ast

import (
	"github.com/hassan/compiler/internal/lexer"
)

// Node is the base interface for all AST nodes.
//
// DESIGN CHOICE: We have a Node interface even though it's mostly empty because:
// - It provides a common type for all AST nodes
// - It allows future extension (e.g., adding common methods)
// - It makes the type system clearer
//
// Every node must be able to report its position for error messages.
type Node interface {
	// Pos returns the starting position of this node in the source.
	Pos() lexer.Position

	// End returns the ending position of this node in the source.
	// This allows calculating the span: Span{Start: Pos(), End: End()}
	End() lexer.Position
}

// Expr is the interface for all expression nodes.
//
// WHAT IS AN EXPRESSION?
// An expression is a piece of code that produces a value.
// Examples: 2 + 3, foo(), x.y, true, "hello"
//
// DESIGN CHOICE: Separate interface from Stmt because:
// - Type safety (can't use a statement where expression is expected)
// - Clearer code (visitor methods explicitly show expression vs statement)
// - Matches language semantics (expressions have values, statements don't)
type Expr interface {
	Node
	// Accept implements the visitor pattern.
	// This allows operations on expressions without type switches.
	Accept(v Visitor) (interface{}, error)
	exprNode() // Marker method to prevent accidental interface satisfaction
}

// Stmt is the interface for all statement nodes.
//
// WHAT IS A STATEMENT?
// A statement is a piece of code that performs an action.
// Examples: if (x) {...}, for (...) {...}, return x, x = 5
//
// In some languages (like Ruby), statements also have values.
// In ours, they don't - only expressions have values.
type Stmt interface {
	Node
	// Accept implements the visitor pattern for statements.
	Accept(v Visitor) error
	stmtNode() // Marker method
}

// Decl is the interface for all declaration nodes.
//
// WHAT IS A DECLARATION?
// A declaration introduces a new name (variable, function, type, etc.).
// Examples: var x int, func foo() {}, type Point struct {...}
//
// DESIGN CHOICE: Separate from Stmt because:
// - Declarations have special scoping rules
// - Some contexts only allow declarations (e.g., top level of a file)
// - Makes semantic analysis clearer
//
// However, declarations are also statements (you can use them in statement context).
type Decl interface {
	Stmt
	declNode() // Marker method
}

// Visitor is the interface for AST traversal.
//
// THE VISITOR PATTERN:
// Instead of having methods like expr.TypeCheck(), expr.Optimize(), etc.,
// we have one Accept() method that takes a Visitor.
// Different visitors implement different operations.
//
// BENEFITS:
// 1. Separation of concerns: AST structure vs operations on the AST
// 2. Easy to add new operations without modifying AST nodes
// 3. Avoids type switches (which are error-prone and hard to extend)
// 4. Follows the Open/Closed Principle (open for extension, closed for modification)
//
// EXAMPLE:
//   type TypeChecker struct { ... }
//   func (tc *TypeChecker) VisitBinaryExpr(expr *BinaryExpr) (interface{}, error) {
//       leftType, _ := expr.Left.Accept(tc)
//       rightType, _ := expr.Right.Accept(tc)
//       // ... check types
//   }
//
// DESIGN CHOICE: Return (interface{}, error) rather than just error because:
// - Some visitors need to return values (evaluator returns the result)
// - Type checker returns types
// - Code generator returns IR
// - Returning interface{} is flexible (callers can type assert to what they need)
type Visitor interface {
	// Expression visitors
	VisitBinaryExpr(expr *BinaryExpr) (interface{}, error)
	VisitUnaryExpr(expr *UnaryExpr) (interface{}, error)
	VisitLiteralExpr(expr *LiteralExpr) (interface{}, error)
	VisitIdentifierExpr(expr *IdentifierExpr) (interface{}, error)
	VisitCallExpr(expr *CallExpr) (interface{}, error)
	VisitIndexExpr(expr *IndexExpr) (interface{}, error)
	VisitMemberExpr(expr *MemberExpr) (interface{}, error)
	VisitAssignmentExpr(expr *AssignmentExpr) (interface{}, error)
	VisitLogicalExpr(expr *LogicalExpr) (interface{}, error)
	VisitGroupingExpr(expr *GroupingExpr) (interface{}, error)
	VisitArrayLiteralExpr(expr *ArrayLiteralExpr) (interface{}, error)
	VisitStructLiteralExpr(expr *StructLiteralExpr) (interface{}, error)

	// Statement visitors
	VisitExprStmt(stmt *ExprStmt) error
	VisitBlockStmt(stmt *BlockStmt) error
	VisitIfStmt(stmt *IfStmt) error
	VisitWhileStmt(stmt *WhileStmt) error
	VisitForStmt(stmt *ForStmt) error
	VisitReturnStmt(stmt *ReturnStmt) error
	VisitBreakStmt(stmt *BreakStmt) error
	VisitContinueStmt(stmt *ContinueStmt) error
	VisitSwitchStmt(stmt *SwitchStmt) error

	// Declaration visitors
	VisitVarDecl(decl *VarDecl) error
	VisitFuncDecl(decl *FuncDecl) error
	VisitTypeDecl(decl *TypeDecl) error
	VisitStructDecl(decl *StructDecl) error
}

// File represents a single source file.
//
// DESIGN CHOICE: The root of our AST is a File, not a Program, because:
// - We want to support separate compilation (compile files independently)
// - It matches how Go and many other languages work
// - It's easier to parallelize (can parse files concurrently)
//
// A program is just a collection of files (handled at a higher level).
type File struct {
	// Package is the package name (e.g., "main", "fmt")
	Package *PackageDecl

	// Imports are the import declarations
	Imports []*ImportDecl

	// Decls are the top-level declarations (functions, types, variables)
	Decls []Decl

	// Comments contains all comments in the file.
	// We store them separately because they're not part of the syntax tree,
	// but we need them for documentation generation and code formatting.
	Comments []*Comment

	// Filename is the name of the source file
	Filename string
}

// PackageDecl represents a package declaration (package foo).
type PackageDecl struct {
	PackagePos lexer.Position // Position of 'package' keyword
	Name       *IdentifierExpr
}

func (p *PackageDecl) Pos() lexer.Position { return p.PackagePos }
func (p *PackageDecl) End() lexer.Position { return p.Name.End() }

// ImportDecl represents an import declaration (import "fmt" or import foo "bar").
type ImportDecl struct {
	ImportPos lexer.Position // Position of 'import' keyword
	Name      *IdentifierExpr // Optional name (for aliasing)
	Path      *LiteralExpr   // Import path (string literal)
}

func (i *ImportDecl) Pos() lexer.Position { return i.ImportPos }
func (i *ImportDecl) End() lexer.Position { return i.Path.End() }

// Comment represents a comment.
//
// DESIGN CHOICE: Comments are not part of the main AST, but we track them because:
// - Documentation tools need them
// - Code formatters need to preserve them
// - IDEs show them in hover info
type Comment struct {
	Position lexer.Position
	Text     string
	IsBlock  bool // true for /* */ comments, false for // comments
}

func (c *Comment) Pos() lexer.Position { return c.Position }
func (c *Comment) End() lexer.Position {
	// Calculate end position based on comment text
	lines := 0
	lastNewline := -1
	for i, ch := range c.Text {
		if ch == '\n' {
			lines++
			lastNewline = i
		}
	}
	endLine := c.Position.Line + lines
	endCol := c.Position.Column
	if lines > 0 {
		endCol = len(c.Text) - lastNewline
	} else {
		endCol += len(c.Text)
	}
	return lexer.Position{
		Filename: c.Position.Filename,
		Line:     endLine,
		Column:   endCol,
		Offset:   c.Position.Offset + len(c.Text),
	}
}

// BaseNode provides common functionality for AST nodes.
//
// DESIGN CHOICE: We use embedding rather than requiring every node to implement
// Pos/End manually. This is:
// - More DRY (Don't Repeat Yourself)
// - Less error-prone
// - Easier to maintain
//
// However, nodes with complex structure (like BlockStmt) may override these.
type BaseNode struct {
	StartPos lexer.Position
	EndPos   lexer.Position
}

func (b *BaseNode) Pos() lexer.Position { return b.StartPos }
func (b *BaseNode) End() lexer.Position { return b.EndPos }
