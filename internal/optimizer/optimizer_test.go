package optimizer

import (
	"testing"

	"github.com/hassan/compiler/internal/ir"
	"github.com/hassan/compiler/internal/semantic/types"
)

// TestConstantFolding tests the constant folding pass
func TestConstantFolding(t *testing.T) {
	tests := []struct {
		name     string
		setup    func() *ir.Function
		validate func(*testing.T, *ir.Function)
	}{
		{
			name: "fold simple addition",
			setup: func() *ir.Function {
				fn := &ir.Function{
					Name:       "test",
					Parameters: nil,
					ReturnType: types.Int,
					Blocks:     make([]*ir.BasicBlock, 0),
				}

				entry := &ir.BasicBlock{
					Label:        "entry",
					Instructions: make([]ir.Instruction, 0),
				}

				// t1 = 2 + 3
				dest := &ir.Value{ID: 1, Type: types.Int}
				left := &ir.Value{ID: -1, Type: types.Int, Kind: ir.ValueConstant, Constant: int64(2)}
				right := &ir.Value{ID: -1, Type: types.Int, Kind: ir.ValueConstant, Constant: int64(3)}

				binop := &ir.BinaryOp{
					Op:    ir.OpAdd,
					Dest:  dest,
					Left:  left,
					Right: right,
				}

				entry.Instructions = append(entry.Instructions, binop)
				fn.Blocks = append(fn.Blocks, entry)
				fn.Entry = entry

				return fn
			},
			validate: func(t *testing.T, fn *ir.Function) {
				// After constant folding, should be a Copy instruction
				if len(fn.Blocks) == 0 || len(fn.Blocks[0].Instructions) == 0 {
					t.Fatal("expected at least one instruction")
				}

				instr := fn.Blocks[0].Instructions[0]
				copy, ok := instr.(*ir.Copy)
				if !ok {
					t.Fatalf("expected Copy instruction, got %T", instr)
				}

				// Check that the value is the constant 5
				if !copy.Value.IsConstant() {
					t.Error("expected constant value")
				}

				if val, ok := copy.Value.Constant.(int64); !ok || val != 5 {
					t.Errorf("expected constant 5, got %v", copy.Value.Constant)
				}
			},
		},
		{
			name: "fold multiplication",
			setup: func() *ir.Function {
				fn := &ir.Function{
					Name:       "test",
					Parameters: nil,
					ReturnType: types.Int,
					Blocks:     make([]*ir.BasicBlock, 0),
				}

				entry := &ir.BasicBlock{
					Label:        "entry",
					Instructions: make([]ir.Instruction, 0),
				}

				// t1 = 7 * 8
				dest := &ir.Value{ID: 1, Type: types.Int}
				left := &ir.Value{ID: -1, Type: types.Int, Kind: ir.ValueConstant, Constant: int64(7)}
				right := &ir.Value{ID: -1, Type: types.Int, Kind: ir.ValueConstant, Constant: int64(8)}

				binop := &ir.BinaryOp{
					Op:    ir.OpMul,
					Dest:  dest,
					Left:  left,
					Right: right,
				}

				entry.Instructions = append(entry.Instructions, binop)
				fn.Blocks = append(fn.Blocks, entry)
				fn.Entry = entry

				return fn
			},
			validate: func(t *testing.T, fn *ir.Function) {
				instr := fn.Blocks[0].Instructions[0]
				copy, ok := instr.(*ir.Copy)
				if !ok {
					t.Fatalf("expected Copy instruction, got %T", instr)
				}

				if val, ok := copy.Value.Constant.(int64); !ok || val != 56 {
					t.Errorf("expected constant 56, got %v", copy.Value.Constant)
				}
			},
		},
		{
			name: "fold comparison",
			setup: func() *ir.Function {
				fn := &ir.Function{
					Name:       "test",
					Parameters: nil,
					ReturnType: types.Bool,
					Blocks:     make([]*ir.BasicBlock, 0),
				}

				entry := &ir.BasicBlock{
					Label:        "entry",
					Instructions: make([]ir.Instruction, 0),
				}

				// t1 = 5 > 3
				dest := &ir.Value{ID: 1, Type: types.Bool}
				left := &ir.Value{ID: -1, Type: types.Int, Kind: ir.ValueConstant, Constant: int64(5)}
				right := &ir.Value{ID: -1, Type: types.Int, Kind: ir.ValueConstant, Constant: int64(3)}

				binop := &ir.BinaryOp{
					Op:    ir.OpGt,
					Dest:  dest,
					Left:  left,
					Right: right,
				}

				entry.Instructions = append(entry.Instructions, binop)
				fn.Blocks = append(fn.Blocks, entry)
				fn.Entry = entry

				return fn
			},
			validate: func(t *testing.T, fn *ir.Function) {
				instr := fn.Blocks[0].Instructions[0]
				copy, ok := instr.(*ir.Copy)
				if !ok {
					t.Fatalf("expected Copy instruction, got %T", instr)
				}

				if val, ok := copy.Value.Constant.(bool); !ok || val != true {
					t.Errorf("expected constant true, got %v", copy.Value.Constant)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fn := tt.setup()
			pass := &ConstantFoldingPass{}

			if err := pass.Run(fn); err != nil {
				t.Fatalf("constant folding failed: %v", err)
			}

			tt.validate(t, fn)
		})
	}
}

// TestDeadCodeElimination tests the dead code elimination pass
func TestDeadCodeElimination(t *testing.T) {
	tests := []struct {
		name     string
		setup    func() *ir.Function
		validate func(*testing.T, *ir.Function)
	}{
		{
			name: "remove unused computation",
			setup: func() *ir.Function {
				fn := &ir.Function{
					Name:       "test",
					Parameters: nil,
					ReturnType: types.Int,
					Blocks:     make([]*ir.BasicBlock, 0),
				}

				entry := &ir.BasicBlock{
					Label:        "entry",
					Instructions: make([]ir.Instruction, 0),
				}

				// t1 = 2 + 3 (unused)
				t1 := &ir.Value{ID: 1, Type: types.Int}
				binop := &ir.BinaryOp{
					Op:    ir.OpAdd,
					Dest:  t1,
					Left:  &ir.Value{ID: -1, Type: types.Int, Kind: ir.ValueConstant, Constant: int64(2)},
					Right: &ir.Value{ID: -1, Type: types.Int, Kind: ir.ValueConstant, Constant: int64(3)},
				}

				// return 42
				ret := &ir.Return{
					Value: &ir.Value{ID: -1, Type: types.Int, Kind: ir.ValueConstant, Constant: int64(42)},
				}

				entry.Instructions = append(entry.Instructions, binop, ret)
				fn.Blocks = append(fn.Blocks, entry)
				fn.Entry = entry

				return fn
			},
			validate: func(t *testing.T, fn *ir.Function) {
				// Should only have the return instruction left
				if len(fn.Blocks[0].Instructions) != 1 {
					t.Errorf("expected 1 instruction, got %d", len(fn.Blocks[0].Instructions))
				}

				if _, ok := fn.Blocks[0].Instructions[0].(*ir.Return); !ok {
					t.Error("expected only Return instruction to remain")
				}
			},
		},
		{
			name: "keep used computation",
			setup: func() *ir.Function {
				fn := &ir.Function{
					Name:       "test",
					Parameters: nil,
					ReturnType: types.Int,
					Blocks:     make([]*ir.BasicBlock, 0),
				}

				entry := &ir.BasicBlock{
					Label:        "entry",
					Instructions: make([]ir.Instruction, 0),
				}

				// t1 = 2 + 3
				t1 := &ir.Value{ID: 1, Type: types.Int}
				binop := &ir.BinaryOp{
					Op:    ir.OpAdd,
					Dest:  t1,
					Left:  &ir.Value{ID: -1, Type: types.Int, Kind: ir.ValueConstant, Constant: int64(2)},
					Right: &ir.Value{ID: -1, Type: types.Int, Kind: ir.ValueConstant, Constant: int64(3)},
				}

				// return t1 (uses t1)
				ret := &ir.Return{Value: t1}

				entry.Instructions = append(entry.Instructions, binop, ret)
				fn.Blocks = append(fn.Blocks, entry)
				fn.Entry = entry

				return fn
			},
			validate: func(t *testing.T, fn *ir.Function) {
				// Should keep both instructions
				if len(fn.Blocks[0].Instructions) != 2 {
					t.Errorf("expected 2 instructions, got %d", len(fn.Blocks[0].Instructions))
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fn := tt.setup()
			pass := &DeadCodeEliminationPass{}

			if err := pass.Run(fn); err != nil {
				t.Fatalf("dead code elimination failed: %v", err)
			}

			tt.validate(t, fn)
		})
	}
}

// TestOptimizerIntegration tests the full optimizer with multiple passes
func TestOptimizerIntegration(t *testing.T) {
	// Create a function with constant folding opportunity and dead code
	fn := &ir.Function{
		Name:       "test",
		Parameters: nil,
		ReturnType: types.Int,
		Blocks:     make([]*ir.BasicBlock, 0),
	}

	entry := &ir.BasicBlock{
		Label:        "entry",
		Instructions: make([]ir.Instruction, 0),
	}

	// t1 = 2 + 3 (will fold to 5, then be marked dead)
	t1 := &ir.Value{ID: 1, Type: types.Int}
	binop1 := &ir.BinaryOp{
		Op:    ir.OpAdd,
		Dest:  t1,
		Left:  &ir.Value{ID: -1, Type: types.Int, Kind: ir.ValueConstant, Constant: int64(2)},
		Right: &ir.Value{ID: -1, Type: types.Int, Kind: ir.ValueConstant, Constant: int64(3)},
	}

	// t2 = 4 * 5 (will fold to 20)
	t2 := &ir.Value{ID: 2, Type: types.Int}
	binop2 := &ir.BinaryOp{
		Op:    ir.OpMul,
		Dest:  t2,
		Left:  &ir.Value{ID: -1, Type: types.Int, Kind: ir.ValueConstant, Constant: int64(4)},
		Right: &ir.Value{ID: -1, Type: types.Int, Kind: ir.ValueConstant, Constant: int64(5)},
	}

	// return t2 (only t2 is used)
	ret := &ir.Return{Value: t2}

	entry.Instructions = append(entry.Instructions, binop1, binop2, ret)
	fn.Blocks = append(fn.Blocks, entry)
	fn.Entry = entry

	// Run optimizer
	opt := NewOptimizer()
	if err := opt.OptimizeFunction(fn); err != nil {
		t.Fatalf("optimization failed: %v", err)
	}

	// Verify results
	// Should have: Copy(t2, const(20)), Return(t2)
	// t1 computation should be eliminated
	instructions := fn.Blocks[0].Instructions

	if len(instructions) != 2 {
		t.Errorf("expected 2 instructions after optimization, got %d", len(instructions))
		for i, instr := range instructions {
			t.Logf("  %d: %T", i, instr)
		}
	}

	// First should be Copy
	if _, ok := instructions[0].(*ir.Copy); !ok {
		t.Errorf("expected first instruction to be Copy, got %T", instructions[0])
	}

	// Second should be Return
	if _, ok := instructions[1].(*ir.Return); !ok {
		t.Errorf("expected second instruction to be Return, got %T", instructions[1])
	}
}
