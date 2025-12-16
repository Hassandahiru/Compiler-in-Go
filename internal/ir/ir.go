// Package ir implements the Intermediate Representation for the compiler.
//
// WHAT IS IR?
// IR is a low-level representation of the program that sits between the AST and machine code.
// It's designed to be:
// 1. Easy to analyze and optimize
// 2. Independent of source language and target machine
// 3. Explicit about control flow and operations
//
// DESIGN PHILOSOPHY:
// We use a Three-Address Code (TAC) style IR similar to LLVM:
// - Each instruction has at most 3 operands
// - Operations are simple and explicit
// - Control flow is represented with basic blocks
// - We use Static Single Assignment (SSA) form where possible
//
// WHY SSA FORM?
// In SSA, each variable is assigned exactly once. This makes optimization much easier:
// - Data flow is explicit
// - Dead code elimination is simpler
// - Constant propagation is easier
// - Many optimizations are more effective
//
// EXAMPLE:
//   Source:  x = a + b; y = x * 2; x = y + 1;
//   SSA:     x1 = a + b; y1 = x1 * 2; x2 = y1 + 1;
package ir

import (
	"fmt"

	"github.com/hassan/compiler/internal/semantic/types"
)

// Value represents a value in the IR (variable, constant, or temporary).
//
// DESIGN CHOICE: Use a single Value type rather than separate Variable/Constant because:
// - Simplifies instruction definitions (uniform operand type)
// - Can easily convert between them
// - Values can be tagged with their kind
type Value struct {
	// ID is a unique identifier for this value
	// In SSA form, this ensures each value is unique
	ID int

	// Name is the original variable name (if any)
	// Empty for temporaries and constants
	Name string

	// Type is the value's type
	Type types.Type

	// Kind indicates what kind of value this is
	Kind ValueKind

	// Constant is the constant value (if Kind == ValueConstant)
	Constant interface{}
}

// ValueKind represents the kind of value.
type ValueKind int

const (
	ValueVariable ValueKind = iota // Regular variable
	ValueTemporary                  // Compiler-generated temporary
	ValueConstant                   // Compile-time constant
	ValueParameter                  // Function parameter
)

func (v *Value) String() string {
	switch v.Kind {
	case ValueConstant:
		return fmt.Sprintf("const(%v)", v.Constant)
	case ValueParameter:
		if v.Name != "" {
			return fmt.Sprintf("param(%s.%d)", v.Name, v.ID)
		}
		return fmt.Sprintf("param(%d)", v.ID)
	case ValueTemporary:
		return fmt.Sprintf("t%d", v.ID)
	default:
		if v.Name != "" {
			return fmt.Sprintf("%s.%d", v.Name, v.ID)
		}
		return fmt.Sprintf("v%d", v.ID)
	}
}

// IsConstant returns true if this is a constant value.
func (v *Value) IsConstant() bool {
	return v.Kind == ValueConstant
}

// Instruction represents a single IR instruction.
//
// DESIGN CHOICE: Use an interface rather than a tagged union because:
// - More idiomatic Go
// - Type-safe pattern matching via type switches
// - Easy to add new instruction types
// - Follows the AST design
type Instruction interface {
	// String returns a human-readable representation
	String() string

	// Operands returns all values read by this instruction
	// Used for data flow analysis
	Operands() []*Value

	// Result returns the value written by this instruction (if any)
	// Returns nil for instructions that don't produce a value
	Result() *Value
}

// Binary arithmetic and logical operations
// Format: result = left op right

type BinaryOp struct {
	Op    BinaryOperator
	Dest  *Value
	Left  *Value
	Right *Value
}

func (b *BinaryOp) String() string {
	return fmt.Sprintf("%s = %s %s %s", b.Dest, b.Left, b.Op, b.Right)
}

func (b *BinaryOp) Operands() []*Value { return []*Value{b.Left, b.Right} }
func (b *BinaryOp) Result() *Value     { return b.Dest }

type BinaryOperator int

const (
	// Arithmetic
	OpAdd BinaryOperator = iota
	OpSub
	OpMul
	OpDiv
	OpMod

	// Comparison
	OpEq  // ==
	OpNeq // !=
	OpLt  // <
	OpLe  // <=
	OpGt  // >
	OpGe  // >=

	// Logical
	OpAnd // &&
	OpOr  // ||

	// Bitwise
	OpBitAnd // &
	OpBitOr  // |
	OpBitXor // ^
	OpShl    // <<
	OpShr    // >>
)

func (op BinaryOperator) String() string {
	switch op {
	case OpAdd:
		return "+"
	case OpSub:
		return "-"
	case OpMul:
		return "*"
	case OpDiv:
		return "/"
	case OpMod:
		return "%"
	case OpEq:
		return "=="
	case OpNeq:
		return "!="
	case OpLt:
		return "<"
	case OpLe:
		return "<="
	case OpGt:
		return ">"
	case OpGe:
		return ">="
	case OpAnd:
		return "&&"
	case OpOr:
		return "||"
	case OpBitAnd:
		return "&"
	case OpBitOr:
		return "|"
	case OpBitXor:
		return "^"
	case OpShl:
		return "<<"
	case OpShr:
		return ">>"
	default:
		return "?"
	}
}

// Unary operations
// Format: result = op operand

type UnaryOp struct {
	Op      UnaryOperator
	Dest    *Value
	Operand *Value
}

func (u *UnaryOp) String() string {
	return fmt.Sprintf("%s = %s%s", u.Dest, u.Op, u.Operand)
}

func (u *UnaryOp) Operands() []*Value { return []*Value{u.Operand} }
func (u *UnaryOp) Result() *Value     { return u.Dest }

type UnaryOperator int

const (
	OpNeg    UnaryOperator = iota // -x
	OpNot                          // !x
	OpBitNot                       // ~x
)

func (op UnaryOperator) String() string {
	switch op {
	case OpNeg:
		return "-"
	case OpNot:
		return "!"
	case OpBitNot:
		return "~"
	default:
		return "?"
	}
}

// Copy instruction
// Format: result = value

type Copy struct {
	Dest  *Value
	Value *Value
}

func (c *Copy) String() string {
	return fmt.Sprintf("%s = %s", c.Dest, c.Value)
}

func (c *Copy) Operands() []*Value { return []*Value{c.Value} }
func (c *Copy) Result() *Value     { return c.Dest }

// Memory operations

// Load from memory
// Format: result = *address

type Load struct {
	Dest    *Value
	Address *Value
}

func (l *Load) String() string {
	return fmt.Sprintf("%s = load %s", l.Dest, l.Address)
}

func (l *Load) Operands() []*Value { return []*Value{l.Address} }
func (l *Load) Result() *Value     { return l.Dest }

// Store to memory
// Format: *address = value

type Store struct {
	Address *Value
	Value   *Value
}

func (s *Store) String() string {
	return fmt.Sprintf("store %s, %s", s.Value, s.Address)
}

func (s *Store) Operands() []*Value { return []*Value{s.Address, s.Value} }
func (s *Store) Result() *Value     { return nil }

// Array/struct access

// GetElementPtr calculates an address offset
// Format: result = &base[index]

type GetElementPtr struct {
	Dest  *Value
	Base  *Value
	Index *Value
}

func (g *GetElementPtr) String() string {
	return fmt.Sprintf("%s = &%s[%s]", g.Dest, g.Base, g.Index)
}

func (g *GetElementPtr) Operands() []*Value { return []*Value{g.Base, g.Index} }
func (g *GetElementPtr) Result() *Value     { return g.Dest }

// Field access
// Format: result = &base.field

type GetFieldPtr struct {
	Dest       *Value
	Base       *Value
	FieldIndex int // Index of field in struct
}

func (g *GetFieldPtr) String() string {
	return fmt.Sprintf("%s = &%s.field%d", g.Dest, g.Base, g.FieldIndex)
}

func (g *GetFieldPtr) Operands() []*Value { return []*Value{g.Base} }
func (g *GetFieldPtr) Result() *Value     { return g.Dest }

// Control flow

// Jump unconditionally to a basic block
type Jump struct {
	Target *BasicBlock
}

func (j *Jump) String() string {
	return fmt.Sprintf("jump %s", j.Target.Label)
}

func (j *Jump) Operands() []*Value { return nil }
func (j *Jump) Result() *Value     { return nil }

// Conditional jump
// Format: if condition then trueBlock else falseBlock

type Branch struct {
	Condition  *Value
	TrueBlock  *BasicBlock
	FalseBlock *BasicBlock
}

func (b *Branch) String() string {
	return fmt.Sprintf("branch %s, %s, %s", b.Condition, b.TrueBlock.Label, b.FalseBlock.Label)
}

func (b *Branch) Operands() []*Value { return []*Value{b.Condition} }
func (b *Branch) Result() *Value     { return nil }

// Function call
// Format: result = call function(args...)

type Call struct {
	Dest     *Value   // Can be nil for void functions
	Function *Value   // Function to call
	Args     []*Value // Arguments
}

func (c *Call) String() string {
	if c.Dest != nil {
		return fmt.Sprintf("%s = call %s(%v)", c.Dest, c.Function, c.Args)
	}
	return fmt.Sprintf("call %s(%v)", c.Function, c.Args)
}

func (c *Call) Operands() []*Value {
	operands := make([]*Value, 0, len(c.Args)+1)
	operands = append(operands, c.Function)
	operands = append(operands, c.Args...)
	return operands
}

func (c *Call) Result() *Value { return c.Dest }

// Return from function
// Format: return value

type Return struct {
	Value *Value // Can be nil for void return
}

func (r *Return) String() string {
	if r.Value != nil {
		return fmt.Sprintf("return %s", r.Value)
	}
	return "return"
}

func (r *Return) Operands() []*Value {
	if r.Value != nil {
		return []*Value{r.Value}
	}
	return nil
}

func (r *Return) Result() *Value { return nil }

// Phi node for SSA form
// Format: result = phi [value1, block1], [value2, block2], ...
//
// PHI NODES:
// In SSA form, when multiple control flow paths merge, we use phi nodes
// to represent which value a variable should have based on which path was taken.
//
// EXAMPLE:
//   if (x > 0) { y = 1; } else { y = 2; }
//   z = y;  // Which y? y1 or y2?
//
// In SSA:
//   if (x > 0) { y1 = 1; } else { y2 = 2; }
//   y3 = phi [y1, block1], [y2, block2]
//   z = y3;

type Phi struct {
	Dest    *Value
	Incomig []PhiIncoming
}

type PhiIncoming struct {
	Value *Value
	Block *BasicBlock
}

func (p *Phi) String() string {
	return fmt.Sprintf("%s = phi %v", p.Dest, p.Incomig)
}

func (p *Phi) Operands() []*Value {
	operands := make([]*Value, len(p.Incomig))
	for i, inc := range p.Incomig {
		operands[i] = inc.Value
	}
	return operands
}

func (p *Phi) Result() *Value { return p.Dest }

// Alloca allocates stack space
// Format: result = alloca type

type Alloca struct {
	Dest *Value
	Type types.Type
}

func (a *Alloca) String() string {
	return fmt.Sprintf("%s = alloca %s", a.Dest, a.Type)
}

func (a *Alloca) Operands() []*Value { return nil }
func (a *Alloca) Result() *Value     { return a.Dest }
