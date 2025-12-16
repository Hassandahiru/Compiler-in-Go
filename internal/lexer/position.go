// Package lexer provides lexical analysis (tokenization) functionality for the compiler.
// It transforms raw source code text into a stream of tokens that can be consumed by the parser.
package lexer

// Position represents a location in the source code.
//
// DESIGN CHOICE: Position is a value type (not a pointer) because:
// 1. It's small (4 integers = 32 bytes on 64-bit systems)
// 2. It's immutable once created
// 3. Copying is cheap and avoids pointer chasing
// 4. No need for nil state - invalid positions can use zero values
//
// Position tracking is critical for:
// - Error reporting: Users need to know exactly where errors occur
// - IDE integration: Jump-to-definition, hover info, etc.
// - Debugging: Source maps for generated code
type Position struct {
	// Filename is the name of the source file.
	// We store this in every Position rather than using a file ID because:
	// - It makes error messages self-contained and easier to read
	// - Memory overhead is acceptable (strings in Go are just pointers + length)
	// - Simplifies multi-file compilation (no need for a global file table)
	Filename string

	// Line is the 1-based line number.
	// We use 1-based indexing because:
	// - It matches how text editors display line numbers
	// - It's more intuitive for users
	// - Zero value (0) can represent "no line" or invalid position
	Line int

	// Column is the 1-based column number (character position in the line).
	// We use 1-based indexing for the same reasons as Line.
	//
	// IMPORTANT: We count in UTF-8 runes (Unicode code points), not bytes.
	// This means "hello 世界" has 8 columns, not 13 bytes.
	// This choice prioritizes user experience over implementation simplicity.
	Column int

	// Offset is the 0-based byte offset from the start of the file.
	// This is used for:
	// - Fast seeking in the source file
	// - Calculating token lengths (end.Offset - start.Offset)
	// - Creating source spans for IDE features
	//
	// We use 0-based indexing here because it's the natural way to index
	// into byte slices in Go (source[offset:offset+length])
	Offset int
}

// String returns a human-readable representation of the position.
// Format: "filename:line:column"
// Example: "main.go:42:15"
//
// DESIGN CHOICE: We follow the GCC/Clang format (file:line:column) because:
// - It's widely recognized and understood
// - Many tools (editors, CI systems) can parse this format and create clickable links
// - It's concise but complete
func (p Position) String() string {
	return p.Filename + ":" + itoa(p.Line) + ":" + itoa(p.Column)
}

// IsValid returns true if the position is valid (has a non-zero line number).
// We consider a position valid if it has a line number because:
// - Line is the minimum information needed for error reporting
// - Zero value Position{} will correctly report as invalid
func (p Position) IsValid() bool {
	return p.Line > 0
}

// Before returns true if this position comes before the other position.
// Positions are compared by offset for accuracy and simplicity.
//
// DESIGN CHOICE: We compare by offset rather than line/column because:
// - It's simpler (single comparison vs multiple)
// - It's faster (one integer comparison)
// - It's more accurate (handles edge cases like varying line lengths)
// - Offset is the source of truth; line/column are derived from it
func (p Position) Before(other Position) bool {
	return p.Offset < other.Offset
}

// After returns true if this position comes after the other position.
func (p Position) After(other Position) bool {
	return p.Offset > other.Offset
}

// itoa is a simple integer to ASCII conversion.
// We implement our own instead of using strconv.Itoa because:
// - It avoids an import
// - It's slightly faster for small numbers (which line/column numbers usually are)
// - We can optimize for the common case (numbers < 1000)
//
// PERFORMANCE NOTE: This is called frequently (every error message), so we optimize it.
// However, we prioritize code clarity over micro-optimizations.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}

	// Handle negative numbers (shouldn't happen for line/column, but be defensive)
	negative := false
	if n < 0 {
		negative = true
		n = -n
	}

	// Build the number in reverse
	buf := make([]byte, 0, 12) // 12 is enough for 32-bit int
	for n > 0 {
		buf = append(buf, byte('0'+n%10))
		n /= 10
	}

	if negative {
		buf = append(buf, '-')
	}

	// Reverse the buffer
	for i, j := 0, len(buf)-1; i < j; i, j = i+1, j-1 {
		buf[i], buf[j] = buf[j], buf[i]
	}

	return string(buf)
}

// Span represents a range in the source code from Start to End (inclusive).
//
// DESIGN CHOICE: Span is separate from Position because:
// - It represents a different concept (a range vs a point)
// - Many operations need ranges (syntax highlighting, error underlining)
// - It makes the API clearer (Token has Position for start, not two Positions)
//
// We use this for:
// - Error reporting: highlighting the entire problematic token/expression
// - IDE features: selection ranges, folding ranges
// - Code formatting: preserving whitespace and comments in the right places
type Span struct {
	Start Position
	End   Position
}

// String returns a human-readable representation of the span.
// Format: "filename:startLine:startCol-endLine:endCol"
// Example: "main.go:42:15-42:23"
//
// For single-line spans, we can optimize the output:
// "main.go:42:15-23" instead of "main.go:42:15-42:23"
func (s Span) String() string {
	if s.Start.Line == s.End.Line {
		// Same line: just show start:col1-col2
		return s.Start.Filename + ":" + itoa(s.Start.Line) + ":" +
			itoa(s.Start.Column) + "-" + itoa(s.End.Column)
	}
	// Different lines: show full range
	return s.Start.String() + "-" + itoa(s.End.Line) + ":" + itoa(s.End.Column)
}

// IsValid returns true if the span is valid (both positions are valid and ordered correctly).
func (s Span) IsValid() bool {
	return s.Start.IsValid() && s.End.IsValid() && !s.End.Before(s.Start)
}

// Contains returns true if the given position is within this span (inclusive).
func (s Span) Contains(pos Position) bool {
	return !pos.Before(s.Start) && !pos.After(s.End)
}

// Length returns the number of bytes covered by this span.
// This is useful for:
// - Allocating buffers for extracting source text
// - Calculating token sizes for performance analysis
func (s Span) Length() int {
	if !s.IsValid() {
		return 0
	}
	return s.End.Offset - s.Start.Offset
}
