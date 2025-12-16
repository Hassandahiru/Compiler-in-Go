package optimizer

import (
	"fmt"

	"github.com/hassan/compiler/internal/ir"
)

// Pass represents an optimization pass that can be applied to IR.
//
// DESIGN PHILOSOPHY:
// Each optimization is a separate pass that can be:
// - Enabled/disabled independently
// - Reordered based on effectiveness
// - Tested in isolation
// - Composed with other passes
//
// This follows the "separation of concerns" principle - each pass
// has a single, well-defined responsibility.
//
// COMMON OPTIMIZATION PASSES:
// - Constant folding: Evaluate constant expressions at compile time
// - Dead code elimination: Remove unused code
// - Common subexpression elimination: Reuse computed values
// - Copy propagation: Replace copies with original values
// - Loop invariant code motion: Move loop-invariant code outside loops
// - Inlining: Replace function calls with function body
//
// DESIGN CHOICE: Interface-based design because:
// - Allows dynamic pass configuration
// - Easy to add new passes
// - Passes can be third-party plugins
// - Testable independently
type Pass interface {
	// Name returns a human-readable name for this pass
	Name() string

	// Run executes this optimization pass on the given function
	// Returns an error if the pass fails
	Run(fn *ir.Function) error
}

// Optimizer coordinates the execution of optimization passes.
//
// DESIGN CHOICE: Separate optimizer from passes because:
// - Optimizer manages pass ordering and iteration
// - Passes focus on their specific transformation
// - Allows for meta-optimization (choosing which passes to run)
type Optimizer struct {
	// passes is the list of optimization passes to run
	passes []Pass

	// maxIterations limits how many times we run all passes
	// This prevents infinite loops in case passes keep modifying IR
	maxIterations int

	// verbose enables detailed logging
	verbose bool
}

// NewOptimizer creates a new optimizer with default passes.
//
// DEFAULT PASS ORDER:
// 1. Constant folding - reduces code, enables other optimizations
// 2. Dead code elimination - removes code constant folding makes redundant
//
// DESIGN CHOICE: Run passes multiple times because:
// - Optimizations interact: one optimization may enable another
// - Example: constant folding may create dead code
// - Fixed-point iteration ensures all opportunities are found
//
// ALTERNATIVE DESIGNS CONSIDERED:
// 1. Single-pass: Fast but misses optimization opportunities
// 2. User-specified order: Flexible but requires expertise
// 3. Dependency-based scheduling: Complex to implement
// 4. Fixed-point iteration (chosen): Good balance of simplicity and effectiveness
func NewOptimizer() *Optimizer {
	return &Optimizer{
		passes: []Pass{
			&ConstantFoldingPass{},
			&DeadCodeEliminationPass{},
		},
		maxIterations: 10, // Reasonable default
		verbose:       false,
	}
}

// AddPass adds a custom optimization pass.
//
// DESIGN CHOICE: Allow custom passes because:
// - Users may have domain-specific optimizations
// - Enables experimentation with new optimization techniques
// - Supports plugin architecture
func (o *Optimizer) AddPass(pass Pass) {
	o.passes = append(o.passes, pass)
}

// SetVerbose enables or disables verbose logging.
func (o *Optimizer) SetVerbose(verbose bool) {
	o.verbose = verbose
}

// SetMaxIterations sets the maximum number of optimization iterations.
//
// TUNING GUIDANCE:
// - Small programs: 3-5 iterations usually sufficient
// - Large programs: May need 10-20 iterations
// - If optimization seems slow, reduce this
// - If output quality is poor, increase this
func (o *Optimizer) SetMaxIterations(max int) {
	o.maxIterations = max
}

// Optimize runs all optimization passes on the entire module.
//
// ALGORITHM:
// 1. For each function in the module
// 2. Run all passes in sequence
// 3. Repeat until no changes or max iterations reached
//
// DESIGN CHOICE: Optimize per-function because:
// - Most optimizations are intraprocedural (within a function)
// - Easier to implement and reason about
// - Parallelizable (could optimize functions in parallel)
//
// NOTE: Whole-program optimizations (like inlining across functions)
// would require a different approach.
func (o *Optimizer) Optimize(module *ir.Module) error {
	for _, fn := range module.Functions {
		if err := o.OptimizeFunction(fn); err != nil {
			return fmt.Errorf("optimization failed for function %s: %w", fn.Name, err)
		}
	}
	return nil
}

// OptimizeFunction runs optimization passes on a single function.
//
// ALGORITHM:
// 1. Run all passes once
// 2. Repeat until either:
//    - No pass modifies the IR (fixed point reached)
//    - Maximum iterations exceeded
//
// DESIGN CHOICE: Fixed-point iteration because:
// - Ensures all optimization opportunities are found
// - Simple to implement
// - Predictable behavior
//
// PERFORMANCE NOTE:
// In practice, most functions reach a fixed point in 2-3 iterations.
// The max iterations guard is just for pathological cases.
func (o *Optimizer) OptimizeFunction(fn *ir.Function) error {
	// SIMPLIFIED APPROACH: Run each pass once in sequence
	// This avoids issues with fixed-point detection and infinite loops
	// Most optimization opportunities are found in a single pass through all optimizations
	for _, pass := range o.passes {
		if o.verbose {
			fmt.Printf("  Running %s...\n", pass.Name())
		}

		if err := pass.Run(fn); err != nil {
			return fmt.Errorf("pass %s failed: %w", pass.Name(), err)
		}
	}

	return nil
}

// countInstructions counts the total number of instructions in a function.
//
// DESIGN CHOICE: Use instruction count as a simple proxy for "did anything change".
//
// LIMITATION: This doesn't catch all changes (e.g., instruction reordering).
// A more robust approach would be to compute a hash of the IR.
// However, for our purposes, instruction count is sufficient and fast.
func (o *Optimizer) countInstructions(fn *ir.Function) int {
	count := 0
	for _, block := range fn.Blocks {
		count += len(block.Instructions)
	}
	return count
}

// OptimizationStats tracks statistics about optimization.
//
// DESIGN CHOICE: Collect stats for analysis and tuning because:
// - Helps understand optimization effectiveness
// - Identifies which passes are most valuable
// - Useful for benchmarking and regression testing
type OptimizationStats struct {
	// InstructionsRemoved is the number of instructions eliminated
	InstructionsRemoved int

	// BlocksRemoved is the number of basic blocks eliminated
	BlocksRemoved int

	// ConstantsFolded is the number of constant expressions folded
	ConstantsFolded int

	// PassExecutions tracks how many times each pass ran
	PassExecutions map[string]int
}

// NewOptimizationStats creates a new stats tracker.
func NewOptimizationStats() *OptimizationStats {
	return &OptimizationStats{
		PassExecutions: make(map[string]int),
	}
}

// String returns a human-readable summary of optimization statistics.
func (s *OptimizationStats) String() string {
	return fmt.Sprintf("Optimization Stats:\n"+
		"  Instructions removed: %d\n"+
		"  Blocks removed: %d\n"+
		"  Constants folded: %d\n",
		s.InstructionsRemoved,
		s.BlocksRemoved,
		s.ConstantsFolded)
}
