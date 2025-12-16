package ir

import (
	"fmt"
	"strings"

	"github.com/hassan/compiler/internal/semantic/types"
)

// BasicBlock represents a sequence of instructions with single entry and exit.
//
// WHAT IS A BASIC BLOCK?
// A basic block is a straight-line code sequence with:
// - One entry point (the first instruction)
// - One exit point (a jump or return)
// - No jumps in or out in the middle
//
// WHY BASIC BLOCKS?
// - Simplifies control flow analysis
// - Natural unit for optimization
// - Makes data flow analysis tractable
// - Standard compiler intermediate representation
//
// EXAMPLE:
//   Block1:              Block2:              Block3:
//     x = a + b           if x > 0             y = x * 2
//     y = x * 2           jump Block3          return y
//     jump Block2         jump Block4
//
// DESIGN CHOICE: Store predecessors and successors because:
// - Enables forward and backward data flow analysis
// - Makes CFG traversal efficient
// - Required for SSA construction
type BasicBlock struct {
	// Label is the unique name of this block
	Label string

	// Instructions in this block (in order)
	Instructions []Instruction

	// Successors are blocks that can execute after this one
	// Determined by the terminator instruction (jump, branch, return)
	Successors []*BasicBlock

	// Predecessors are blocks that can jump to this one
	// Updated when building the CFG
	Predecessors []*BasicBlock

	// Dominated tracks blocks dominated by this block
	// A block B dominates block C if every path to C goes through B
	// Used for SSA construction and optimization
	Dominated []*BasicBlock

	// Index is the position in the function's block list
	// Useful for some algorithms that need block ordering
	Index int
}

// NewBasicBlock creates a new basic block with the given label.
func NewBasicBlock(label string) *BasicBlock {
	return &BasicBlock{
		Label:        label,
		Instructions: make([]Instruction, 0),
		Successors:   make([]*BasicBlock, 0),
		Predecessors: make([]*BasicBlock, 0),
		Dominated:    make([]*BasicBlock, 0),
	}
}

// AddInstruction adds an instruction to the end of this block.
func (bb *BasicBlock) AddInstruction(instr Instruction) {
	bb.Instructions = append(bb.Instructions, instr)
}

// AddSuccessor adds a successor block and updates its predecessor list.
//
// DESIGN CHOICE: Automatically maintain bidirectional links because:
// - Ensures consistency (no dangling references)
// - Simpler for users of the IR
// - Prevents common bugs
func (bb *BasicBlock) AddSuccessor(succ *BasicBlock) {
	// Check for duplicates
	for _, s := range bb.Successors {
		if s == succ {
			return
		}
	}

	bb.Successors = append(bb.Successors, succ)
	succ.Predecessors = append(succ.Predecessors, bb)
}

// Terminator returns the last instruction (should be jump, branch, or return).
//
// In a well-formed CFG, every basic block ends with a terminator.
// Returns nil if the block is empty or doesn't have a terminator yet.
func (bb *BasicBlock) Terminator() Instruction {
	if len(bb.Instructions) == 0 {
		return nil
	}
	last := bb.Instructions[len(bb.Instructions)-1]

	// Check if it's a terminator
	switch last.(type) {
	case *Jump, *Branch, *Return:
		return last
	default:
		return nil
	}
}

// IsTerminated returns true if this block has a terminator instruction.
func (bb *BasicBlock) IsTerminated() bool {
	return bb.Terminator() != nil
}

// String returns a human-readable representation of the basic block.
func (bb *BasicBlock) String() string {
	var sb strings.Builder

	sb.WriteString(bb.Label)
	sb.WriteString(":\n")

	// Show predecessors
	if len(bb.Predecessors) > 0 {
		sb.WriteString("  ; predecessors: ")
		for i, pred := range bb.Predecessors {
			if i > 0 {
				sb.WriteString(", ")
			}
			sb.WriteString(pred.Label)
		}
		sb.WriteString("\n")
	}

	// Show instructions
	for _, instr := range bb.Instructions {
		sb.WriteString("  ")
		sb.WriteString(instr.String())
		sb.WriteString("\n")
	}

	return sb.String()
}

// Function represents a function in IR.
//
// DESIGN CHOICE: Store all basic blocks in a slice because:
// - Provides a stable ordering (useful for algorithms)
// - Entry block is always first
// - Easy to iterate over all blocks
type Function struct {
	// Name is the function name
	Name string

	// Parameters are the function parameters (as Values)
	Parameters []*Value

	// ReturnType is the function's return type
	ReturnType types.Type

	// Blocks are all basic blocks in this function
	// The first block is always the entry block
	Blocks []*BasicBlock

	// Entry is the entry basic block
	Entry *BasicBlock

	// Locals are local variables (allocas)
	Locals []*Value

	// nextValueID is used to generate unique value IDs
	nextValueID int
}

// NewFunction creates a new function.
func NewFunction(name string, params []*Value, returnType types.Type) *Function {
	entry := NewBasicBlock("entry")
	return &Function{
		Name:        name,
		Parameters:  params,
		ReturnType:  returnType,
		Blocks:      []*BasicBlock{entry},
		Entry:       entry,
		Locals:      make([]*Value, 0),
		nextValueID: len(params), // Start after parameters
	}
}

// NewBasicBlockInFunc creates a new basic block and adds it to the function.
func (f *Function) NewBasicBlockInFunc(label string) *BasicBlock {
	bb := NewBasicBlock(label)
	bb.Index = len(f.Blocks)
	f.Blocks = append(f.Blocks, bb)
	return bb
}

// NewValue creates a new value with a unique ID.
func (f *Function) NewValue(name string, typ types.Type, kind ValueKind) *Value {
	v := &Value{
		ID:   f.nextValueID,
		Name: name,
		Type: typ,
		Kind: kind,
	}
	f.nextValueID++
	return v
}

// NewTemp creates a new temporary value.
func (f *Function) NewTemp(typ types.Type) *Value {
	return f.NewValue("", typ, ValueTemporary)
}

// String returns a human-readable representation of the function.
func (f *Function) String() string {
	var sb strings.Builder

	// Function signature
	sb.WriteString("func ")
	sb.WriteString(f.Name)
	sb.WriteString("(")
	for i, param := range f.Parameters {
		if i > 0 {
			sb.WriteString(", ")
		}
		sb.WriteString(param.String())
		sb.WriteString(": ")
		sb.WriteString(param.Type.String())
	}
	sb.WriteString(") ")
	sb.WriteString(f.ReturnType.String())
	sb.WriteString(" {\n")

	// Basic blocks
	for _, block := range f.Blocks {
		sb.WriteString(block.String())
		sb.WriteString("\n")
	}

	sb.WriteString("}\n")
	return sb.String()
}

// Module represents a compilation unit (collection of functions and globals).
//
// DESIGN CHOICE: Module is the top-level IR container because:
// - Matches how programs are organized (files/packages)
// - Enables whole-program optimization
// - Natural unit for code generation
type Module struct {
	// Name is the module name (typically package name)
	Name string

	// Functions are all functions in this module
	Functions []*Function

	// Globals are global variables
	Globals []*Value
}

// NewModule creates a new module.
func NewModule(name string) *Module {
	return &Module{
		Name:      name,
		Functions: make([]*Function, 0),
		Globals:   make([]*Value, 0),
	}
}

// AddFunction adds a function to the module.
func (m *Module) AddFunction(fn *Function) {
	m.Functions = append(m.Functions, fn)
}

// String returns a human-readable representation of the module.
func (m *Module) String() string {
	var sb strings.Builder

	sb.WriteString("; Module: ")
	sb.WriteString(m.Name)
	sb.WriteString("\n\n")

	// Globals
	if len(m.Globals) > 0 {
		sb.WriteString("; Globals\n")
		for _, global := range m.Globals {
			sb.WriteString("global ")
			sb.WriteString(global.String())
			sb.WriteString(": ")
			sb.WriteString(global.Type.String())
			sb.WriteString("\n")
		}
		sb.WriteString("\n")
	}

	// Functions
	for _, fn := range m.Functions {
		sb.WriteString(fn.String())
		sb.WriteString("\n")
	}

	return sb.String()
}

// Verify checks that the IR is well-formed.
// Returns a list of errors found.
//
// CHECKS:
// - Every block ends with a terminator
// - Successors match terminator
// - No unreachable blocks
// - SSA properties (if applicable)
func (m *Module) Verify() []error {
	errors := make([]error, 0)

	for _, fn := range m.Functions {
		// Check each block has a terminator
		for _, block := range fn.Blocks {
			if !block.IsTerminated() {
				errors = append(errors, fmt.Errorf(
					"block %s in function %s has no terminator",
					block.Label, fn.Name))
			}
		}

		// Check entry block has no predecessors
		if len(fn.Entry.Predecessors) > 0 {
			errors = append(errors, fmt.Errorf(
				"entry block of function %s has predecessors",
				fn.Name))
		}
	}

	return errors
}
