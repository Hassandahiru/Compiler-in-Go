// Package parser implements a recursive descent parser for the compiler.
//
// PARSING STRATEGY:
// We use a combination of:
// 1. Recursive Descent for statements and declarations
// 2. Pratt Parsing (precedence climbing) for expressions
//
// WHY RECURSIVE DESCENT?
// - Easy to understand and implement
// - Direct mapping from grammar to code
// - Good error messages (you know exactly what you expected)
// - Efficient (no table lookups or complex data structures)
//
// WHY PRATT PARSING FOR EXPRESSIONS?
// - Elegant handling of operator precedence
// - Easy to extend with new operators
// - Compact code
// - Better than precedence climbing for complex expression grammars
//
// ERROR HANDLING STRATEGY:
// - Report errors but continue parsing (find multiple errors in one pass)
// - Use panic/recover for error recovery at statement boundaries
// - Return errors to caller for fine-grained control
package parser

import (
	"fmt"
	"strconv"
	"unicode/utf8"

	"github.com/hassan/compiler/internal/lexer"
	"github.com/hassan/compiler/internal/parser/ast"
)

// Parser converts a stream of tokens into an Abstract Syntax Tree.
//
// DESIGN CHOICE: Parser is a struct with methods rather than functions because:
// - State management (current token, errors, etc.)
// - Error recovery needs access to parser state
// - Recursive descent naturally fits object-oriented style
type Parser struct {
	// lexer is the source of tokens
	lexer *lexer.Lexer

	// current is the token we're currently examining
	current lexer.Token

	// previous is the last token we consumed (useful for error messages)
	previous lexer.Token

	// errors accumulates all parsing errors
	// DESIGN CHOICE: Accumulate errors rather than stopping at first error because:
	// - Better developer experience (see all errors at once)
	// - Matches what modern compilers do
	// - Doesn't slow down the parser significantly
	errors []error

	// panicMode tracks if we're in panic mode (recovering from an error)
	// During panic mode, we skip tokens until we find a synchronization point
	panicMode bool
}

// New creates a new parser for the given lexer.
func New(l *lexer.Lexer) *Parser {
	p := &Parser{
		lexer:  l,
		errors: make([]error, 0),
	}
	// Prime the parser by reading the first token
	p.advance()
	return p
}

// ParseFile parses a complete source file.
//
// GRAMMAR:
//   file = package imports* decls* EOF
//
// Returns the AST and any errors encountered.
// DESIGN CHOICE: Return both AST and errors (not nil AST on error) because:
// - Partial AST is useful for IDE features even with errors
// - Allows incremental parsing in IDEs
// - Error recovery produces a valid (though incomplete) AST
func (p *Parser) ParseFile(filename string) (*ast.File, []error) {
	file := &ast.File{
		Filename: filename,
		Imports:  make([]*ast.ImportDecl, 0),
		Decls:    make([]ast.Decl, 0),
		Comments: make([]*ast.Comment, 0),
	}

	// Skip any leading comments and collect them
	for p.match(lexer.TokenComment) {
		file.Comments = append(file.Comments, &ast.Comment{
			Position: p.previous.Position,
			Text:     p.previous.Lexeme,
			IsBlock:  p.previous.Lexeme[1] == '*', // /* vs //
		})
	}

	// Parse package declaration (required)
	if p.match(lexer.TokenPackage) {
		file.Package = p.parsePackageDecl()
	} else {
		p.error("expected 'package' declaration at start of file")
	}

	// Parse imports
	for p.match(lexer.TokenImport) {
		file.Imports = append(file.Imports, p.parseImportDecl())
	}

	// Parse top-level declarations
	for !p.isAtEnd() {
		// Skip comments
		if p.match(lexer.TokenComment) {
			file.Comments = append(file.Comments, &ast.Comment{
				Position: p.previous.Position,
				Text:     p.previous.Lexeme,
				IsBlock:  p.previous.Lexeme[1] == '*',
			})
			continue
		}

		decl := p.parseDecl()
		if decl != nil {
			file.Decls = append(file.Decls, decl)
		}
	}

	return file, p.errors
}

// parsePackageDecl parses a package declaration: package name
func (p *Parser) parsePackageDecl() *ast.PackageDecl {
	// We've already consumed the 'package' keyword
	packagePos := p.previous.Position

	if !p.check(lexer.TokenIdentifier) {
		p.error("expected package name")
		return nil
	}

	name := &ast.IdentifierExpr{
		Token: p.current,
		Name:  p.current.Lexeme,
	}
	p.advance()

	return &ast.PackageDecl{
		PackagePos: packagePos,
		Name:       name,
	}
}

// parseImportDecl parses an import declaration:
//   import "path"
//   import alias "path"
func (p *Parser) parseImportDecl() *ast.ImportDecl {
	// We've already consumed the 'import' keyword
	importPos := p.previous.Position

	var name *ast.IdentifierExpr

	// Check for optional alias
	if p.check(lexer.TokenIdentifier) {
		name = &ast.IdentifierExpr{
			Token: p.current,
			Name:  p.current.Lexeme,
		}
		p.advance()
	}

	// Expect string path
	if !p.check(lexer.TokenString) {
		p.error("expected import path (string)")
		return nil
	}

	path := &ast.LiteralExpr{
		Token: p.current,
		Value: p.parseStringLiteral(p.current.Lexeme),
	}
	p.advance()

	return &ast.ImportDecl{
		ImportPos: importPos,
		Name:      name,
		Path:      path,
	}
}

// parseDecl parses a top-level declaration.
//
// GRAMMAR:
//   decl = varDecl | funcDecl | typeDecl | structDecl
func (p *Parser) parseDecl() ast.Decl {
	// Use panic/recover for error recovery
	// If we panic during parsing, we'll recover at this level
	defer func() {
		if r := recover(); r != nil {
			// We panicked - synchronize to the next statement
			p.synchronize()
		}
	}()

	switch {
	case p.match(lexer.TokenVar):
		return p.parseVarDecl()
	case p.match(lexer.TokenFunc):
		return p.parseFuncDecl()
	case p.match(lexer.TokenTypeKeyword):
		return p.parseTypeDecl()
	case p.match(lexer.TokenStruct):
		return p.parseStructDecl()
	default:
		p.error(fmt.Sprintf("expected declaration, got %s", p.current.Type))
		panic("invalid declaration")
	}
}

// parseVarDecl parses a variable declaration:
//   var name type
//   var name type = value
//   var name = value (type inferred)
//   var name1, name2, name3 type
func (p *Parser) parseVarDecl() *ast.VarDecl {
	// We've already consumed 'var'
	varPos := p.previous.Position

	// Parse variable names (can be multiple: var x, y, z int)
	names := make([]*ast.IdentifierExpr, 0)
	for {
		if !p.check(lexer.TokenIdentifier) {
			p.error("expected variable name")
			panic("invalid variable declaration")
		}

		names = append(names, &ast.IdentifierExpr{
			Token: p.current,
			Name:  p.current.Lexeme,
		})
		p.advance()

		if !p.match(lexer.TokenComma) {
			break
		}
	}

	var typeExpr ast.Expr
	var initializer ast.Expr

	// Parse optional type annotation
	if !p.check(lexer.TokenAssign) && !p.check(lexer.TokenSemicolon) {
		typeExpr = p.parseType()
	}

	// Parse optional initializer
	if p.match(lexer.TokenAssign) {
		initializer = p.parseExpression()
	}

	// Validate: must have either type or initializer
	if typeExpr == nil && initializer == nil {
		p.error("variable declaration must have either type or initializer")
	}

	// Expect semicolon
	p.consume(lexer.TokenSemicolon, "expected ';' after variable declaration")

	return &ast.VarDecl{
		VarPos:      varPos,
		Names:       names,
		Type:        typeExpr,
		Initializer: initializer,
	}
}

// parseFuncDecl parses a function declaration:
//   func name(params) returnType { body }
//   func name(params) { body } (void function)
func (p *Parser) parseFuncDecl() *ast.FuncDecl {
	// We've already consumed 'func'
	funcPos := p.previous.Position

	// Parse function name
	if !p.check(lexer.TokenIdentifier) {
		p.error("expected function name")
		panic("invalid function declaration")
	}

	name := &ast.IdentifierExpr{
		Token: p.current,
		Name:  p.current.Lexeme,
	}
	p.advance()

	// Parse parameters
	p.consume(lexer.TokenLeftParen, "expected '(' after function name")
	params := p.parseParameters()
	p.consume(lexer.TokenRightParen, "expected ')' after parameters")

	// Parse optional return type
	var returnType ast.Expr
	if !p.check(lexer.TokenLeftBrace) {
		returnType = p.parseType()
	}

	// Parse body
	var body *ast.BlockStmt
	if p.check(lexer.TokenLeftBrace) {
		body = p.parseBlockStmt()
	} else {
		p.error("expected function body")
	}

	return &ast.FuncDecl{
		FuncPos:    funcPos,
		Name:       name,
		Params:     params,
		ReturnType: returnType,
		Body:       body,
	}
}

// parseParameters parses function parameters: name type, name type, ...
func (p *Parser) parseParameters() []*ast.Parameter {
	params := make([]*ast.Parameter, 0)

	if p.check(lexer.TokenRightParen) {
		// No parameters
		return params
	}

	for {
		if !p.check(lexer.TokenIdentifier) {
			p.error("expected parameter name")
			break
		}

		name := &ast.IdentifierExpr{
			Token: p.current,
			Name:  p.current.Lexeme,
		}
		p.advance()

		typeExpr := p.parseType()

		params = append(params, &ast.Parameter{
			Name: name,
			Type: typeExpr,
		})

		if !p.match(lexer.TokenComma) {
			break
		}
	}

	return params
}

// parseTypeDecl parses a type alias declaration: type Name = Type
func (p *Parser) parseTypeDecl() *ast.TypeDecl {
	// We've already consumed 'type'
	typePos := p.previous.Position

	// Parse type name
	if !p.check(lexer.TokenIdentifier) {
		p.error("expected type name")
		panic("invalid type declaration")
	}

	name := &ast.IdentifierExpr{
		Token: p.current,
		Name:  p.current.Lexeme,
	}
	p.advance()

	// Expect '='
	p.consume(lexer.TokenAssign, "expected '=' in type declaration")

	// Parse the type
	typeExpr := p.parseType()

	p.consume(lexer.TokenSemicolon, "expected ';' after type declaration")

	return &ast.TypeDecl{
		TypePos: typePos,
		Name:    name,
		Type:    typeExpr,
	}
}

// parseStructDecl parses a struct declaration:
//   struct Name { fields }
func (p *Parser) parseStructDecl() *ast.StructDecl {
	// We've already consumed 'struct'
	structPos := p.previous.Position

	// Parse struct name
	if !p.check(lexer.TokenIdentifier) {
		p.error("expected struct name")
		panic("invalid struct declaration")
	}

	name := &ast.IdentifierExpr{
		Token: p.current,
		Name:  p.current.Lexeme,
	}
	p.advance()

	// Parse fields
	p.consume(lexer.TokenLeftBrace, "expected '{' before struct body")
	leftBrace := p.previous

	fields := make([]*ast.FieldDecl, 0)
	for !p.check(lexer.TokenRightBrace) && !p.isAtEnd() {
		// Parse field name
		if !p.check(lexer.TokenIdentifier) {
			p.error("expected field name")
			break
		}

		fieldName := &ast.IdentifierExpr{
			Token: p.current,
			Name:  p.current.Lexeme,
		}
		p.advance()

		// Parse field type
		fieldType := p.parseType()

		fields = append(fields, &ast.FieldDecl{
			Name: fieldName,
			Type: fieldType,
		})

		// Expect semicolon after each field
		p.consume(lexer.TokenSemicolon, "expected ';' after field declaration")
	}

	p.consume(lexer.TokenRightBrace, "expected '}' after struct body")
	rightBrace := p.previous

	return &ast.StructDecl{
		StructPos:  structPos,
		Name:       name,
		LeftBrace:  leftBrace,
		Fields:     fields,
		RightBrace: rightBrace,
	}
}

// parseType parses a type expression.
//
// For now, we just parse identifiers as types.
// Later, we can extend this to support:
// - Array types: []int, [10]int
// - Pointer types: *int
// - Function types: func(int) int
// - Map types: map[string]int
func (p *Parser) parseType() ast.Expr {
	// For now, just parse identifier types
	if !p.check(lexer.TokenIdentifier) {
		p.error("expected type name")
		return nil
	}

	typeExpr := &ast.IdentifierExpr{
		Token: p.current,
		Name:  p.current.Lexeme,
	}
	p.advance()

	return typeExpr
}

// parseStmt parses a statement.
//
// GRAMMAR:
//   stmt = exprStmt | blockStmt | ifStmt | whileStmt | forStmt
//        | returnStmt | breakStmt | continueStmt | switchStmt
//        | varDecl
func (p *Parser) parseStmt() ast.Stmt {
	// Use panic/recover for error recovery
	defer func() {
		if r := recover(); r != nil {
			p.synchronize()
		}
	}()

	switch {
	case p.check(lexer.TokenLeftBrace):
		return p.parseBlockStmt()
	case p.match(lexer.TokenIf):
		return p.parseIfStmt()
	case p.match(lexer.TokenWhile):
		return p.parseWhileStmt()
	case p.match(lexer.TokenFor):
		return p.parseForStmt()
	case p.match(lexer.TokenReturn):
		return p.parseReturnStmt()
	case p.match(lexer.TokenBreak):
		return p.parseBreakStmt()
	case p.match(lexer.TokenContinue):
		return p.parseContinueStmt()
	case p.match(lexer.TokenSwitch):
		return p.parseSwitchStmt()
	case p.match(lexer.TokenVar):
		return p.parseVarDecl()
	default:
		return p.parseExprStmt()
	}
}

// parseBlockStmt parses a block statement: { stmt* }
func (p *Parser) parseBlockStmt() *ast.BlockStmt {
	p.consume(lexer.TokenLeftBrace, "expected '{'")
	leftBrace := p.previous

	statements := make([]ast.Stmt, 0)
	for !p.check(lexer.TokenRightBrace) && !p.isAtEnd() {
		statements = append(statements, p.parseStmt())
	}

	p.consume(lexer.TokenRightBrace, "expected '}'")
	rightBrace := p.previous

	return &ast.BlockStmt{
		LeftBrace:  leftBrace,
		Statements: statements,
		RightBrace: rightBrace,
	}
}

// parseIfStmt parses an if statement:
//   if (condition) { ... }
//   if (condition) { ... } else { ... }
//   if (condition) { ... } else if (condition) { ... }
func (p *Parser) parseIfStmt() *ast.IfStmt {
	// We've already consumed 'if'
	ifPos := p.previous.Position

	// Parse condition
	p.consume(lexer.TokenLeftParen, "expected '(' after 'if'")
	condition := p.parseExpression()
	p.consume(lexer.TokenRightParen, "expected ')' after condition")

	// Parse then branch
	thenBranch := p.parseBlockStmt()

	// Parse optional else branch
	var elseBranch ast.Stmt
	if p.match(lexer.TokenElse) {
		if p.check(lexer.TokenIf) {
			// else if - parse as another if statement
			p.advance()
			elseBranch = p.parseIfStmt()
		} else {
			// else - parse block
			elseBranch = p.parseBlockStmt()
		}
	}

	return &ast.IfStmt{
		IfPos:      ifPos,
		Condition:  condition,
		ThenBranch: thenBranch,
		ElseBranch: elseBranch,
	}
}

// parseWhileStmt parses a while statement: while (condition) { ... }
func (p *Parser) parseWhileStmt() *ast.WhileStmt {
	// We've already consumed 'while'
	whilePos := p.previous.Position

	p.consume(lexer.TokenLeftParen, "expected '(' after 'while'")
	condition := p.parseExpression()
	p.consume(lexer.TokenRightParen, "expected ')' after condition")

	body := p.parseBlockStmt()

	return &ast.WhileStmt{
		WhilePos:  whilePos,
		Condition: condition,
		Body:      body,
	}
}

// parseForStmt parses a for statement:
//   for (init; condition; post) { ... }
func (p *Parser) parseForStmt() *ast.ForStmt {
	// We've already consumed 'for'
	forPos := p.previous.Position

	p.consume(lexer.TokenLeftParen, "expected '(' after 'for'")

	// Parse init (optional)
	var init ast.Stmt
	if p.match(lexer.TokenSemicolon) {
		// No init
	} else if p.match(lexer.TokenVar) {
		init = p.parseVarDecl()
		// VarDecl already consumes its semicolon
	} else {
		init = p.parseExprStmt()
		// ExprStmt will consume its semicolon
	}

	// Parse condition (optional)
	var condition ast.Expr
	if !p.check(lexer.TokenSemicolon) {
		condition = p.parseExpression()
	}
	p.consume(lexer.TokenSemicolon, "expected ';' after loop condition")

	// Parse post (optional)
	var post ast.Stmt
	if !p.check(lexer.TokenRightParen) {
		post = &ast.ExprStmt{Expression: p.parseExpression()}
	}

	p.consume(lexer.TokenRightParen, "expected ')' after for clauses")

	body := p.parseBlockStmt()

	return &ast.ForStmt{
		ForPos:    forPos,
		Init:      init,
		Condition: condition,
		Post:      post,
		Body:      body,
	}
}

// parseReturnStmt parses a return statement: return expr;
func (p *Parser) parseReturnStmt() *ast.ReturnStmt {
	// We've already consumed 'return'
	returnPos := p.previous.Position

	var value ast.Expr
	if !p.check(lexer.TokenSemicolon) {
		value = p.parseExpression()
	}

	p.consume(lexer.TokenSemicolon, "expected ';' after return statement")

	return &ast.ReturnStmt{
		ReturnPos: returnPos,
		Value:     value,
	}
}

// parseBreakStmt parses a break statement: break;
func (p *Parser) parseBreakStmt() *ast.BreakStmt {
	// We've already consumed 'break'
	breakPos := p.previous.Position

	p.consume(lexer.TokenSemicolon, "expected ';' after 'break'")

	return &ast.BreakStmt{
		BreakPos: breakPos,
	}
}

// parseContinueStmt parses a continue statement: continue;
func (p *Parser) parseContinueStmt() *ast.ContinueStmt {
	// We've already consumed 'continue'
	continuePos := p.previous.Position

	p.consume(lexer.TokenSemicolon, "expected ';' after 'continue'")

	return &ast.ContinueStmt{
		ContinuePos: continuePos,
	}
}

// parseSwitchStmt parses a switch statement:
//   switch (expr) {
//     case value: stmts
//     default: stmts
//   }
func (p *Parser) parseSwitchStmt() *ast.SwitchStmt {
	// We've already consumed 'switch'
	switchPos := p.previous.Position

	p.consume(lexer.TokenLeftParen, "expected '(' after 'switch'")
	value := p.parseExpression()
	p.consume(lexer.TokenRightParen, "expected ')' after switch value")

	p.consume(lexer.TokenLeftBrace, "expected '{' before switch body")

	cases := make([]*ast.CaseClause, 0)
	for !p.check(lexer.TokenRightBrace) && !p.isAtEnd() {
		cases = append(cases, p.parseCaseClause())
	}

	p.consume(lexer.TokenRightBrace, "expected '}' after switch body")

	return &ast.SwitchStmt{
		SwitchPos: switchPos,
		Value:     value,
		Cases:     cases,
	}
}

// parseCaseClause parses a case clause:
//   case value1, value2: stmts
//   default: stmts
func (p *Parser) parseCaseClause() *ast.CaseClause {
	var casePos lexer.Position
	var values []ast.Expr
	isDefault := false

	if p.match(lexer.TokenCase) {
		casePos = p.previous.Position

		// Parse case values (can be multiple)
		for {
			values = append(values, p.parseExpression())
			if !p.match(lexer.TokenComma) {
				break
			}
		}
	} else if p.match(lexer.TokenDefault) {
		casePos = p.previous.Position
		isDefault = true
	} else {
		p.error("expected 'case' or 'default'")
		return nil
	}

	p.consume(lexer.TokenColon, "expected ':' after case")
	colon := p.previous

	// Parse statements until next case or end of switch
	body := make([]ast.Stmt, 0)
	for !p.check(lexer.TokenCase) && !p.check(lexer.TokenDefault) &&
		!p.check(lexer.TokenRightBrace) && !p.isAtEnd() {
		body = append(body, p.parseStmt())
	}

	return &ast.CaseClause{
		CasePos:   casePos,
		Values:    values,
		Colon:     colon,
		Body:      body,
		IsDefault: isDefault,
	}
}

// parseExprStmt parses an expression statement: expr;
func (p *Parser) parseExprStmt() *ast.ExprStmt {
	expr := p.parseExpression()
	p.consume(lexer.TokenSemicolon, "expected ';' after expression")
	return &ast.ExprStmt{Expression: expr}
}

// Expression parsing using Pratt parsing (precedence climbing)
//
// PRATT PARSING:
// Instead of recursive descent for expressions (which struggles with precedence),
// we use Pratt parsing. The key idea:
// - Each operator has a precedence level
// - Parse with minimum precedence, climbing up as needed
// - Handles left/right associativity elegantly
//
// REFERENCE: "Top Down Operator Precedence" by Vaughan Pratt (1973)

// parseExpression parses an expression with any precedence.
func (p *Parser) parseExpression() ast.Expr {
	return p.parsePrecedence(PrecAssignment)
}

// parsePrecedence parses an expression with at least the given precedence.
//
// This is the core of Pratt parsing.
func (p *Parser) parsePrecedence(precedence Precedence) ast.Expr {
	// Parse prefix expression
	left := p.parsePrefix()
	if left == nil {
		p.error(fmt.Sprintf("expected expression, got %s", p.current.Type))
		return nil
	}

	// Parse infix expressions with sufficient precedence
	for precedence <= getPrecedence(p.current.Type) {
		left = p.parseInfix(left)
	}

	return left
}

// parsePrefix parses a prefix expression (one that starts an expression).
//
// PREFIX EXPRESSIONS:
// - Literals: 42, "hello", true
// - Identifiers: foo, bar
// - Unary operators: -x, !flag, ++i
// - Grouping: (expr)
// - Array literals: [1, 2, 3]
// - Struct literals: Point{x: 1, y: 2}
func (p *Parser) parsePrefix() ast.Expr {
	switch p.current.Type {
	// Literals
	case lexer.TokenNumber:
		return p.parseNumberLiteral()
	case lexer.TokenString:
		return p.parseStringLiteralExpr()
	case lexer.TokenChar:
		return p.parseCharLiteral()
	case lexer.TokenTrue, lexer.TokenFalse:
		return p.parseBoolLiteral()
	case lexer.TokenNil:
		return p.parseNilLiteral()

	// Identifier
	case lexer.TokenIdentifier:
		return p.parseIdentifier()

	// Grouping
	case lexer.TokenLeftParen:
		return p.parseGrouping()

	// Array literal
	case lexer.TokenLeftBracket:
		return p.parseArrayLiteral()

	// Unary operators
	case lexer.TokenMinus, lexer.TokenNot, lexer.TokenBitNot,
		lexer.TokenPlusPlus, lexer.TokenMinusMinus:
		return p.parseUnary()

	default:
		return nil
	}
}

// parseInfix parses an infix expression (operator that appears between operands).
//
// INFIX EXPRESSIONS:
// - Binary operators: +, -, *, /, etc.
// - Logical operators: &&, ||
// - Comparison operators: ==, !=, <, >, etc.
// - Assignment: =, +=, -=, etc.
// - Member access: obj.field
// - Function call: func(args)
// - Array indexing: arr[index]
// - Postfix operators: i++, i--
func (p *Parser) parseInfix(left ast.Expr) ast.Expr {
	switch p.current.Type {
	// Binary operators
	case lexer.TokenPlus, lexer.TokenMinus, lexer.TokenStar, lexer.TokenSlash,
		lexer.TokenPercent, lexer.TokenStarStar,
		lexer.TokenEqual, lexer.TokenNotEqual,
		lexer.TokenLess, lexer.TokenLessEqual,
		lexer.TokenGreater, lexer.TokenGreaterEqual,
		lexer.TokenBitAnd, lexer.TokenBitOr, lexer.TokenBitXor,
		lexer.TokenShl, lexer.TokenShr:
		return p.parseBinary(left)

	// Logical operators (short-circuit)
	case lexer.TokenAnd, lexer.TokenOr:
		return p.parseLogical(left)

	// Assignment operators
	case lexer.TokenAssign, lexer.TokenPlusEq, lexer.TokenMinusEq,
		lexer.TokenStarEq, lexer.TokenSlashEq, lexer.TokenPercentEq,
		lexer.TokenAndEq, lexer.TokenOrEq, lexer.TokenXorEq,
		lexer.TokenShlEq, lexer.TokenShrEq:
		return p.parseAssignment(left)

	// Member access
	case lexer.TokenDot:
		return p.parseMember(left)

	// Function call
	case lexer.TokenLeftParen:
		return p.parseCall(left)

	// Array indexing
	case lexer.TokenLeftBracket:
		return p.parseIndex(left)

	// Postfix operators
	case lexer.TokenPlusPlus, lexer.TokenMinusMinus:
		// Check if this is really postfix (no space before it)
		// For simplicity, we'll always treat ++ and -- after an expression as postfix
		operator := p.current
		p.advance()
		return &ast.UnaryExpr{
			Operator:  operator,
			Operand:   left,
			IsPostfix: true,
		}

	default:
		return left
	}
}

// Literal parsing

func (p *Parser) parseNumberLiteral() ast.Expr {
	token := p.current
	p.advance()

	// Try to parse as integer first
	if value, err := strconv.ParseInt(token.Lexeme, 0, 64); err == nil {
		return &ast.LiteralExpr{
			Token: token,
			Value: value,
		}
	}

	// Parse as float
	value, err := strconv.ParseFloat(token.Lexeme, 64)
	if err != nil {
		p.error(fmt.Sprintf("invalid number literal: %s", token.Lexeme))
		return &ast.LiteralExpr{Token: token, Value: 0.0}
	}

	return &ast.LiteralExpr{
		Token: token,
		Value: value,
	}
}

func (p *Parser) parseStringLiteralExpr() ast.Expr {
	token := p.current
	p.advance()
	return &ast.LiteralExpr{
		Token: token,
		Value: p.parseStringLiteral(token.Lexeme),
	}
}

func (p *Parser) parseStringLiteral(lexeme string) string {
	// Remove quotes and unescape
	if len(lexeme) < 2 {
		return ""
	}
	// Remove surrounding quotes
	s := lexeme[1 : len(lexeme)-1]

	// Simple unescaping (could be more sophisticated)
	// For now, just handle common escapes
	result := ""
	for i := 0; i < len(s); i++ {
		if s[i] == '\\' && i+1 < len(s) {
			switch s[i+1] {
			case 'n':
				result += "\n"
			case 't':
				result += "\t"
			case 'r':
				result += "\r"
			case '\\':
				result += "\\"
			case '"':
				result += "\""
			default:
				result += string(s[i+1])
			}
			i++ // Skip next character
		} else {
			result += string(s[i])
		}
	}
	return result
}

func (p *Parser) parseCharLiteral() ast.Expr {
	token := p.current
	p.advance()

	// Remove quotes and get the character
	if len(token.Lexeme) < 3 {
		p.error("invalid character literal")
		return &ast.LiteralExpr{Token: token, Value: rune(0)}
	}

	s := token.Lexeme[1 : len(token.Lexeme)-1]
	if s[0] == '\\' {
		// Escape sequence
		if len(s) < 2 {
			p.error("invalid escape sequence")
			return &ast.LiteralExpr{Token: token, Value: rune(0)}
		}
		switch s[1] {
		case 'n':
			return &ast.LiteralExpr{Token: token, Value: '\n'}
		case 't':
			return &ast.LiteralExpr{Token: token, Value: '\t'}
		case 'r':
			return &ast.LiteralExpr{Token: token, Value: '\r'}
		case '\\':
			return &ast.LiteralExpr{Token: token, Value: '\\'}
		case '\'':
			return &ast.LiteralExpr{Token: token, Value: '\''}
		default:
			return &ast.LiteralExpr{Token: token, Value: rune(s[1])}
		}
	}

	// Regular character
	ch, _ := utf8.DecodeRuneInString(s)
	return &ast.LiteralExpr{Token: token, Value: ch}
}

func (p *Parser) parseBoolLiteral() ast.Expr {
	token := p.current
	p.advance()
	return &ast.LiteralExpr{
		Token: token,
		Value: token.Type == lexer.TokenTrue,
	}
}

func (p *Parser) parseNilLiteral() ast.Expr {
	token := p.current
	p.advance()
	return &ast.LiteralExpr{
		Token: token,
		Value: nil,
	}
}

func (p *Parser) parseIdentifier() ast.Expr {
	token := p.current
	p.advance()

	// Check if this is a struct literal: TypeName{...}
	if p.check(lexer.TokenLeftBrace) {
		return p.parseStructLiteral(&ast.IdentifierExpr{
			Token: token,
			Name:  token.Lexeme,
		})
	}

	return &ast.IdentifierExpr{
		Token: token,
		Name:  token.Lexeme,
	}
}

func (p *Parser) parseGrouping() ast.Expr {
	leftParen := p.current
	p.advance()

	expr := p.parseExpression()

	p.consume(lexer.TokenRightParen, "expected ')' after expression")
	rightParen := p.previous

	return &ast.GroupingExpr{
		LeftParen:  leftParen,
		Expression: expr,
		RightParen: rightParen,
	}
}

func (p *Parser) parseArrayLiteral() ast.Expr {
	leftBracket := p.current
	p.advance()

	elements := make([]ast.Expr, 0)

	// Parse elements
	if !p.check(lexer.TokenRightBracket) {
		for {
			elements = append(elements, p.parseExpression())
			if !p.match(lexer.TokenComma) {
				break
			}
		}
	}

	p.consume(lexer.TokenRightBracket, "expected ']' after array elements")
	// For now, use right bracket as right brace (we'd need to adjust the AST)
	rightBrace := p.previous

	return &ast.ArrayLiteralExpr{
		LeftBracket: leftBracket,
		Elements:    elements,
		RightBrace:  rightBrace,
	}
}

func (p *Parser) parseStructLiteral(typeName *ast.IdentifierExpr) ast.Expr {
	leftBrace := p.current
	p.consume(lexer.TokenLeftBrace, "expected '{'")

	fields := make([]*ast.FieldInit, 0)

	if !p.check(lexer.TokenRightBrace) {
		for {
			// Parse field name
			if !p.check(lexer.TokenIdentifier) {
				p.error("expected field name")
				break
			}
			fieldName := &ast.IdentifierExpr{
				Token: p.current,
				Name:  p.current.Lexeme,
			}
			p.advance()

			p.consume(lexer.TokenColon, "expected ':' after field name")
			colon := p.previous

			// Parse field value
			value := p.parseExpression()

			fields = append(fields, &ast.FieldInit{
				Name:  fieldName,
				Colon: colon,
				Value: value,
			})

			if !p.match(lexer.TokenComma) {
				break
			}
		}
	}

	p.consume(lexer.TokenRightBrace, "expected '}' after struct fields")
	rightBrace := p.previous

	return &ast.StructLiteralExpr{
		TypeName:   typeName,
		LeftBrace:  leftBrace,
		Fields:     fields,
		RightBrace: rightBrace,
	}
}

// Operator parsing

func (p *Parser) parseUnary() ast.Expr {
	operator := p.current
	p.advance()

	operand := p.parsePrecedence(PrecUnary)

	return &ast.UnaryExpr{
		Operator:  operator,
		Operand:   operand,
		IsPostfix: false,
	}
}

func (p *Parser) parseBinary(left ast.Expr) ast.Expr {
	operator := p.current
	precedence := getPrecedence(operator.Type)
	p.advance()

	// Adjust precedence for right-associative operators
	if isRightAssociative(operator.Type) {
		precedence--
	}

	right := p.parsePrecedence(precedence + 1)

	return &ast.BinaryExpr{
		Left:     left,
		Operator: operator,
		Right:    right,
	}
}

func (p *Parser) parseLogical(left ast.Expr) ast.Expr {
	operator := p.current
	precedence := getPrecedence(operator.Type)
	p.advance()

	right := p.parsePrecedence(precedence + 1)

	return &ast.LogicalExpr{
		Left:     left,
		Operator: operator,
		Right:    right,
	}
}

func (p *Parser) parseAssignment(left ast.Expr) ast.Expr {
	operator := p.current
	p.advance()

	// Assignment is right-associative
	right := p.parsePrecedence(PrecAssignment)

	return &ast.AssignmentExpr{
		Target:   left,
		Operator: operator,
		Value:    right,
	}
}

func (p *Parser) parseMember(left ast.Expr) ast.Expr {
	dot := p.current
	p.advance()

	if !p.check(lexer.TokenIdentifier) {
		p.error("expected property name after '.'")
		return left
	}

	member := &ast.IdentifierExpr{
		Token: p.current,
		Name:  p.current.Lexeme,
	}
	p.advance()

	return &ast.MemberExpr{
		Object: left,
		Dot:    dot,
		Member: member,
	}
}

func (p *Parser) parseCall(left ast.Expr) ast.Expr {
	leftParen := p.current
	p.advance()

	args := make([]ast.Expr, 0)
	if !p.check(lexer.TokenRightParen) {
		for {
			args = append(args, p.parseExpression())
			if !p.match(lexer.TokenComma) {
				break
			}
		}
	}

	p.consume(lexer.TokenRightParen, "expected ')' after arguments")
	rightParen := p.previous

	return &ast.CallExpr{
		Callee:     left,
		LeftParen:  leftParen,
		Args:       args,
		RightParen: rightParen,
	}
}

func (p *Parser) parseIndex(left ast.Expr) ast.Expr {
	leftBracket := p.current
	p.advance()

	index := p.parseExpression()

	p.consume(lexer.TokenRightBracket, "expected ']' after index")
	rightBracket := p.previous

	return &ast.IndexExpr{
		Object:       left,
		LeftBracket:  leftBracket,
		Index:        index,
		RightBracket: rightBracket,
	}
}

// Helper methods

func (p *Parser) advance() {
	p.previous = p.current
	token, err := p.lexer.NextToken()
	if err != nil {
		p.error(err.Error())
		p.current = lexer.Token{Type: lexer.TokenInvalid}
	} else {
		p.current = token
	}
}

func (p *Parser) check(tokenType lexer.TokenType) bool {
	return p.current.Type == tokenType
}

func (p *Parser) match(tokenTypes ...lexer.TokenType) bool {
	for _, tokenType := range tokenTypes {
		if p.check(tokenType) {
			p.advance()
			return true
		}
	}
	return false
}

func (p *Parser) consume(tokenType lexer.TokenType, message string) {
	if p.check(tokenType) {
		p.advance()
		return
	}
	p.error(message)
	panic(message)
}

func (p *Parser) isAtEnd() bool {
	return p.current.Type == lexer.TokenEOF
}

func (p *Parser) error(message string) {
	if p.panicMode {
		return
	}
	p.panicMode = true
	err := fmt.Errorf("%s: %s", p.current.Position.String(), message)
	p.errors = append(p.errors, err)
}

// synchronize skips tokens until we reach a statement boundary.
// This is used for error recovery.
func (p *Parser) synchronize() {
	p.panicMode = false

	for !p.isAtEnd() {
		// Semicolon marks the end of a statement
		if p.previous.Type == lexer.TokenSemicolon {
			return
		}

		// These tokens start new statements
		switch p.current.Type {
		case lexer.TokenFunc, lexer.TokenVar, lexer.TokenFor,
			lexer.TokenIf, lexer.TokenWhile, lexer.TokenReturn,
			lexer.TokenStruct, lexer.TokenTypeKeyword:
			return
		}

		p.advance()
	}
}
