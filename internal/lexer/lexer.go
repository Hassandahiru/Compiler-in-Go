package lexer

import (
	"fmt"
	"unicode"
	"unicode/utf8"
)

// Lexer performs lexical analysis on source code, converting it into a stream of tokens.
//
// DESIGN PHILOSOPHY:
// The lexer is the first phase of compilation. Its responsibilities are:
// 1. Break source into tokens (lexical analysis)
// 2. Track position information for error reporting
// 3. Handle whitespace and comments appropriately
// 4. Recognize keywords, identifiers, literals, and operators
// 5. Report lexical errors clearly
//
// The lexer does NOT:
// - Parse syntax (that's the parser's job)
// - Perform type checking (semantic analyzer's job)
// - Evaluate expressions (runtime's job)
//
// DESIGN CHOICE: We use a struct with methods rather than a functional approach because:
// - State management is clearer (current position, line, column)
// - Error handling is simpler (errors can reference lexer state)
// - It matches Go idioms (similar to bufio.Scanner)
type Lexer struct {
	// source is the complete source code being lexed.
	// We store the entire source rather than streaming because:
	// - It simplifies lookahead (can peek multiple characters ahead)
	// - Position tracking is easier (can seek to any offset)
	// - Error reporting can show context (lines before/after error)
	// - Modern compilers typically fit entire files in memory
	source string

	// filename is the name of the source file (for error reporting).
	filename string

	// start is the byte offset of the current token being scanned.
	// This is set when we begin scanning a token and used to extract
	// the token's lexeme (source[start:current]).
	start int

	// current is the byte offset we're currently examining.
	// This advances as we scan through the source.
	current int

	// line is the current line number (1-based).
	// Updated when we encounter newlines.
	line int

	// lineStart is the byte offset where the current line started.
	// Used to calculate column numbers: column = current - lineStart + 1
	//
	// DESIGN CHOICE: We track lineStart rather than column directly because:
	// - It's more efficient (no increment on every character)
	// - It's more accurate (handles multi-byte UTF-8 correctly)
	// - Column can be computed on demand when creating tokens
	lineStart int
}

// New creates a new Lexer for the given source code.
//
// DESIGN CHOICE: Constructor function (New) rather than struct literal because:
// - It can perform initialization (set line to 1, etc.)
// - It provides a clear entry point to the API
// - It can validate parameters if needed
// - It matches Go conventions (strings.Builder, bufio.Scanner, etc.)
func New(source, filename string) *Lexer {
	return &Lexer{
		source:    source,
		filename:  filename,
		start:     0,
		current:   0,
		line:      1, // Lines are 1-based
		lineStart: 0,
	}
}

// NextToken returns the next token from the source.
//
// This is the main entry point for consuming tokens. The parser will call this
// repeatedly until it gets TokenEOF.
//
// DESIGN CHOICE: Return (Token, error) rather than just Token with Type=Invalid because:
// - It follows Go conventions (explicit error handling)
// - Errors can carry more context than just a token
// - Parser can decide how to handle errors (stop, recover, accumulate)
//
// However, for some errors we still return a token with TokenInvalid because:
// - It allows error recovery (parser can continue)
// - The position in the token is useful
// - Multiple errors can be reported in one pass
func (l *Lexer) NextToken() (Token, error) {
	// Skip whitespace and comments before each token.
	// DESIGN CHOICE: Skip whitespace here rather than in a separate phase because:
	// - It's simpler (one pass through the source)
	// - Position tracking is easier
	// - Some languages need to preserve whitespace (Python), ours doesn't
	l.skipWhitespace()

	// Mark the start of this token
	l.start = l.current

	// Check for end of file
	if l.isAtEnd() {
		return l.makeToken(TokenEOF, ""), nil
	}

	// Read the next character
	ch, size := l.advance()

	// Scan the token based on the first character.
	// We use a big switch statement rather than a map of handlers because:
	// - It's faster (no map lookup, direct jump table)
	// - It's more readable (everything in one place)
	// - It's easier for the compiler to optimize
	// - Most tokens are 1-2 characters, so no benefit from complex dispatch

	// Identifiers and keywords start with a letter or underscore
	if isLetter(ch) {
		return l.scanIdentifier(), nil
	}

	// Numbers start with a digit
	if isDigit(ch) {
		return l.scanNumber()
	}

	// Everything else is operators, delimiters, or invalid
	switch ch {
	// Single-character tokens
	case '(':
		return l.makeToken(TokenLeftParen, "("), nil
	case ')':
		return l.makeToken(TokenRightParen, ")"), nil
	case '{':
		return l.makeToken(TokenLeftBrace, "{"), nil
	case '}':
		return l.makeToken(TokenRightBrace, "}"), nil
	case '[':
		return l.makeToken(TokenLeftBracket, "["), nil
	case ']':
		return l.makeToken(TokenRightBracket, "]"), nil
	case ';':
		return l.makeToken(TokenSemicolon, ";"), nil
	case ',':
		return l.makeToken(TokenComma, ","), nil
	case '~':
		return l.makeToken(TokenBitNot, "~"), nil
	case '?':
		return l.makeToken(TokenQuestion, "?"), nil

	// Operators that can be single or double characters
	case '+':
		if l.match('+') {
			return l.makeToken(TokenPlusPlus, "++"), nil
		} else if l.match('=') {
			return l.makeToken(TokenPlusEq, "+="), nil
		}
		return l.makeToken(TokenPlus, "+"), nil

	case '-':
		if l.match('-') {
			return l.makeToken(TokenMinusMinus, "--"), nil
		} else if l.match('=') {
			return l.makeToken(TokenMinusEq, "-="), nil
		} else if l.match('>') {
			return l.makeToken(TokenArrow, "->"), nil
		}
		return l.makeToken(TokenMinus, "-"), nil

	case '*':
		if l.match('*') {
			return l.makeToken(TokenStarStar, "**"), nil
		} else if l.match('=') {
			return l.makeToken(TokenStarEq, "*="), nil
		}
		return l.makeToken(TokenStar, "*"), nil

	case '/':
		// Check for comments
		if l.match('/') {
			return l.scanLineComment(), nil
		} else if l.match('*') {
			return l.scanBlockComment()
		} else if l.match('=') {
			return l.makeToken(TokenSlashEq, "/="), nil
		}
		return l.makeToken(TokenSlash, "/"), nil

	case '%':
		if l.match('=') {
			return l.makeToken(TokenPercentEq, "%="), nil
		}
		return l.makeToken(TokenPercent, "%"), nil

	case '&':
		if l.match('&') {
			return l.makeToken(TokenAnd, "&&"), nil
		} else if l.match('=') {
			return l.makeToken(TokenAndEq, "&="), nil
		}
		return l.makeToken(TokenBitAnd, "&"), nil

	case '|':
		if l.match('|') {
			return l.makeToken(TokenOr, "||"), nil
		} else if l.match('=') {
			return l.makeToken(TokenOrEq, "|="), nil
		}
		return l.makeToken(TokenBitOr, "|"), nil

	case '^':
		if l.match('=') {
			return l.makeToken(TokenXorEq, "^="), nil
		}
		return l.makeToken(TokenBitXor, "^"), nil

	case '=':
		if l.match('=') {
			return l.makeToken(TokenEqual, "=="), nil
		}
		return l.makeToken(TokenAssign, "="), nil

	case '!':
		if l.match('=') {
			return l.makeToken(TokenNotEqual, "!="), nil
		}
		return l.makeToken(TokenNot, "!"), nil

	case '<':
		if l.match('<') {
			if l.match('=') {
				return l.makeToken(TokenShlEq, "<<="), nil
			}
			return l.makeToken(TokenShl, "<<"), nil
		} else if l.match('=') {
			return l.makeToken(TokenLessEqual, "<="), nil
		}
		return l.makeToken(TokenLess, "<"), nil

	case '>':
		if l.match('>') {
			if l.match('=') {
				return l.makeToken(TokenShrEq, ">>="), nil
			}
			return l.makeToken(TokenShr, ">>"), nil
		} else if l.match('=') {
			return l.makeToken(TokenGreaterEqual, ">="), nil
		}
		return l.makeToken(TokenGreater, ">"), nil

	case ':':
		if l.match(':') {
			return l.makeToken(TokenColonColon, "::"), nil
		}
		return l.makeToken(TokenColon, ":"), nil

	case '.':
		// Check for ellipsis (...)
		if l.match('.') && l.match('.') {
			return l.makeToken(TokenEllipsis, "..."), nil
		}
		return l.makeToken(TokenDot, "."), nil

	case '"':
		return l.scanString()

	case '\'':
		return l.scanChar()

	default:
		// Invalid character
		_ = size // Unused for now, but needed for UTF-8 multi-byte handling
		return l.makeToken(TokenInvalid, ""),
			l.error(fmt.Sprintf("unexpected character: %q", ch))
	}
}

// advance reads and returns the next character, advancing the current position.
//
// DESIGN CHOICE: Return both the rune and its size because:
// - We need the rune for character classification (isDigit, isLetter, etc.)
// - We need the size for position tracking (current += size)
//
// UNICODE HANDLING: We use utf8.DecodeRuneInString for proper Unicode support.
// This means "世界" is two characters, not six bytes.
func (l *Lexer) advance() (rune, int) {
	if l.isAtEnd() {
		return 0, 0
	}
	ch, size := utf8.DecodeRuneInString(l.source[l.current:])
	l.current += size
	return ch, size
}

// peek returns the current character without advancing.
// Returns 0 if at end of file.
//
// This is used for lookahead when we need to decide how to tokenize based on
// the next character (e.g., "=" vs "==", "." vs "...").
func (l *Lexer) peek() rune {
	if l.isAtEnd() {
		return 0
	}
	ch, _ := utf8.DecodeRuneInString(l.source[l.current:])
	return ch
}

// peekNext returns the character after the current one without advancing.
// Returns 0 if not enough characters remain.
//
// This is used for two-character lookahead (e.g., "..." for ellipsis).
func (l *Lexer) peekNext() rune {
	if l.current+1 >= len(l.source) {
		return 0
	}
	// Skip the current character to get the next one
	_, size := utf8.DecodeRuneInString(l.source[l.current:])
	ch, _ := utf8.DecodeRuneInString(l.source[l.current+size:])
	return ch
}

// match checks if the current character matches the expected one.
// If it does, advance and return true. Otherwise, return false.
//
// This is a convenience function for optional characters in operators.
// Example: after seeing '+', we match('+') to check for '++'.
func (l *Lexer) match(expected rune) bool {
	if l.isAtEnd() {
		return false
	}
	ch, size := utf8.DecodeRuneInString(l.source[l.current:])
	if ch != expected {
		return false
	}
	l.current += size
	return true
}

// isAtEnd returns true if we've consumed all the source code.
func (l *Lexer) isAtEnd() bool {
	return l.current >= len(l.source)
}

// skipWhitespace skips over whitespace characters.
//
// DESIGN CHOICE: We skip whitespace rather than tokenizing it because:
// - Our language doesn't use whitespace for structure (unlike Python)
// - It simplifies the parser (doesn't have to filter whitespace tokens)
// - It's more efficient (fewer tokens to process)
//
// However, we DO track newlines for position information.
func (l *Lexer) skipWhitespace() {
	for {
		if l.isAtEnd() {
			return
		}

		ch := l.peek()
		switch ch {
		case ' ', '\r', '\t':
			// Simple whitespace - just skip it
			l.advance()
		case '\n':
			// Newline - skip it but update line tracking
			l.advance()
			l.line++
			l.lineStart = l.current
		default:
			// Not whitespace - stop skipping
			return
		}
	}
}

// scanIdentifier scans an identifier or keyword.
//
// RULES:
// - Starts with a letter or underscore
// - Continues with letters, digits, or underscores
// - Examples: foo, _bar, hello123, _
//
// DESIGN CHOICE: We allow leading underscores because:
// - It's common in many languages (Go, C, Python)
// - Useful for "private" or "internal" identifiers
// - No ambiguity (operators don't start with underscore)
func (l *Lexer) scanIdentifier() Token {
	// Continue while we see letters, digits, or underscores
	for !l.isAtEnd() {
		ch := l.peek()
		if isLetter(ch) || isDigit(ch) {
			l.advance()
		} else {
			break
		}
	}

	// Extract the identifier text
	text := l.source[l.start:l.current]

	// Check if it's a keyword
	tokenType := LookupKeyword(text)

	return l.makeToken(tokenType, text)
}

// scanNumber scans a numeric literal.
//
// SUPPORTED FORMATS:
// - Integers: 123, 0, 999999
// - Floats: 123.456, 0.5, .5 (if we allow leading dot)
// - Scientific notation: 1.23e10, 1e-5 (if we support it)
// - Hex: 0x1234, 0xFF (if we support it)
// - Binary: 0b1010 (if we support it)
// - Octal: 0o777 (if we support it)
//
// For now, we'll implement a simple version that handles integers and floats.
//
// DESIGN CHOICE: The lexer doesn't validate number format (e.g., overflow).
// It just recognizes that it's a number and passes it to the parser.
// The semantic analyzer will validate and convert to the appropriate type.
func (l *Lexer) scanNumber() (Token, error) {
	// Scan integer part
	for !l.isAtEnd() && isDigit(l.peek()) {
		l.advance()
	}

	// Check for decimal point
	if !l.isAtEnd() && l.peek() == '.' {
		// Make sure it's not "..." (ellipsis) or ".field" (member access)
		if l.peekNext() != '.' && isDigit(l.peekNext()) {
			// Consume the '.'
			l.advance()

			// Scan fractional part
			for !l.isAtEnd() && isDigit(l.peek()) {
				l.advance()
			}
		}
	}

	// Check for scientific notation (e.g., 1e10, 1.5e-3)
	if !l.isAtEnd() && (l.peek() == 'e' || l.peek() == 'E') {
		// Save position in case this isn't scientific notation
		savedCurrent := l.current
		l.advance() // consume 'e' or 'E'

		// Optional sign
		if !l.isAtEnd() && (l.peek() == '+' || l.peek() == '-') {
			l.advance()
		}

		// Must have at least one digit
		if l.isAtEnd() || !isDigit(l.peek()) {
			// Not a valid exponent - backtrack
			l.current = savedCurrent
		} else {
			// Scan exponent digits
			for !l.isAtEnd() && isDigit(l.peek()) {
				l.advance()
			}
		}
	}

	text := l.source[l.start:l.current]
	return l.makeToken(TokenNumber, text), nil
}

// scanString scans a string literal.
//
// SUPPORTED FEATURES:
// - Escape sequences: \n, \t, \r, \\, \", etc.
// - Unicode escapes: \u1234, \U00012345 (if we support them)
// - Raw strings: `...` (if we support them, like Go)
//
// DESIGN CHOICE: We don't process escape sequences here.
// We just scan the raw string and let the parser/semantic analyzer handle escaping.
// This is because:
// - It keeps the lexer simple
// - Error reporting is better (can show the original string)
// - Some languages have complex escaping rules
func (l *Lexer) scanString() (Token, error) {
	// We've already consumed the opening quote
	for !l.isAtEnd() {
		ch := l.peek()

		if ch == '"' {
			// Found closing quote
			l.advance()
			text := l.source[l.start:l.current]
			return l.makeToken(TokenString, text), nil
		}

		if ch == '\n' {
			// Unterminated string (newline before closing quote)
			return l.makeToken(TokenInvalid, ""),
				l.error("unterminated string literal")
		}

		if ch == '\\' {
			// Escape sequence - consume both backslash and next char
			l.advance()
			if !l.isAtEnd() {
				l.advance()
			}
		} else {
			l.advance()
		}
	}

	// Reached end of file without closing quote
	return l.makeToken(TokenInvalid, ""),
		l.error("unterminated string literal")
}

// scanChar scans a character literal.
//
// EXAMPLES: 'a', '\n', '\t', '\u1234'
//
// RULES:
// - Single character between single quotes
// - Can use escape sequences
// - Must have exactly one character (after escaping)
func (l *Lexer) scanChar() (Token, error) {
	// We've already consumed the opening quote

	if l.isAtEnd() {
		return l.makeToken(TokenInvalid, ""),
			l.error("unterminated character literal")
	}

	ch := l.peek()

	if ch == '\n' {
		return l.makeToken(TokenInvalid, ""),
			l.error("unterminated character literal")
	}

	if ch == '\\' {
		// Escape sequence
		l.advance() // consume backslash
		if !l.isAtEnd() {
			l.advance() // consume escaped character
		}
	} else {
		l.advance() // consume regular character
	}

	// Expect closing quote
	if l.isAtEnd() || l.peek() != '\'' {
		return l.makeToken(TokenInvalid, ""),
			l.error("unterminated character literal")
	}

	l.advance() // consume closing quote

	text := l.source[l.start:l.current]
	return l.makeToken(TokenChar, text), nil
}

// scanLineComment scans a line comment (// ...).
//
// DESIGN CHOICE: We return comments as tokens rather than skipping them because:
// - Documentation tools need them
// - Code formatters need them
// - IDE features may use them
//
// The parser can choose to ignore them if it wants.
func (l *Lexer) scanLineComment() Token {
	// Consume everything until newline
	for !l.isAtEnd() && l.peek() != '\n' {
		l.advance()
	}

	text := l.source[l.start:l.current]
	return l.makeToken(TokenComment, text)
}

// scanBlockComment scans a block comment (/* ... */).
//
// DESIGN CHOICE: We support nested block comments because:
// - It's more useful (can comment out code that contains comments)
// - It's what users expect (from languages like Swift, Rust)
// - It's only slightly more complex to implement
func (l *Lexer) scanBlockComment() (Token, error) {
	// Track nesting depth
	depth := 1

	for !l.isAtEnd() && depth > 0 {
		ch := l.peek()

		if ch == '/' && l.peekNext() == '*' {
			// Nested comment start
			l.advance()
			l.advance()
			depth++
		} else if ch == '*' && l.peekNext() == '/' {
			// Comment end
			l.advance()
			l.advance()
			depth--
		} else {
			if ch == '\n' {
				l.line++
				l.lineStart = l.current + 1
			}
			l.advance()
		}
	}

	if depth > 0 {
		return l.makeToken(TokenInvalid, ""),
			l.error("unterminated block comment")
	}

	text := l.source[l.start:l.current]
	return l.makeToken(TokenComment, text), nil
}

// makeToken creates a token with the current position information.
//
// DESIGN CHOICE: Helper function to ensure consistent token creation.
// All position tracking happens here in one place.
func (l *Lexer) makeToken(tokenType TokenType, lexeme string) Token {
	return Token{
		Type:     tokenType,
		Lexeme:   lexeme,
		Position: l.currentPosition(),
		Length:   l.current - l.start,
	}
}

// currentPosition returns the current position in the source.
func (l *Lexer) currentPosition() Position {
	return Position{
		Filename: l.filename,
		Line:     l.line,
		Column:   l.start - l.lineStart + 1, // 1-based column at start of token
		Offset:   l.start,                    // 0-based
	}
}

// error creates an error with the current position.
func (l *Lexer) error(message string) error {
	return fmt.Errorf("%s: %s", l.currentPosition().String(), message)
}

// Helper functions for character classification

// isLetter returns true if the rune is a letter or underscore.
//
// DESIGN CHOICE: We use Unicode letter classification rather than just ASCII because:
// - It's more inclusive (supports identifiers in other languages)
// - Go provides good Unicode support (unicode.IsLetter)
// - No significant performance cost
//
// However, we could restrict to ASCII for a simpler language.
func isLetter(ch rune) bool {
	return unicode.IsLetter(ch) || ch == '_'
}

// isDigit returns true if the rune is a decimal digit (0-9).
//
// DESIGN CHOICE: We only support ASCII digits, not Unicode digits because:
// - Numeric literals should be ASCII for clarity
// - It's what most programming languages do
// - It's simpler and faster
func isDigit(ch rune) bool {
	return ch >= '0' && ch <= '9'
}
