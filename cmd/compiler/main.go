// Package main provides the compiler entry point.
//
// This demonstrates the complete compiler pipeline:
// 1. Lexical Analysis (tokenization)
// 2. Syntax Analysis (parsing)
// 3. Semantic Analysis (type checking, name resolution)
// 4. IR Generation (intermediate representation)
// 5. Optimization (constant folding, dead code elimination)
//
// Future versions will add code generation for target architectures.
package main

import (
	"fmt"
	"os"

	"github.com/hassan/compiler/internal/ir"
	"github.com/hassan/compiler/internal/lexer"
	"github.com/hassan/compiler/internal/optimizer"
	"github.com/hassan/compiler/internal/parser"
	"github.com/hassan/compiler/internal/parser/ast"
	"github.com/hassan/compiler/internal/semantic"
)

func main() {
	// Check command line arguments
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: %s <source-file>\n", os.Args[0])
		os.Exit(1)
	}

	filename := os.Args[1]

	// Read the source file
	source, err := os.ReadFile(filename)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading file: %v\n", err)
		os.Exit(1)
	}

	// Create lexer
	lex := lexer.New(string(source), filename)

	// Create parser
	p := parser.New(lex)

	// Parse the file
	file, errors := p.ParseFile(filename)

	// Report parsing errors
	if len(errors) > 0 {
		fmt.Fprintf(os.Stderr, "Parsing errors:\n")
		for _, err := range errors {
			fmt.Fprintf(os.Stderr, "  %v\n", err)
		}
		os.Exit(1)
	}

	fmt.Printf("✓ Parsing successful\n")

	// Perform semantic analysis
	analyzer := semantic.New()
	semanticErrors := analyzer.Analyze(file)

	// Report semantic errors
	if len(semanticErrors) > 0 {
		fmt.Fprintf(os.Stderr, "\nSemantic errors:\n")
		for _, err := range semanticErrors {
			fmt.Fprintf(os.Stderr, "  %v\n", err)
		}
		os.Exit(1)
	}

	fmt.Printf("✓ Semantic analysis successful\n")

	// Generate IR
	builder := ir.NewBuilder(analyzer)
	module, irErrors := builder.Build(file)

	// Report IR generation errors
	if len(irErrors) > 0 {
		fmt.Fprintf(os.Stderr, "\nIR generation errors:\n")
		for _, err := range irErrors {
			fmt.Fprintf(os.Stderr, "  %v\n", err)
		}
		os.Exit(1)
	}

	fmt.Printf("✓ IR generation successful\n")

	// Verify IR before optimization
	verifyErrors := module.Verify()
	if len(verifyErrors) > 0 {
		fmt.Fprintf(os.Stderr, "\nIR verification errors:\n")
		for _, err := range verifyErrors {
			fmt.Fprintf(os.Stderr, "  %v\n", err)
		}
		os.Exit(1)
	}

	// Show unoptimized IR
	fmt.Printf("\n=== Unoptimized IR ===\n\n")
	fmt.Println(module.String())

	// Optimize the IR
	opt := optimizer.NewOptimizer()
	opt.SetVerbose(false) // Set to true to see optimization details

	if err := opt.Optimize(module); err != nil {
		fmt.Fprintf(os.Stderr, "\nOptimization error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✓ Optimization successful\n")

	// Verify IR after optimization
	verifyErrors = module.Verify()
	if len(verifyErrors) > 0 {
		fmt.Fprintf(os.Stderr, "\nIR verification errors after optimization:\n")
		for _, err := range verifyErrors {
			fmt.Fprintf(os.Stderr, "  %v\n", err)
		}
		os.Exit(1)
	}

	// Success!
	fmt.Printf("\n=== Compilation Summary ===\n")
	fmt.Printf("File: %s\n", filename)
	fmt.Printf("Package: %s\n", file.Package.Name.Name)
	fmt.Printf("Imports: %d\n", len(file.Imports))
	fmt.Printf("Declarations: %d\n", len(file.Decls))
	fmt.Printf("Comments: %d\n", len(file.Comments))
	fmt.Printf("\n=== Optimized IR ===\n\n")
	fmt.Println(module.String())

	// Print summary of declarations
	fmt.Println("\nDeclarations:")
	for i, decl := range file.Decls {
		switch d := decl.(type) {
		case *ast.FuncDecl:
			fmt.Printf("  %d. Function: %s\n", i+1, d.Name.Name)
		case *ast.VarDecl:
			names := make([]string, len(d.Names))
			for j, name := range d.Names {
				names[j] = name.Name
			}
			fmt.Printf("  %d. Variable(s): %v\n", i+1, names)
		case *ast.StructDecl:
			fmt.Printf("  %d. Struct: %s (%d fields)\n", i+1, d.Name.Name, len(d.Fields))
		case *ast.TypeDecl:
			fmt.Printf("  %d. Type alias: %s\n", i+1, d.Name.Name)
		}
	}
}
