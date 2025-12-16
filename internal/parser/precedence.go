package parser

import (
	"github.com/hassan/compiler/internal/lexer"
)

// Precedence represents operator precedence levels.
//
// DESIGN CHOICE: Use integer precedence levels rather than enums because:
// - Easy to compare (higher number = higher precedence)
// - Easy to add new levels between existing ones (use 5, 10, 15, etc.)
// - Matches how most parsing algorithms work
//
// PRECEDENCE RULES (from lowest to highest):
// 1. Assignment (=, +=, -=, etc.)
// 2. Logical OR (||)
// 3. Logical AND (&&)
// 4. Equality (==, !=)
// 5. Comparison (<, <=, >, >=)
// 6. Bitwise OR (|)
// 7. Bitwise XOR (^)
// 8. Bitwise AND (&)
// 9. Shift (<<, >>)
// 10. Addition/Subtraction (+, -)
// 11. Multiplication/Division (*, /, %)
// 12. Exponentiation (**)
// 13. Unary (!, -, ~, ++, --)
// 14. Member access (., [], ())
//
// These match C/C++/Java conventions, which are well-understood by programmers.
type Precedence int

const (
	PrecNone Precedence = iota
	PrecAssignment // =, +=, -=, etc.
	PrecOr         // ||
	PrecAnd        // &&
	PrecEquality   // ==, !=
	PrecComparison // <, <=, >, >=
	PrecBitOr      // |
	PrecBitXor     // ^
	PrecBitAnd     // &
	PrecShift      // <<, >>
	PrecTerm       // +, -
	PrecFactor     // *, /, %
	PrecExponent   // **
	PrecUnary      // !, -, ~, ++, --
	PrecCall       // ., [], ()
	PrecPrimary    // literals, identifiers, grouping
)

// getPrecedence returns the precedence level for a given token type.
//
// DESIGN CHOICE: Function rather than map because:
// - Faster (no map lookup, direct switch with jump table)
// - More maintainable (easier to see all precedences at once)
// - Compile-time checking (typos in token types are caught)
//
// This is used by the Pratt parser to decide when to stop parsing.
func getPrecedence(tokenType lexer.TokenType) Precedence {
	switch tokenType {
	// Assignment operators (lowest precedence)
	case lexer.TokenAssign,
		lexer.TokenPlusEq,
		lexer.TokenMinusEq,
		lexer.TokenStarEq,
		lexer.TokenSlashEq,
		lexer.TokenPercentEq,
		lexer.TokenAndEq,
		lexer.TokenOrEq,
		lexer.TokenXorEq,
		lexer.TokenShlEq,
		lexer.TokenShrEq:
		return PrecAssignment

	// Logical OR
	case lexer.TokenOr:
		return PrecOr

	// Logical AND
	case lexer.TokenAnd:
		return PrecAnd

	// Equality
	case lexer.TokenEqual, lexer.TokenNotEqual:
		return PrecEquality

	// Comparison
	case lexer.TokenLess,
		lexer.TokenLessEqual,
		lexer.TokenGreater,
		lexer.TokenGreaterEqual:
		return PrecComparison

	// Bitwise OR
	case lexer.TokenBitOr:
		return PrecBitOr

	// Bitwise XOR
	case lexer.TokenBitXor:
		return PrecBitXor

	// Bitwise AND
	case lexer.TokenBitAnd:
		return PrecBitAnd

	// Shift
	case lexer.TokenShl, lexer.TokenShr:
		return PrecShift

	// Addition and subtraction
	case lexer.TokenPlus, lexer.TokenMinus:
		return PrecTerm

	// Multiplication, division, modulo
	case lexer.TokenStar, lexer.TokenSlash, lexer.TokenPercent:
		return PrecFactor

	// Exponentiation
	case lexer.TokenStarStar:
		return PrecExponent

	// Member access, indexing, function calls
	case lexer.TokenDot, lexer.TokenLeftBracket, lexer.TokenLeftParen:
		return PrecCall

	default:
		return PrecNone
	}
}

// isRightAssociative returns true if the operator is right-associative.
//
// ASSOCIATIVITY:
// - Left-associative: a + b + c = (a + b) + c
// - Right-associative: a = b = c = (a = (b = c))
//
// Most operators are left-associative. Right-associative operators:
// - Assignment (x = y = 5 means x = (y = 5))
// - Exponentiation (2 ** 3 ** 4 means 2 ** (3 ** 4) in some languages)
//
// DESIGN CHOICE: We make assignment right-associative because:
// - It matches programmer expectations
// - Allows chaining (x = y = z = 0)
// - Consistent with C/Java/Go
//
// DESIGN CHOICE: We make exponentiation right-associative because:
// - It matches mathematical convention (2^3^4 = 2^(3^4) = 2^81)
// - Consistent with Python, Ruby, etc.
func isRightAssociative(tokenType lexer.TokenType) bool {
	switch tokenType {
	case lexer.TokenAssign,
		lexer.TokenPlusEq,
		lexer.TokenMinusEq,
		lexer.TokenStarEq,
		lexer.TokenSlashEq,
		lexer.TokenPercentEq,
		lexer.TokenAndEq,
		lexer.TokenOrEq,
		lexer.TokenXorEq,
		lexer.TokenShlEq,
		lexer.TokenShrEq,
		lexer.TokenStarStar:
		return true
	default:
		return false
	}
}
