package optimizer

import (
	"github.com/hassan/compiler/internal/ir"
	"github.com/hassan/compiler/internal/semantic/types"
)

// ConstantFoldingPass performs constant folding optimization.
//
// WHAT IS CONSTANT FOLDING?
// Constant folding evaluates constant expressions at compile time rather than runtime.
//
// EXAMPLE:
//   Before:  t1 = 2 + 3
//            t2 = t1 * 4
//   After:   t1 = const(5)
//            t2 = const(20)
//
// WHY CONSTANT FOLDING?
// 1. Reduces runtime computation
// 2. Enables further optimizations (dead code elimination)
// 3. Can reveal optimization opportunities
// 4. Standard optimization in all compilers
//
// DESIGN CHOICE: Process instructions in order because:
// - Simple single-pass algorithm
// - Dependencies are guaranteed to be defined before use
// - Can fold chains of constant operations
type ConstantFoldingPass struct{}

// Name returns the name of this optimization pass.
func (c *ConstantFoldingPass) Name() string {
	return "ConstantFolding"
}

// Run executes constant folding on the given function.
//
// ALGORITHM:
// 1. Build a map of values to their constant values (if known)
// 2. For each instruction, try to fold using the constant map
// 3. Update the constant map with newly folded values
// 4. Replace instructions with simpler versions
//
// DESIGN CHOICE: Single pass with constant propagation because:
// - Handles chains of constant operations in one pass
// - Avoids infinite loops
// - More efficient than iterating
func (c *ConstantFoldingPass) Run(fn *ir.Function) error {
	// Map from values to their constant values
	constants := make(map[*ir.Value]interface{})

	// First pass: identify all existing constants
	for _, block := range fn.Blocks {
		for _, instr := range block.Instructions {
			// Check for Copy instructions from constants
			if copy, ok := instr.(*ir.Copy); ok {
				if copy.Value.IsConstant() {
					constants[copy.Dest] = copy.Value.Constant
				}
			}
		}
	}

	// Second pass: fold instructions
	for _, block := range fn.Blocks {
		for i, instr := range block.Instructions {
			folded := c.foldInstructionWithConstants(instr, constants)
			if folded != nil {
				block.Instructions[i] = folded

				// Update constants map with newly folded value
				if copy, ok := folded.(*ir.Copy); ok {
					if copy.Value.IsConstant() {
						constants[copy.Dest] = copy.Value.Constant
					}
				}
			}
		}
	}

	return nil
}

// foldInstructionWithConstants attempts to fold using a constant map.
// Returns a replacement instruction if folding succeeded, nil otherwise.
func (c *ConstantFoldingPass) foldInstructionWithConstants(instr ir.Instruction, constants map[*ir.Value]interface{}) ir.Instruction {
	switch i := instr.(type) {
	case *ir.BinaryOp:
		return c.foldBinaryOpWithConstants(i, constants)
	case *ir.UnaryOp:
		return c.foldUnaryOpWithConstants(i, constants)
	default:
		return nil
	}
}

// getConstantValue returns the constant value of a Value, looking in the constants map
func (c *ConstantFoldingPass) getConstantValue(v *ir.Value, constants map[*ir.Value]interface{}) (interface{}, bool) {
	if v.IsConstant() {
		return v.Constant, true
	}
	if constVal, ok := constants[v]; ok {
		return constVal, true
	}
	return nil, false
}

// foldBinaryOpWithConstants attempts to fold a binary operation using constant map.
//
// IMPLEMENTATION NOTE:
// We only fold integer operations for now. Floating point folding
// is tricky due to precision and rounding modes.
func (c *ConstantFoldingPass) foldBinaryOpWithConstants(op *ir.BinaryOp, constants map[*ir.Value]interface{}) ir.Instruction {
	// Check if both operands are constants (directly or via the map)
	leftConst, leftOk := c.getConstantValue(op.Left, constants)
	rightConst, rightOk := c.getConstantValue(op.Right, constants)

	if !leftOk || !rightOk {
		return nil
	}

	// Only fold integer constants for now
	leftVal, leftOk := leftConst.(int64)
	rightVal, rightOk := rightConst.(int64)

	if !leftOk || !rightOk {
		return nil
	}

	// Evaluate the operation
	var result int64
	var isValid bool = true

	switch op.Op {
	case ir.OpAdd:
		result = leftVal + rightVal
	case ir.OpSub:
		result = leftVal - rightVal
	case ir.OpMul:
		result = leftVal * rightVal
	case ir.OpDiv:
		// Don't fold division by zero
		if rightVal == 0 {
			return nil
		}
		result = leftVal / rightVal
	case ir.OpMod:
		// Don't fold modulo by zero
		if rightVal == 0 {
			return nil
		}
		result = leftVal % rightVal

	// Comparison operations return bool
	case ir.OpEq:
		if leftVal == rightVal {
			return c.createBoolCopy(op.Dest, true)
		}
		return c.createBoolCopy(op.Dest, false)
	case ir.OpNeq:
		if leftVal != rightVal {
			return c.createBoolCopy(op.Dest, true)
		}
		return c.createBoolCopy(op.Dest, false)
	case ir.OpLt:
		if leftVal < rightVal {
			return c.createBoolCopy(op.Dest, true)
		}
		return c.createBoolCopy(op.Dest, false)
	case ir.OpLe:
		if leftVal <= rightVal {
			return c.createBoolCopy(op.Dest, true)
		}
		return c.createBoolCopy(op.Dest, false)
	case ir.OpGt:
		if leftVal > rightVal {
			return c.createBoolCopy(op.Dest, true)
		}
		return c.createBoolCopy(op.Dest, false)
	case ir.OpGe:
		if leftVal >= rightVal {
			return c.createBoolCopy(op.Dest, true)
		}
		return c.createBoolCopy(op.Dest, false)

	// Bitwise operations
	case ir.OpBitAnd:
		result = leftVal & rightVal
	case ir.OpBitOr:
		result = leftVal | rightVal
	case ir.OpBitXor:
		result = leftVal ^ rightVal
	case ir.OpShl:
		result = leftVal << uint(rightVal)
	case ir.OpShr:
		result = leftVal >> uint(rightVal)

	default:
		isValid = false
	}

	if !isValid {
		return nil
	}

	// Create a constant value
	constValue := &ir.Value{
		ID:       -1, // Temporary ID
		Type:     types.Int,
		Kind:     ir.ValueConstant,
		Constant: result,
	}

	// Replace with a copy instruction
	return &ir.Copy{
		Dest:  op.Dest,
		Value: constValue,
	}
}

// foldUnaryOpWithConstants attempts to fold a unary operation using constant map.
func (c *ConstantFoldingPass) foldUnaryOpWithConstants(op *ir.UnaryOp, constants map[*ir.Value]interface{}) ir.Instruction {
	// Check if operand is constant (directly or via the map)
	operandConst, ok := c.getConstantValue(op.Operand, constants)
	if !ok {
		return nil
	}

	// Handle integer constants
	if intVal, ok := operandConst.(int64); ok {
		var result int64

		switch op.Op {
		case ir.OpNeg:
			result = -intVal
		case ir.OpBitNot:
			result = ^intVal
		default:
			return nil
		}

		constValue := &ir.Value{
			ID:       -1,
			Type:     types.Int,
			Kind:     ir.ValueConstant,
			Constant: result,
		}

		return &ir.Copy{
			Dest:  op.Dest,
			Value: constValue,
		}
	}

	// Handle boolean constants
	if boolVal, ok := operandConst.(bool); ok {
		if op.Op == ir.OpNot {
			return c.createBoolCopy(op.Dest, !boolVal)
		}
	}

	return nil
}

// createBoolCopy creates a Copy instruction with a boolean constant.
func (c *ConstantFoldingPass) createBoolCopy(dest *ir.Value, value bool) ir.Instruction {
	constValue := &ir.Value{
		ID:       -1,
		Type:     types.Bool,
		Kind:     ir.ValueConstant,
		Constant: value,
	}

	return &ir.Copy{
		Dest:  dest,
		Value: constValue,
	}
}
