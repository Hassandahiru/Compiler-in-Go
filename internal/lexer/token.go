package lexer

// TokenType represents the type of a token.
//
// DESIGN CHOICE: We use an int-based enum (via iota) rather than strings because:
// 1. Faster comparisons (integer vs string comparison)
// 2. Less memory (1 int vs string pointer + length + data)
// 3. Type safety (compiler catches typos)
// 4. Easy to add new token types without breaking existing code
//
// The downside is slightly more verbose String() implementation, but that's
// only used for debugging and error messages, not hot paths.
type TokenType int

// Token type enumeration.
//
// ORGANIZATION: Tokens are grouped logically:
// 1. Special tokens (EOF, Invalid, Comment)
// 2. Literals (Number, String, etc.)
// 3. Identifiers and keywords
// 4. Operators (grouped by precedence/category)
// 5. Delimiters
//
// This organization makes it easier to:
// - Add new tokens in the right place
// - Understand token relationships
// - Implement precedence climbing in the parser
const (
	// Special tokens

	// TokenEOF marks the end of the input.
	// DESIGN CHOICE: We use a token instead of nil/error because:
	// - It simplifies parser logic (no special case for end of input)
	// - It has a position (useful for "unexpected end of file" errors)
	// - It's consistent with how most compilers work
	TokenEOF TokenType = iota

	// TokenInvalid represents a lexical error.
	// We use this instead of returning an error immediately because:
	// - It allows the lexer to continue and find more errors
	// - Parser can recover and report multiple errors in one pass
	// - The error details are stored in Token.Lexeme
	TokenInvalid

	// TokenComment represents a comment.
	// We tokenize comments (rather than skipping them) because:
	// - Doc generation tools need them
	// - Code formatters need to preserve them
	// - IDE features (hover, completion) may use them
	// However, the parser typically ignores them.
	TokenComment

	// Literals

	// TokenNumber represents any numeric literal (int or float).
	// DESIGN CHOICE: We use a single token type for all numbers because:
	// - The lexer doesn't need to distinguish int vs float
	// - The parser/semantic analyzer will do type inference
	// - It simplifies the lexer implementation
	//
	// The actual value is stored in Token.Lexeme as a string,
	// and converted to the appropriate type later.
	TokenNumber

	// TokenString represents a string literal.
	// We store the raw string (including quotes) in Token.Lexeme
	// and unescape it during parsing. This is because:
	// - It preserves the original source for error messages
	// - It allows the parser to validate escape sequences
	// - It's simpler for the lexer (just match until closing quote)
	TokenString

	// TokenChar represents a character literal ('a', '\n', etc.)
	TokenChar

	// TokenTrue and TokenFalse are boolean literals.
	// DESIGN CHOICE: Separate tokens vs keyword "true"/"false" because:
	// - Makes parser code clearer (no conversion needed)
	// - Faster to check (no string comparison)
	// - Consistent with how most languages treat booleans
	TokenTrue
	TokenFalse

	// TokenNil represents a nil/null literal
	TokenNil

	// Identifiers and Keywords

	// TokenIdentifier represents a variable/function/type name.
	// The actual name is stored in Token.Lexeme.
	TokenIdentifier

	// Keywords - Control Flow
	// These are ordered alphabetically for easier maintenance

	TokenIf
	TokenElse
	TokenFor
	TokenWhile
	TokenBreak
	TokenContinue
	TokenReturn
	TokenSwitch
	TokenCase
	TokenDefault

	// Keywords - Declarations
	TokenFunc
	TokenVar
	TokenConst
	TokenTypeKeyword
	TokenStruct
	TokenInterface
	TokenImport
	TokenPackage

	// Operators - Arithmetic
	// DESIGN CHOICE: We have separate tokens for each operator rather than
	// a generic "operator" token because:
	// - It makes the parser simpler (switch on token type vs string comparison)
	// - It's more efficient (no string allocations/comparisons)
	// - It makes precedence handling clearer

	TokenPlus     // +
	TokenMinus    // -
	TokenStar     // *
	TokenSlash    // /
	TokenPercent  // %
	TokenStarStar // ** (exponentiation, if we support it)

	// Operators - Comparison
	TokenEqual        // ==
	TokenNotEqual     // !=
	TokenLess         // <
	TokenLessEqual    // <=
	TokenGreater      // >
	TokenGreaterEqual // >=

	// Operators - Logical
	TokenAnd // && (logical AND)
	TokenOr  // || (logical OR)
	TokenNot // ! (logical NOT)

	// Operators - Bitwise
	TokenBitAnd // & (bitwise AND)
	TokenBitOr  // | (bitwise OR)
	TokenBitXor // ^ (bitwise XOR)
	TokenBitNot // ~ (bitwise NOT)
	TokenShl    // << (shift left)
	TokenShr    // >> (shift right)

	// Operators - Assignment
	TokenAssign    // =
	TokenPlusEq    // +=
	TokenMinusEq   // -=
	TokenStarEq    // *=
	TokenSlashEq   // /=
	TokenPercentEq // %=
	TokenAndEq     // &=
	TokenOrEq      // |=
	TokenXorEq     // ^=
	TokenShlEq     // <<=
	TokenShrEq     // >>=

	// Operators - Increment/Decrement
	TokenPlusPlus   // ++
	TokenMinusMinus // --

	// Operators - Other
	TokenDot       // . (member access)
	TokenArrow     // -> (pointer member access or function type)
	TokenQuestion  // ? (ternary operator)
	TokenColon     // : (ternary, labels, type annotations)
	TokenColonColon // :: (scope resolution)

	// Delimiters
	TokenLeftParen    // (
	TokenRightParen   // )
	TokenLeftBrace    // {
	TokenRightBrace   // }
	TokenLeftBracket  // [
	TokenRightBracket // ]
	TokenSemicolon    // ;
	TokenComma        // ,
	TokenEllipsis     // ... (variadic parameters)
)

// Token represents a single lexical token.
//
// DESIGN CHOICE: Token is a value type (not pointer) because:
// 1. Tokens are small and cheap to copy (< 100 bytes)
// 2. No need for sharing/mutation after creation
// 3. Avoids GC pressure (no allocations for token values)
// 4. Can be used as map keys (if needed)
type Token struct {
	// Type is the token type.
	Type TokenType

	// Lexeme is the actual text from the source code.
	// We store this rather than just the token type because:
	// - Identifiers: need the actual name
	// - Numbers: need the actual value
	// - Strings: need the actual content
	// - Errors: need to show what was invalid
	//
	// For tokens where the type is sufficient (keywords, operators),
	// this will be the expected string (e.g., "if", "==").
	Lexeme string

	// Position is where this token appears in the source.
	// This is crucial for error reporting.
	Position Position

	// Length is the length of the token in bytes.
	// We store this rather than computing it from Lexeme because:
	// - Lexeme might be modified (e.g., unescaping strings)
	// - It's useful for highlighting the exact source range
	// - It's cheap to compute during lexing (we know it anyway)
	Length int
}

// String returns a human-readable representation of the token.
// Format: "TYPE(lexeme) at position"
// Example: "IDENTIFIER(foo) at main.go:42:15"
//
// This is primarily for debugging and error messages.
func (t Token) String() string {
	return t.Type.String() + "(" + t.Lexeme + ") at " + t.Position.String()
}

// Span returns the source span covered by this token.
// This is useful for error reporting and IDE features.
func (t Token) Span() Span {
	return Span{
		Start: t.Position,
		End: Position{
			Filename: t.Position.Filename,
			Line:     t.Position.Line,
			Column:   t.Position.Column + runeCount(t.Lexeme),
			Offset:   t.Position.Offset + t.Length,
		},
	}
}

// runeCount returns the number of runes (Unicode code points) in s.
// We use this for column calculation because we count columns in runes, not bytes.
//
// PERFORMANCE: This is O(n) where n is the string length, but:
// - It's only called for debugging/error messages
// - Most tokens are short
// - The alternative (storing both byte length and rune count) wastes memory
func runeCount(s string) int {
	count := 0
	for range s {
		count++
	}
	return count
}

// String returns the string representation of a token type.
//
// DESIGN CHOICE: We implement this manually rather than using a tool like stringer because:
// - It gives us full control over the output format
// - No external dependencies
// - The list is unlikely to change dramatically
//
// We use descriptive names for clarity in error messages.
func (tt TokenType) String() string {
	switch tt {
	case TokenEOF:
		return "EOF"
	case TokenInvalid:
		return "INVALID"
	case TokenComment:
		return "COMMENT"
	case TokenNumber:
		return "NUMBER"
	case TokenString:
		return "STRING"
	case TokenChar:
		return "CHAR"
	case TokenTrue:
		return "TRUE"
	case TokenFalse:
		return "FALSE"
	case TokenNil:
		return "NIL"
	case TokenIdentifier:
		return "IDENTIFIER"
	case TokenIf:
		return "IF"
	case TokenElse:
		return "ELSE"
	case TokenFor:
		return "FOR"
	case TokenWhile:
		return "WHILE"
	case TokenBreak:
		return "BREAK"
	case TokenContinue:
		return "CONTINUE"
	case TokenReturn:
		return "RETURN"
	case TokenSwitch:
		return "SWITCH"
	case TokenCase:
		return "CASE"
	case TokenDefault:
		return "DEFAULT"
	case TokenFunc:
		return "FUNC"
	case TokenVar:
		return "VAR"
	case TokenConst:
		return "CONST"
	case TokenTypeKeyword:
		return "TYPE"
	case TokenStruct:
		return "STRUCT"
	case TokenInterface:
		return "INTERFACE"
	case TokenImport:
		return "IMPORT"
	case TokenPackage:
		return "PACKAGE"
	case TokenPlus:
		return "PLUS"
	case TokenMinus:
		return "MINUS"
	case TokenStar:
		return "STAR"
	case TokenSlash:
		return "SLASH"
	case TokenPercent:
		return "PERCENT"
	case TokenStarStar:
		return "STARSTAR"
	case TokenEqual:
		return "EQUAL"
	case TokenNotEqual:
		return "NOTEQUAL"
	case TokenLess:
		return "LESS"
	case TokenLessEqual:
		return "LESSEQUAL"
	case TokenGreater:
		return "GREATER"
	case TokenGreaterEqual:
		return "GREATEREQUAL"
	case TokenAnd:
		return "AND"
	case TokenOr:
		return "OR"
	case TokenNot:
		return "NOT"
	case TokenBitAnd:
		return "BITAND"
	case TokenBitOr:
		return "BITOR"
	case TokenBitXor:
		return "BITXOR"
	case TokenBitNot:
		return "BITNOT"
	case TokenShl:
		return "SHL"
	case TokenShr:
		return "SHR"
	case TokenAssign:
		return "ASSIGN"
	case TokenPlusEq:
		return "PLUSEQ"
	case TokenMinusEq:
		return "MINUSEQ"
	case TokenStarEq:
		return "STAREQ"
	case TokenSlashEq:
		return "SLASHEQ"
	case TokenPercentEq:
		return "PERCENTEQ"
	case TokenAndEq:
		return "ANDEQ"
	case TokenOrEq:
		return "OREQ"
	case TokenXorEq:
		return "XOREQ"
	case TokenShlEq:
		return "SHLEQ"
	case TokenShrEq:
		return "SHREQ"
	case TokenPlusPlus:
		return "PLUSPLUS"
	case TokenMinusMinus:
		return "MINUSMINUS"
	case TokenDot:
		return "DOT"
	case TokenArrow:
		return "ARROW"
	case TokenQuestion:
		return "QUESTION"
	case TokenColon:
		return "COLON"
	case TokenColonColon:
		return "COLONCOLON"
	case TokenLeftParen:
		return "LPAREN"
	case TokenRightParen:
		return "RPAREN"
	case TokenLeftBrace:
		return "LBRACE"
	case TokenRightBrace:
		return "RBRACE"
	case TokenLeftBracket:
		return "LBRACKET"
	case TokenRightBracket:
		return "RBRACKET"
	case TokenSemicolon:
		return "SEMICOLON"
	case TokenComma:
		return "COMMA"
	case TokenEllipsis:
		return "ELLIPSIS"
	default:
		return "UNKNOWN"
	}
}

// keywords maps keyword strings to their token types.
//
// DESIGN CHOICE: We use a map rather than a long if-else chain because:
// - O(1) lookup vs O(n) linear search
// - Easier to maintain (just add to the map)
// - More readable
//
// The map is initialized once and never modified (effectively const).
// Go doesn't have const maps, but we can use a package-level var that's
// initialized in init() or as a literal.
var keywords = map[string]TokenType{
	"if":        TokenIf,
	"else":      TokenElse,
	"for":       TokenFor,
	"while":     TokenWhile,
	"break":     TokenBreak,
	"continue":  TokenContinue,
	"return":    TokenReturn,
	"switch":    TokenSwitch,
	"case":      TokenCase,
	"default":   TokenDefault,
	"func":      TokenFunc,
	"var":       TokenVar,
	"const":     TokenConst,
	"type":      TokenTypeKeyword,
	"struct":    TokenStruct,
	"interface": TokenInterface,
	"import":    TokenImport,
	"package":   TokenPackage,
	"true":      TokenTrue,
	"false":     TokenFalse,
	"nil":       TokenNil,
}

// LookupKeyword checks if an identifier is actually a keyword.
// Returns the keyword token type if it is, or TokenIdentifier if not.
//
// USAGE: After lexing an identifier, call this to determine if it's a keyword.
//
// DESIGN CHOICE: This is a function rather than exposing the map because:
// - It encapsulates the implementation (could change to a trie, for example)
// - It provides a clear API
// - It prevents accidental modification of the keywords map
func LookupKeyword(identifier string) TokenType {
	if tokenType, ok := keywords[identifier]; ok {
		return tokenType
	}
	return TokenIdentifier
}

// IsKeyword returns true if the token is a keyword.
// This is useful for parser error recovery and syntax highlighting.
func (tt TokenType) IsKeyword() bool {
	return tt >= TokenIf && tt <= TokenPackage
}

// IsOperator returns true if the token is an operator.
func (tt TokenType) IsOperator() bool {
	return tt >= TokenPlus && tt <= TokenColonColon
}

// IsLiteral returns true if the token is a literal value.
func (tt TokenType) IsLiteral() bool {
	return tt >= TokenNumber && tt <= TokenNil
}
