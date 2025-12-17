package parser

import (
	"testing"

	"github.com/hassan/compiler/internal/lexer"
)

func TestGetPrecedence(t *testing.T) {
	tests := []struct {
		name     string
		token    lexer.TokenType
		expected Precedence
	}{
		// Assignment (lowest)
		{"assign", lexer.TokenAssign, PrecAssignment},
		{"plus equals", lexer.TokenPlusEq, PrecAssignment},
		{"minus equals", lexer.TokenMinusEq, PrecAssignment},

		// Logical OR
		{"logical or", lexer.TokenOr, PrecOr},

		// Logical AND
		{"logical and", lexer.TokenAnd, PrecAnd},

		// Equality
		{"equal", lexer.TokenEqual, PrecEquality},
		{"not equal", lexer.TokenNotEqual, PrecEquality},

		// Comparison
		{"less than", lexer.TokenLess, PrecComparison},
		{"less equal", lexer.TokenLessEqual, PrecComparison},
		{"greater than", lexer.TokenGreater, PrecComparison},
		{"greater equal", lexer.TokenGreaterEqual, PrecComparison},

		// Bitwise OR
		{"bit or", lexer.TokenBitOr, PrecBitOr},

		// Bitwise XOR
		{"bit xor", lexer.TokenBitXor, PrecBitXor},

		// Bitwise AND
		{"bit and", lexer.TokenBitAnd, PrecBitAnd},

		// Shift
		{"shift left", lexer.TokenShl, PrecShift},
		{"shift right", lexer.TokenShr, PrecShift},

		// Term (addition/subtraction)
		{"plus", lexer.TokenPlus, PrecTerm},
		{"minus", lexer.TokenMinus, PrecTerm},

		// Factor (multiplication/division/modulo)
		{"star", lexer.TokenStar, PrecFactor},
		{"slash", lexer.TokenSlash, PrecFactor},
		{"percent", lexer.TokenPercent, PrecFactor},

		// Exponentiation
		{"star star", lexer.TokenStarStar, PrecExponent},

		// Call (highest)
		{"dot", lexer.TokenDot, PrecCall},
		{"left bracket", lexer.TokenLeftBracket, PrecCall},
		{"left paren", lexer.TokenLeftParen, PrecCall},

		// Non-operators
		{"identifier", lexer.TokenIdentifier, PrecNone},
		{"number", lexer.TokenNumber, PrecNone},
		{"semicolon", lexer.TokenSemicolon, PrecNone},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getPrecedence(tt.token)
			if result != tt.expected {
				t.Errorf("getPrecedence(%v) = %v, want %v", tt.token, result, tt.expected)
			}
		})
	}
}

func TestIsRightAssociative(t *testing.T) {
	tests := []struct {
		name     string
		token    lexer.TokenType
		expected bool
	}{
		// Right-associative
		{"assign", lexer.TokenAssign, true},
		{"plus equals", lexer.TokenPlusEq, true},
		{"minus equals", lexer.TokenMinusEq, true},
		{"star star (exponent)", lexer.TokenStarStar, true},

		// Left-associative
		{"plus", lexer.TokenPlus, false},
		{"minus", lexer.TokenMinus, false},
		{"star", lexer.TokenStar, false},
		{"slash", lexer.TokenSlash, false},
		{"equal", lexer.TokenEqual, false},
		{"and", lexer.TokenAnd, false},
		{"or", lexer.TokenOr, false},
		{"dot", lexer.TokenDot, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isRightAssociative(tt.token)
			if result != tt.expected {
				t.Errorf("isRightAssociative(%v) = %v, want %v", tt.token, result, tt.expected)
			}
		})
	}
}

func TestPrecedenceOrdering(t *testing.T) {
	// Test that precedence increases as expected
	if PrecAssignment >= PrecOr {
		t.Error("Assignment should have lower precedence than OR")
	}
	if PrecOr >= PrecAnd {
		t.Error("OR should have lower precedence than AND")
	}
	if PrecAnd >= PrecEquality {
		t.Error("AND should have lower precedence than Equality")
	}
	if PrecEquality >= PrecComparison {
		t.Error("Equality should have lower precedence than Comparison")
	}
	if PrecComparison >= PrecBitOr {
		t.Error("Comparison should have lower precedence than BitOr")
	}
	if PrecBitOr >= PrecBitXor {
		t.Error("BitOr should have lower precedence than BitXor")
	}
	if PrecBitXor >= PrecBitAnd {
		t.Error("BitXor should have lower precedence than BitAnd")
	}
	if PrecBitAnd >= PrecShift {
		t.Error("BitAnd should have lower precedence than Shift")
	}
	if PrecShift >= PrecTerm {
		t.Error("Shift should have lower precedence than Term")
	}
	if PrecTerm >= PrecFactor {
		t.Error("Term should have lower precedence than Factor")
	}
	if PrecFactor >= PrecExponent {
		t.Error("Factor should have lower precedence than Exponent")
	}
	if PrecExponent >= PrecUnary {
		t.Error("Exponent should have lower precedence than Unary")
	}
	if PrecUnary >= PrecCall {
		t.Error("Unary should have lower precedence than Call")
	}
	if PrecCall >= PrecPrimary {
		t.Error("Call should have lower precedence than Primary")
	}
}
