package lexer

import (
	"testing"
)

func TestPosition_String(t *testing.T) {
	tests := []struct {
		name     string
		pos      Position
		expected string
	}{
		{
			name: "valid position",
			pos: Position{
				Filename: "test.go",
				Line:     42,
				Column:   15,
				Offset:   100,
			},
			expected: "test.go:42:15",
		},
		{
			name: "zero position",
			pos: Position{
				Filename: "",
				Line:     0,
				Column:   0,
				Offset:   0,
			},
			expected: ":0:0",
		},
		{
			name: "line 1 column 1",
			pos: Position{
				Filename: "main.go",
				Line:     1,
				Column:   1,
				Offset:   0,
			},
			expected: "main.go:1:1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.pos.String()
			if result != tt.expected {
				t.Errorf("Position.String() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestPosition_IsValid(t *testing.T) {
	tests := []struct {
		name     string
		pos      Position
		expected bool
	}{
		{
			name: "valid position",
			pos: Position{
				Filename: "test.go",
				Line:     1,
				Column:   1,
			},
			expected: true,
		},
		{
			name: "zero line (invalid)",
			pos: Position{
				Filename: "test.go",
				Line:     0,
				Column:   1,
			},
			expected: false,
		},
		{
			name: "negative line (invalid)",
			pos: Position{
				Filename: "test.go",
				Line:     -1,
				Column:   1,
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.pos.IsValid()
			if result != tt.expected {
				t.Errorf("Position.IsValid() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestPosition_Before(t *testing.T) {
	tests := []struct {
		name     string
		pos      Position
		other    Position
		expected bool
	}{
		{
			name: "pos before other",
			pos: Position{
				Offset: 10,
			},
			other: Position{
				Offset: 20,
			},
			expected: true,
		},
		{
			name: "pos after other",
			pos: Position{
				Offset: 30,
			},
			other: Position{
				Offset: 20,
			},
			expected: false,
		},
		{
			name: "pos equals other",
			pos: Position{
				Offset: 20,
			},
			other: Position{
				Offset: 20,
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.pos.Before(tt.other)
			if result != tt.expected {
				t.Errorf("Position.Before() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestPosition_After(t *testing.T) {
	tests := []struct {
		name     string
		pos      Position
		other    Position
		expected bool
	}{
		{
			name: "pos after other",
			pos: Position{
				Offset: 30,
			},
			other: Position{
				Offset: 20,
			},
			expected: true,
		},
		{
			name: "pos before other",
			pos: Position{
				Offset: 10,
			},
			other: Position{
				Offset: 20,
			},
			expected: false,
		},
		{
			name: "pos equals other",
			pos: Position{
				Offset: 20,
			},
			other: Position{
				Offset: 20,
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.pos.After(tt.other)
			if result != tt.expected {
				t.Errorf("Position.After() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestItoa(t *testing.T) {
	tests := []struct {
		name     string
		input    int
		expected string
	}{
		{
			name:     "zero",
			input:    0,
			expected: "0",
		},
		{
			name:     "positive number",
			input:    42,
			expected: "42",
		},
		{
			name:     "negative number",
			input:    -10,
			expected: "-10",
		},
		{
			name:     "large number",
			input:    123456,
			expected: "123456",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := itoa(tt.input)
			if result != tt.expected {
				t.Errorf("itoa(%d) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestSpan_String(t *testing.T) {
	tests := []struct {
		name     string
		span     Span
		expected string
	}{
		{
			name: "single line span",
			span: Span{
				Start: Position{
					Filename: "test.go",
					Line:     42,
					Column:   15,
				},
				End: Position{
					Filename: "test.go",
					Line:     42,
					Column:   23,
				},
			},
			expected: "test.go:42:15-23",
		},
		{
			name: "multi-line span",
			span: Span{
				Start: Position{
					Filename: "test.go",
					Line:     42,
					Column:   15,
				},
				End: Position{
					Filename: "test.go",
					Line:     44,
					Column:   10,
				},
			},
			expected: "test.go:42:15-44:10",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.span.String()
			if result != tt.expected {
				t.Errorf("Span.String() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestSpan_IsValid(t *testing.T) {
	tests := []struct {
		name     string
		span     Span
		expected bool
	}{
		{
			name: "valid span",
			span: Span{
				Start: Position{Line: 1, Column: 1, Offset: 0},
				End:   Position{Line: 1, Column: 10, Offset: 9},
			},
			expected: true,
		},
		{
			name: "invalid start",
			span: Span{
				Start: Position{Line: 0, Column: 1, Offset: 0},
				End:   Position{Line: 1, Column: 10, Offset: 9},
			},
			expected: false,
		},
		{
			name: "invalid end",
			span: Span{
				Start: Position{Line: 1, Column: 1, Offset: 0},
				End:   Position{Line: 0, Column: 10, Offset: 9},
			},
			expected: false,
		},
		{
			name: "end before start",
			span: Span{
				Start: Position{Line: 1, Column: 10, Offset: 9},
				End:   Position{Line: 1, Column: 1, Offset: 0},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.span.IsValid()
			if result != tt.expected {
				t.Errorf("Span.IsValid() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestSpan_Contains(t *testing.T) {
	span := Span{
		Start: Position{Line: 1, Column: 5, Offset: 4},
		End:   Position{Line: 1, Column: 10, Offset: 9},
	}

	tests := []struct {
		name     string
		pos      Position
		expected bool
	}{
		{
			name:     "position at start",
			pos:      Position{Line: 1, Column: 5, Offset: 4},
			expected: true,
		},
		{
			name:     "position in middle",
			pos:      Position{Line: 1, Column: 7, Offset: 6},
			expected: true,
		},
		{
			name:     "position at end",
			pos:      Position{Line: 1, Column: 10, Offset: 9},
			expected: true,
		},
		{
			name:     "position before start",
			pos:      Position{Line: 1, Column: 3, Offset: 2},
			expected: false,
		},
		{
			name:     "position after end",
			pos:      Position{Line: 1, Column: 15, Offset: 14},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := span.Contains(tt.pos)
			if result != tt.expected {
				t.Errorf("Span.Contains() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestSpan_Length(t *testing.T) {
	tests := []struct {
		name     string
		span     Span
		expected int
	}{
		{
			name: "normal span",
			span: Span{
				Start: Position{Line: 1, Offset: 10},
				End:   Position{Line: 1, Offset: 20},
			},
			expected: 10,
		},
		{
			name: "zero length span",
			span: Span{
				Start: Position{Line: 1, Offset: 10},
				End:   Position{Line: 1, Offset: 10},
			},
			expected: 0,
		},
		{
			name: "invalid span (end before start)",
			span: Span{
				Start: Position{Line: 1, Offset: 20},
				End:   Position{Line: 0, Offset: 10},
			},
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.span.Length()
			if result != tt.expected {
				t.Errorf("Span.Length() = %v, want %v", result, tt.expected)
			}
		})
	}
}
