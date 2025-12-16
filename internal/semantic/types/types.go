// Package types implements the type system for the compiler.
//
// DESIGN PHILOSOPHY:
// A strong, static type system catches errors at compile time and enables optimizations.
// Our type system supports:
// 1. Primitive types (int, float, bool, string, etc.)
// 2. Composite types (arrays, structs)
// 3. Function types
// 4. Type checking and inference
// 5. Type compatibility and conversion rules
//
// KEY DESIGN CHOICES:
// - Nominal typing for structs (struct Point != struct{x int; y int})
// - Structural typing for function types (func(int) int == func(int) int)
// - Explicit conversions required (no implicit int->float)
// - Type inference from initializers (var x = 5 infers int)
package types

import (
	"fmt"
	"strings"
)

// Type is the interface that all types implement.
//
// DESIGN CHOICE: Use an interface rather than a struct with a "kind" field because:
// - Type-safe (each type has its own struct)
// - Easy to extend (add new methods to specific types)
// - Pattern matching via type switches
// - Follows Go conventions (ast.Node, etc.)
type Type interface {
	// String returns a human-readable representation of the type
	String() string

	// Equals checks if this type is identical to another type
	//
	// IDENTITY RULES:
	// - Primitive types: equal if same kind (int == int)
	// - Arrays: equal if element type and size are equal
	// - Structs: equal if same struct (nominal typing)
	// - Functions: equal if parameters and return type are equal (structural)
	Equals(other Type) bool

	// AssignableTo checks if a value of this type can be assigned to another type
	//
	// ASSIGNABILITY RULES:
	// - Identical types are assignable
	// - nil is assignable to any pointer/array/struct type (in some languages)
	// - Specific rules for each type (see individual types)
	//
	// This is more lenient than Equals (e.g., named type vs underlying type)
	AssignableTo(other Type) bool

	// kind returns the kind of type (for internal use)
	// We don't export this because external code should use type switches
	kind() TypeKind
}

// TypeKind represents the kind of type.
// This is used internally for quick type checks.
type TypeKind int

const (
	KindInvalid TypeKind = iota
	KindVoid
	KindInt
	KindFloat
	KindBool
	KindString
	KindChar
	KindArray
	KindStruct
	KindFunction
	KindNil
)

// Base type implementations

// InvalidType represents an invalid or error type.
// This is used when type checking fails, to allow checking to continue.
//
// DESIGN CHOICE: Use a special type for errors rather than nil because:
// - Prevents nil pointer panics
// - Can continue type checking after errors
// - Errors are caught, but we can still analyze rest of code
type InvalidType struct{}

func (i *InvalidType) String() string           { return "<invalid>" }
func (i *InvalidType) Equals(other Type) bool   { return false }
func (i *InvalidType) AssignableTo(Type) bool   { return false }
func (i *InvalidType) kind() TypeKind            { return KindInvalid }

// VoidType represents the absence of a type (void functions)
type VoidType struct{}

func (v *VoidType) String() string           { return "void" }
func (v *VoidType) Equals(other Type) bool   { _, ok := other.(*VoidType); return ok }
func (v *VoidType) AssignableTo(Type) bool   { return false }
func (v *VoidType) kind() TypeKind            { return KindVoid }

// IntType represents integer type
type IntType struct{}

func (i *IntType) String() string           { return "int" }
func (i *IntType) Equals(other Type) bool   { _, ok := other.(*IntType); return ok }
func (i *IntType) AssignableTo(other Type) bool { return i.Equals(other) }
func (i *IntType) kind() TypeKind            { return KindInt }

// FloatType represents floating-point type
type FloatType struct{}

func (f *FloatType) String() string           { return "float" }
func (f *FloatType) Equals(other Type) bool   { _, ok := other.(*FloatType); return ok }
func (f *FloatType) AssignableTo(other Type) bool { return f.Equals(other) }
func (f *FloatType) kind() TypeKind            { return KindFloat }

// BoolType represents boolean type
type BoolType struct{}

func (b *BoolType) String() string           { return "bool" }
func (b *BoolType) Equals(other Type) bool   { _, ok := other.(*BoolType); return ok }
func (b *BoolType) AssignableTo(other Type) bool { return b.Equals(other) }
func (b *BoolType) kind() TypeKind            { return KindBool }

// StringType represents string type
type StringType struct{}

func (s *StringType) String() string           { return "string" }
func (s *StringType) Equals(other Type) bool   { _, ok := other.(*StringType); return ok }
func (s *StringType) AssignableTo(other Type) bool { return s.Equals(other) }
func (s *StringType) kind() TypeKind            { return KindString }

// CharType represents character type
type CharType struct{}

func (c *CharType) String() string           { return "char" }
func (c *CharType) Equals(other Type) bool   { _, ok := other.(*CharType); return ok }
func (c *CharType) AssignableTo(other Type) bool { return c.Equals(other) }
func (c *CharType) kind() TypeKind            { return KindChar }

// NilType represents the type of the nil literal
//
// DESIGN CHOICE: Separate type for nil because:
// - nil is assignable to many types (pointers, arrays, etc.)
// - Makes type checking clearer
// - Matches languages like Go, Java
type NilType struct{}

func (n *NilType) String() string           { return "nil" }
func (n *NilType) Equals(other Type) bool   { _, ok := other.(*NilType); return ok }
func (n *NilType) AssignableTo(other Type) bool {
	// nil is assignable to arrays and structs (nullable types)
	switch other.(type) {
	case *ArrayType, *StructType:
		return true
	default:
		return false
	}
}
func (n *NilType) kind() TypeKind { return KindNil }

// Composite types

// ArrayType represents an array type: []T or [N]T
//
// DESIGN CHOICE: Single type for both fixed and dynamic arrays because:
// - Similar operations (indexing, iteration)
// - Size -1 indicates dynamic array
// - Simplifies type checking
//
// Alternative: Separate SliceType and ArrayType (like Go)
// - More accurate representation
// - Different semantics (slices are references)
// - But more complex for our simple language
type ArrayType struct {
	ElementType Type
	Size        int // -1 for dynamic arrays (slices)
}

func (a *ArrayType) String() string {
	if a.Size < 0 {
		return "[]" + a.ElementType.String()
	}
	return fmt.Sprintf("[%d]%s", a.Size, a.ElementType.String())
}

func (a *ArrayType) Equals(other Type) bool {
	if otherArray, ok := other.(*ArrayType); ok {
		return a.Size == otherArray.Size &&
			a.ElementType.Equals(otherArray.ElementType)
	}
	return false
}

func (a *ArrayType) AssignableTo(other Type) bool {
	return a.Equals(other)
}

func (a *ArrayType) kind() TypeKind {
	return KindArray
}

// StructType represents a struct type
//
// DESIGN CHOICE: Store fields as a slice rather than a map because:
// - Preserves field order (important for memory layout)
// - Simpler to iterate over
// - Field lookup is done via symbol table, not here
//
// NOMINAL TYPING: Structs are equal only if they're the same struct.
// struct Point {x int; y int} != struct {x int; y int}
// This is because:
// - Clearer semantics (explicit type names required)
// - Better error messages ("expected Point, got Position")
// - Matches Go, Java, C++
type StructType struct {
	Name   string
	Fields []StructField
}

// StructField represents a field in a struct
type StructField struct {
	Name string
	Type Type
}

func (s *StructType) String() string {
	if s.Name != "" {
		return "struct " + s.Name
	}
	// Anonymous struct
	parts := make([]string, len(s.Fields))
	for i, field := range s.Fields {
		parts[i] = field.Name + " " + field.Type.String()
	}
	return "struct {" + strings.Join(parts, "; ") + "}"
}

func (s *StructType) Equals(other Type) bool {
	if otherStruct, ok := other.(*StructType); ok {
		// Named structs: compare by name (nominal typing)
		if s.Name != "" && otherStruct.Name != "" {
			return s.Name == otherStruct.Name
		}
		// Anonymous structs: compare structurally
		if len(s.Fields) != len(otherStruct.Fields) {
			return false
		}
		for i, field := range s.Fields {
			otherField := otherStruct.Fields[i]
			if field.Name != otherField.Name || !field.Type.Equals(otherField.Type) {
				return false
			}
		}
		return true
	}
	return false
}

func (s *StructType) AssignableTo(other Type) bool {
	return s.Equals(other)
}

func (s *StructType) kind() TypeKind {
	return KindStruct
}

// LookupField finds a field by name
// Returns nil if not found
func (s *StructType) LookupField(name string) *StructField {
	for i := range s.Fields {
		if s.Fields[i].Name == name {
			return &s.Fields[i]
		}
	}
	return nil
}

// FunctionType represents a function type
//
// STRUCTURAL TYPING: Functions are equal if they have the same signature.
// func(int, int) int == func(int, int) int
// This is because:
// - Functions are values (can be passed around)
// - Names don't matter (func foo(a int) vs func bar(x int))
// - Matches how most languages handle function types
type FunctionType struct {
	Parameters []Type
	ReturnType Type
}

func (f *FunctionType) String() string {
	params := make([]string, len(f.Parameters))
	for i, param := range f.Parameters {
		params[i] = param.String()
	}
	returnStr := f.ReturnType.String()
	return fmt.Sprintf("func(%s) %s", strings.Join(params, ", "), returnStr)
}

func (f *FunctionType) Equals(other Type) bool {
	if otherFunc, ok := other.(*FunctionType); ok {
		// Check return type
		if !f.ReturnType.Equals(otherFunc.ReturnType) {
			return false
		}
		// Check parameters
		if len(f.Parameters) != len(otherFunc.Parameters) {
			return false
		}
		for i, param := range f.Parameters {
			if !param.Equals(otherFunc.Parameters[i]) {
				return false
			}
		}
		return true
	}
	return false
}

func (f *FunctionType) AssignableTo(other Type) bool {
	return f.Equals(other)
}

func (f *FunctionType) kind() TypeKind {
	return KindFunction
}

// Predefined type instances (singletons)
// These are used throughout the compiler to avoid allocating new type instances
var (
	Invalid = &InvalidType{}
	Void    = &VoidType{}
	Int     = &IntType{}
	Float   = &FloatType{}
	Bool    = &BoolType{}
	String  = &StringType{}
	Char    = &CharType{}
	Nil     = &NilType{}
)

// Helper functions

// IsNumeric returns true if the type is numeric (int or float)
func IsNumeric(t Type) bool {
	switch t.(type) {
	case *IntType, *FloatType:
		return true
	default:
		return false
	}
}

// IsComparable returns true if values of this type can be compared with ==, !=
func IsComparable(t Type) bool {
	switch t.(type) {
	case *IntType, *FloatType, *BoolType, *StringType, *CharType:
		return true
	default:
		return false
	}
}

// IsOrdered returns true if values of this type can be compared with <, <=, >, >=
func IsOrdered(t Type) bool {
	switch t.(type) {
	case *IntType, *FloatType, *StringType, *CharType:
		return true
	default:
		return false
	}
}

// IsBooleanType returns true if the type is boolean
func IsBooleanType(t Type) bool {
	_, ok := t.(*BoolType)
	return ok
}

// IsIntegerType returns true if the type is integer
func IsIntegerType(t Type) bool {
	_, ok := t.(*IntType)
	return ok
}

// NewArray creates a new array type
func NewArray(elementType Type, size int) *ArrayType {
	return &ArrayType{
		ElementType: elementType,
		Size:        size,
	}
}

// NewStruct creates a new struct type
func NewStruct(name string, fields []StructField) *StructType {
	return &StructType{
		Name:   name,
		Fields: fields,
	}
}

// NewFunction creates a new function type
func NewFunction(parameters []Type, returnType Type) *FunctionType {
	return &FunctionType{
		Parameters: parameters,
		ReturnType: returnType,
	}
}
