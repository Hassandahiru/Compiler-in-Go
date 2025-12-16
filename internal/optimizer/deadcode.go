package optimizer

import (
	"github.com/hassan/compiler/internal/ir"
)

// DeadCodeEliminationPass removes unused instructions and unreachable code.
//
// WHAT IS DEAD CODE?
// Dead code is code that either:
// 1. Computes values that are never used
// 2. Is unreachable (no control flow path reaches it)
//
// EXAMPLE 1 - Unused computation:
//   Before:  t1 = 2 + 3    // t1 is never used
//            t2 = 4 * 5
//            return t2
//   After:   t2 = 4 * 5
//            return t2
//
// EXAMPLE 2 - Unreachable code:
//   Before:  return x
//            t1 = 1         // Never reached
//   After:   return x
//
// WHY ELIMINATE DEAD CODE?
// 1. Reduces code size
// 2. Improves runtime performance
// 3. Simplifies further optimization
// 4. Often created by other optimizations (constant folding, inlining)
//
// DESIGN CHOICE: Two-pass algorithm because:
// - First pass: mark all used values (backward analysis)
// - Second pass: remove unmarked instructions (forward sweep)
// - Simple and correct
// - Standard textbook algorithm
type DeadCodeEliminationPass struct{}

// Name returns the name of this optimization pass.
func (d *DeadCodeEliminationPass) Name() string {
	return "DeadCodeElimination"
}

// Run executes dead code elimination on the given function.
//
// ALGORITHM:
// 1. Mark all "critical" instructions (those with side effects)
// 2. Recursively mark all values used by critical instructions
// 3. Remove unmarked instructions
// 4. Remove unreachable blocks
func (d *DeadCodeEliminationPass) Run(fn *ir.Function) error {
	modified := true

	// Keep running until no changes (handles transitive dependencies)
	for modified {
		modified = false

		// Pass 1: Mark used values
		usedValues := d.markUsedValues(fn)

		// Pass 2: Remove unused instructions
		if d.removeUnusedInstructions(fn, usedValues) {
			modified = true
		}

		// Pass 3: Remove unreachable blocks
		if d.removeUnreachableBlocks(fn) {
			modified = true
		}
	}

	return nil
}

// markUsedValues identifies all values that are actually used.
//
// DESIGN CHOICE: Use map for O(1) lookup when removing instructions.
//
// CRITICAL INSTRUCTIONS (must be kept):
// - Store operations (modify memory)
// - Function calls (may have side effects)
// - Return statements (define function behavior)
// - Branches/jumps (affect control flow)
func (d *DeadCodeEliminationPass) markUsedValues(fn *ir.Function) map[*ir.Value]bool {
	used := make(map[*ir.Value]bool)

	// Process all blocks
	for _, block := range fn.Blocks {
		for _, instr := range block.Instructions {
			// Check if this instruction is critical
			if d.isCritical(instr) {
				// Mark all operands as used
				for _, operand := range instr.Operands() {
					d.markValue(operand, used, fn)
				}
			}
		}
	}

	return used
}

// isCritical returns true if an instruction has side effects and must be kept.
//
// DESIGN CHOICE: Conservative approach - if we're unsure, keep it.
// Better to keep unnecessary code than break the program.
func (d *DeadCodeEliminationPass) isCritical(instr ir.Instruction) bool {
	switch instr.(type) {
	case *ir.Store:
		// Stores modify memory - critical
		return true
	case *ir.Call:
		// Function calls may have side effects - critical
		return true
	case *ir.Return:
		// Returns define function behavior - critical
		return true
	case *ir.Branch:
		// Branches affect control flow - critical
		return true
	case *ir.Jump:
		// Jumps affect control flow - critical
		return true
	default:
		// Pure computation - only keep if result is used
		return false
	}
}

// markValue recursively marks a value and all values it depends on as used.
//
// DESIGN CHOICE: Recursive algorithm because:
// - Natural way to follow def-use chains
// - Simple to implement
// - Depth is bounded by function size
func (d *DeadCodeEliminationPass) markValue(v *ir.Value, used map[*ir.Value]bool, fn *ir.Function) {
	if v == nil {
		return
	}

	// Already marked?
	if used[v] {
		return
	}

	// Constants are always available, no need to mark
	if v.IsConstant() {
		return
	}

	// Mark this value
	used[v] = true

	// Find the instruction that defines this value and mark its operands
	for _, block := range fn.Blocks {
		for _, instr := range block.Instructions {
			if instr.Result() == v {
				// Mark all operands
				for _, operand := range instr.Operands() {
					d.markValue(operand, used, fn)
				}
				return
			}
		}
	}
}

// removeUnusedInstructions removes instructions whose results are not used.
// Returns true if any instructions were removed.
//
// DESIGN CHOICE: Create new slice rather than modify in place because:
// - Avoids index shifting bugs
// - Cleaner code
// - Performance difference is negligible for typical function sizes
func (d *DeadCodeEliminationPass) removeUnusedInstructions(fn *ir.Function, used map[*ir.Value]bool) bool {
	modified := false

	for _, block := range fn.Blocks {
		newInstructions := make([]ir.Instruction, 0, len(block.Instructions))

		for _, instr := range block.Instructions {
			// Keep critical instructions
			if d.isCritical(instr) {
				newInstructions = append(newInstructions, instr)
				continue
			}

			// Keep if result is used
			result := instr.Result()
			if result != nil && used[result] {
				newInstructions = append(newInstructions, instr)
				continue
			}

			// Otherwise, this instruction is dead - remove it
			modified = true
		}

		block.Instructions = newInstructions
	}

	return modified
}

// removeUnreachableBlocks removes basic blocks that cannot be reached.
// Returns true if any blocks were removed.
//
// ALGORITHM:
// 1. Start from entry block
// 2. Do a graph traversal (DFS/BFS) following successor edges
// 3. Remove blocks not visited
//
// DESIGN CHOICE: Use DFS with explicit stack because:
// - Avoids recursion depth limits
// - Simple to implement
// - Performance is fine for typical CFGs
func (d *DeadCodeEliminationPass) removeUnreachableBlocks(fn *ir.Function) bool {
	// Mark reachable blocks using DFS
	reachable := make(map[*ir.BasicBlock]bool)
	stack := []*ir.BasicBlock{fn.Entry}

	for len(stack) > 0 {
		// Pop from stack
		current := stack[len(stack)-1]
		stack = stack[:len(stack)-1]

		// Already visited?
		if reachable[current] {
			continue
		}

		// Mark as reachable
		reachable[current] = true

		// Add successors to stack
		for _, succ := range current.Successors {
			if !reachable[succ] {
				stack = append(stack, succ)
			}
		}
	}

	// Remove unreachable blocks
	newBlocks := make([]*ir.BasicBlock, 0, len(fn.Blocks))
	modified := false

	for _, block := range fn.Blocks {
		if reachable[block] {
			newBlocks = append(newBlocks, block)
		} else {
			modified = true
		}
	}

	if modified {
		fn.Blocks = newBlocks

		// Update block indices
		for i, block := range fn.Blocks {
			block.Index = i
		}
	}

	return modified
}
