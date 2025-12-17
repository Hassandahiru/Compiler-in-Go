package types

import (
	"testing"
)

func TestPrimitiveType_String(t *testing.T) {
	tests := []struct {
		typ      Type
		expected string
	}{
		{Int, "int"},
		{Float, "float"},
		{Bool, "bool"},
		{String, "string"},
		{Char, "char"},
		{Void, "void"},
		{Invalid, "<invalid>"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := tt.typ.String()
			if result != tt.expected {
				t.Errorf("Type.String() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestPrimitiveType_Equals(t *testing.T) {
	tests := []struct {
		name     string
		t1       Type
		t2       Type
		expected bool
	}{
		{"int equals int", Int, Int, true},
		{"float equals float", Float, Float, true},
		{"int not equals float", Int, Float, false},
		{"bool not equals int", Bool, Int, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.t1.Equals(tt.t2)
			if result != tt.expected {
				t.Errorf("%s.Equals(%s) = %v, want %v",
					tt.t1, tt.t2, result, tt.expected)
			}
		})
	}
}

func TestPrimitiveType_AssignableTo(t *testing.T) {
	tests := []struct {
		name     string
		value    Type
		target   Type
		expected bool
	}{
		{"int to int", Int, Int, true},
		{"float to float", Float, Float, true},
		{"int to float (not allowed)", Int, Float, false},
		{"bool to int (not allowed)", Bool, Int, false},
		{"invalid to anything", Invalid, Int, false},
		{"anything to invalid", Int, Invalid, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.value.AssignableTo(tt.target)
			if result != tt.expected {
				t.Errorf("%s.AssignableTo(%s) = %v, want %v",
					tt.value, tt.target, result, tt.expected)
			}
		})
	}
}

func TestFunctionType(t *testing.T) {
	params := []Type{Int, Float}
	returnType := Bool
	funcType := NewFunction(params, returnType)

	// Test String
	expected := "func(int, float) bool"
	if funcType.String() != expected {
		t.Errorf("FunctionType.String() = %q, want %q", funcType.String(), expected)
	}

	// Test Equals
	sameFuncType := NewFunction(params, returnType)
	if !funcType.Equals(sameFuncType) {
		t.Error("Expected same function types to be equal")
	}

	differentFuncType := NewFunction([]Type{Int}, returnType)
	if funcType.Equals(differentFuncType) {
		t.Error("Expected different function types to not be equal")
	}

	// Function types should not equal primitive types
	if funcType.Equals(Int) {
		t.Error("Expected function type to not equal primitive type")
	}
}

func TestStructType(t *testing.T) {
	fields := []StructField{
		{Name: "x", Type: Int},
		{Name: "y", Type: Float},
	}
	structType := NewStruct("Point", fields)

	// Test String
	if !contains(structType.String(), "Point") {
		t.Errorf("StructType.String() should contain name, got %q", structType.String())
	}

	// Test LookupField
	field := structType.LookupField("x")
	if field == nil {
		t.Error("Expected to find field 'x'")
	} else if field.Name != "x" {
		t.Errorf("Expected field name 'x', got %q", field.Name)
	}

	// Test non-existent field
	field = structType.LookupField("z")
	if field != nil {
		t.Error("Expected nil for non-existent field 'z'")
	}

	// Test Equals
	sameStructType := NewStruct("Point", fields)
	if !structType.Equals(sameStructType) {
		t.Error("Expected same struct types to be equal")
	}

	differentStructType := NewStruct("Point2", fields)
	if structType.Equals(differentStructType) {
		t.Error("Expected different struct names to not be equal")
	}
}

func TestIsNumeric(t *testing.T) {
	tests := []struct {
		name     string
		typ      Type
		expected bool
	}{
		{"int is numeric", Int, true},
		{"float is numeric", Float, true},
		{"bool is not numeric", Bool, false},
		{"string is not numeric", String, false},
		{"void is not numeric", Void, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsNumeric(tt.typ)
			if result != tt.expected {
				t.Errorf("IsNumeric(%s) = %v, want %v",
					tt.typ, result, tt.expected)
			}
		})
	}
}

func TestIsBooleanType(t *testing.T) {
	tests := []struct {
		name     string
		typ      Type
		expected bool
	}{
		{"bool is boolean", Bool, true},
		{"int is not boolean", Int, false},
		{"string is not boolean", String, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsBooleanType(tt.typ)
			if result != tt.expected {
				t.Errorf("IsBooleanType(%s) = %v, want %v",
					tt.typ, result, tt.expected)
			}
		})
	}
}

func TestIsIntegerType(t *testing.T) {
	tests := []struct {
		name     string
		typ      Type
		expected bool
	}{
		{"int is integer", Int, true},
		{"float is not integer", Float, false},
		{"bool is not integer", Bool, false},
		{"char is not integer", Char, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsIntegerType(tt.typ)
			if result != tt.expected {
				t.Errorf("IsIntegerType(%s) = %v, want %v",
					tt.typ, result, tt.expected)
			}
		})
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
