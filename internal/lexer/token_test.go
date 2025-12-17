package lexer

import (
	"testing"
)

func TestToken_String(t *testing.T) {
	tests := []struct {
		name     string
		token    Token
		expected string
	}{
		{
			name: "identifier token",
			token: Token{
				Type:     TokenIdentifier,
				Lexeme:   "foo",
				Position: Position{Filename: "test.go", Line: 1, Column: 1},
			},
			expected: "IDENTIFIER(foo) at test.go:1:1",
		},
		{
			name: "number token",
			token: Token{
				Type:     TokenNumber,
				Lexeme:   "42",
				Position: Position{Filename: "test.go", Line: 5, Column: 10},
			},
			expected: "NUMBER(42) at test.go:5:10",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.token.String()
			if result != tt.expected {
				t.Errorf("Token.String() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestToken_Span(t *testing.T) {
	token := Token{
		Type:   TokenIdentifier,
		Lexeme: "hello",
		Position: Position{
			Filename: "test.go",
			Line:     1,
			Column:   5,
			Offset:   4,
		},
		Length: 5,
	}

	span := token.Span()

	if span.Start.Offset != 4 {
		t.Errorf("Span start offset = %d, want 4", span.Start.Offset)
	}

	if span.End.Offset != 9 {
		t.Errorf("Span end offset = %d, want 9", span.End.Offset)
	}

	if span.Start.Line != 1 {
		t.Errorf("Span start line = %d, want 1", span.Start.Line)
	}

	if span.End.Line != 1 {
		t.Errorf("Span end line = %d, want 1", span.End.Line)
	}
}

func TestTokenType_String(t *testing.T) {
	tests := []struct {
		name     string
		tt       TokenType
		expected string
	}{
		{"EOF", TokenEOF, "EOF"},
		{"Invalid", TokenInvalid, "INVALID"},
		{"Number", TokenNumber, "NUMBER"},
		{"String", TokenString, "STRING"},
		{"Identifier", TokenIdentifier, "IDENTIFIER"},
		{"If keyword", TokenIf, "IF"},
		{"Plus operator", TokenPlus, "PLUS"},
		{"Left paren", TokenLeftParen, "LPAREN"},
		{"Unknown type", TokenType(9999), "UNKNOWN"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.tt.String()
			if result != tt.expected {
				t.Errorf("TokenType.String() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestLookupKeyword(t *testing.T) {
	tests := []struct {
		name       string
		identifier string
		expected   TokenType
	}{
		{"if keyword", "if", TokenIf},
		{"else keyword", "else", TokenElse},
		{"for keyword", "for", TokenFor},
		{"while keyword", "while", TokenWhile},
		{"func keyword", "func", TokenFunc},
		{"var keyword", "var", TokenVar},
		{"true keyword", "true", TokenTrue},
		{"false keyword", "false", TokenFalse},
		{"nil keyword", "nil", TokenNil},
		{"not a keyword", "foobar", TokenIdentifier},
		{"case sensitive - If", "If", TokenIdentifier},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := LookupKeyword(tt.identifier)
			if result != tt.expected {
				t.Errorf("LookupKeyword(%q) = %v, want %v", tt.identifier, result, tt.expected)
			}
		})
	}
}

func TestTokenType_IsKeyword(t *testing.T) {
	tests := []struct {
		name     string
		tt       TokenType
		expected bool
	}{
		{"If keyword", TokenIf, true},
		{"Var keyword", TokenVar, true},
		{"Return keyword", TokenReturn, true},
		{"Identifier", TokenIdentifier, false},
		{"Number", TokenNumber, false},
		{"Plus operator", TokenPlus, false},
		{"EOF", TokenEOF, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.tt.IsKeyword()
			if result != tt.expected {
				t.Errorf("TokenType.IsKeyword() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestTokenType_IsOperator(t *testing.T) {
	tests := []struct {
		name     string
		tt       TokenType
		expected bool
	}{
		{"Plus", TokenPlus, true},
		{"Minus", TokenMinus, true},
		{"Star", TokenStar, true},
		{"Equal", TokenEqual, true},
		{"And", TokenAnd, true},
		{"Dot", TokenDot, true},
		{"Identifier", TokenIdentifier, false},
		{"Number", TokenNumber, false},
		{"If keyword", TokenIf, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.tt.IsOperator()
			if result != tt.expected {
				t.Errorf("TokenType.IsOperator() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestTokenType_IsLiteral(t *testing.T) {
	tests := []struct {
		name     string
		tt       TokenType
		expected bool
	}{
		{"Number", TokenNumber, true},
		{"String", TokenString, true},
		{"Char", TokenChar, true},
		{"True", TokenTrue, true},
		{"False", TokenFalse, true},
		{"Nil", TokenNil, true},
		{"Identifier", TokenIdentifier, false},
		{"Plus operator", TokenPlus, false},
		{"If keyword", TokenIf, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.tt.IsLiteral()
			if result != tt.expected {
				t.Errorf("TokenType.IsLiteral() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestRuneCount(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected int
	}{
		{"empty string", "", 0},
		{"ascii", "hello", 5},
		{"unicode", "hello 世界", 8},
		{"emojis", "😀😁😂", 3},
		{"mixed", "abc世", 4},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := runeCount(tt.input)
			if result != tt.expected {
				t.Errorf("runeCount(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}
