package lexer

import (
	"testing"
)

func TestLexer_Keywords(t *testing.T) {
	source := "if else for while func var return"
	l := New(source, "test.src")

	expectedTypes := []TokenType{
		TokenIf,
		TokenElse,
		TokenFor,
		TokenWhile,
		TokenFunc,
		TokenVar,
		TokenReturn,
		TokenEOF,
	}

	for i, expected := range expectedTypes {
		token, err := l.NextToken()
		if err != nil {
			t.Fatalf("token %d: unexpected error: %v", i, err)
		}
		if token.Type != expected {
			t.Errorf("token %d: expected %v, got %v", i, expected, token.Type)
		}
	}
}

func TestLexer_Identifiers(t *testing.T) {
	source := "foo bar _temp myVar123"
	l := New(source, "test.src")

	expected := []string{"foo", "bar", "_temp", "myVar123"}

	for i, expectedName := range expected {
		token, err := l.NextToken()
		if err != nil {
			t.Fatalf("token %d: unexpected error: %v", i, err)
		}
		if token.Type != TokenIdentifier {
			t.Errorf("token %d: expected TokenIdentifier, got %v", i, token.Type)
		}
		if token.Lexeme != expectedName {
			t.Errorf("token %d: expected %q, got %q", i, expectedName, token.Lexeme)
		}
	}
}

func TestLexer_Numbers(t *testing.T) {
	tests := []struct {
		source string
		want   string
	}{
		{"42", "42"},
		{"3.14", "3.14"},
		{"1e10", "1e10"},
		{"2.5e-3", "2.5e-3"},
	}

	for _, tt := range tests {
		t.Run(tt.source, func(t *testing.T) {
			l := New(tt.source, "test.src")
			token, err := l.NextToken()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if token.Type != TokenNumber {
				t.Errorf("expected TokenNumber, got %v", token.Type)
			}
			if token.Lexeme != tt.want {
				t.Errorf("expected %q, got %q", tt.want, token.Lexeme)
			}
		})
	}
}

func TestLexer_Strings(t *testing.T) {
	source := `"hello" "world\n" "with\"quotes"`
	l := New(source, "test.src")

	expectedLexemes := []string{
		`"hello"`,
		`"world\n"`,
		`"with\"quotes"`,
	}

	for i, expected := range expectedLexemes {
		token, err := l.NextToken()
		if err != nil {
			t.Fatalf("token %d: unexpected error: %v", i, err)
		}
		if token.Type != TokenString {
			t.Errorf("token %d: expected TokenString, got %v", i, token.Type)
		}
		if token.Lexeme != expected {
			t.Errorf("token %d: expected %q, got %q", i, expected, token.Lexeme)
		}
	}
}

func TestLexer_Operators(t *testing.T) {
	source := "+ - * / == != < <= > >= && || ! = +="
	l := New(source, "test.src")

	expectedTypes := []TokenType{
		TokenPlus,
		TokenMinus,
		TokenStar,
		TokenSlash,
		TokenEqual,
		TokenNotEqual,
		TokenLess,
		TokenLessEqual,
		TokenGreater,
		TokenGreaterEqual,
		TokenAnd,
		TokenOr,
		TokenNot,
		TokenAssign,
		TokenPlusEq,
		TokenEOF,
	}

	for i, expected := range expectedTypes {
		token, err := l.NextToken()
		if err != nil {
			t.Fatalf("token %d: unexpected error: %v", i, err)
		}
		if token.Type != expected {
			t.Errorf("token %d: expected %v, got %v", i, expected, token.Type)
		}
	}
}

func TestLexer_Comments(t *testing.T) {
	source := `
// line comment
/* block comment */
/* nested /* comment */ here */
foo
`
	l := New(source, "test.src")

	// Skip comments and find the identifier
	var token Token
	var err error
	for {
		token, err = l.NextToken()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if token.Type != TokenComment {
			break
		}
	}

	if token.Type != TokenIdentifier || token.Lexeme != "foo" {
		t.Errorf("expected identifier 'foo', got %v %q", token.Type, token.Lexeme)
	}
}

func TestLexer_PositionTracking(t *testing.T) {
	source := "foo\nbar"
	l := New(source, "test.src")

	// First token: foo on line 1
	token1, _ := l.NextToken()
	if token1.Position.Line != 1 {
		t.Errorf("expected line 1, got %d", token1.Position.Line)
	}
	if token1.Position.Column != 1 {
		t.Errorf("expected column 1, got %d", token1.Position.Column)
	}

	// Second token: bar on line 2
	token2, _ := l.NextToken()
	if token2.Position.Line != 2 {
		t.Errorf("expected line 2, got %d", token2.Position.Line)
	}
	if token2.Position.Column != 1 {
		t.Errorf("expected column 1, got %d", token2.Position.Column)
	}
}
