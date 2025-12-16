package semantic

import (
	"fmt"

	"github.com/hassan/compiler/internal/lexer"
	"github.com/hassan/compiler/internal/parser/ast"
	"github.com/hassan/compiler/internal/semantic/types"
	"github.com/hassan/compiler/internal/symtab"
)

// Expression visitor methods for semantic analysis

func (a *Analyzer) VisitBinaryExpr(expr *ast.BinaryExpr) (interface{}, error) {
	// Check operands
	leftType, _ := expr.Left.Accept(a)
	rightType, _ := expr.Right.Accept(a)

	left := leftType.(types.Type)
	right := rightType.(types.Type)

	var resultType types.Type

	switch expr.Operator.Type {
	// Arithmetic operators: +, -, *, /, %
	case lexer.TokenPlus, lexer.TokenMinus, lexer.TokenStar,
		lexer.TokenSlash, lexer.TokenPercent:
		if !types.IsNumeric(left) || !types.IsNumeric(right) {
			a.error(expr.Operator.Position,
				fmt.Sprintf("operator %s requires numeric operands", expr.Operator.Lexeme))
			resultType = types.Invalid
		} else if !left.Equals(right) {
			a.error(expr.Operator.Position,
				fmt.Sprintf("mismatched types: %s and %s", left, right))
			resultType = types.Invalid
		} else {
			resultType = left
		}

	// Comparison operators: ==, !=
	case lexer.TokenEqual, lexer.TokenNotEqual:
		if !types.IsComparable(left) || !types.IsComparable(right) {
			a.error(expr.Operator.Position, "operands must be comparable")
			resultType = types.Invalid
		} else if !left.Equals(right) {
			a.error(expr.Operator.Position,
				fmt.Sprintf("cannot compare %s and %s", left, right))
			resultType = types.Invalid
		} else {
			resultType = types.Bool
		}

	// Relational operators: <, <=, >, >=
	case lexer.TokenLess, lexer.TokenLessEqual,
		lexer.TokenGreater, lexer.TokenGreaterEqual:
		if !types.IsOrdered(left) || !types.IsOrdered(right) {
			a.error(expr.Operator.Position, "operands must be ordered")
			resultType = types.Invalid
		} else if !left.Equals(right) {
			a.error(expr.Operator.Position,
				fmt.Sprintf("cannot compare %s and %s", left, right))
			resultType = types.Invalid
		} else {
			resultType = types.Bool
		}

	// Bitwise operators: &, |, ^, <<, >>
	case lexer.TokenBitAnd, lexer.TokenBitOr, lexer.TokenBitXor,
		lexer.TokenShl, lexer.TokenShr:
		if !types.IsIntegerType(left) || !types.IsIntegerType(right) {
			a.error(expr.Operator.Position, "bitwise operators require integer operands")
			resultType = types.Invalid
		} else {
			resultType = types.Int
		}

	default:
		a.error(expr.Operator.Position,
			fmt.Sprintf("unknown binary operator: %s", expr.Operator.Lexeme))
		resultType = types.Invalid
	}

	a.exprTypes[expr] = resultType
	return resultType, nil
}

func (a *Analyzer) VisitUnaryExpr(expr *ast.UnaryExpr) (interface{}, error) {
	operandType, _ := expr.Operand.Accept(a)
	opType := operandType.(types.Type)

	var resultType types.Type

	switch expr.Operator.Type {
	// Arithmetic negation: -
	case lexer.TokenMinus:
		if !types.IsNumeric(opType) {
			a.error(expr.Operator.Position, "unary - requires numeric operand")
			resultType = types.Invalid
		} else {
			resultType = opType
		}

	// Logical NOT: !
	case lexer.TokenNot:
		if !types.IsBooleanType(opType) {
			a.error(expr.Operator.Position, "unary ! requires boolean operand")
			resultType = types.Invalid
		} else {
			resultType = types.Bool
		}

	// Bitwise NOT: ~
	case lexer.TokenBitNot:
		if !types.IsIntegerType(opType) {
			a.error(expr.Operator.Position, "unary ~ requires integer operand")
			resultType = types.Invalid
		} else {
			resultType = types.Int
		}

	// Increment/Decrement: ++, --
	case lexer.TokenPlusPlus, lexer.TokenMinusMinus:
		if !types.IsNumeric(opType) {
			a.error(expr.Operator.Position,
				fmt.Sprintf("%s requires numeric operand", expr.Operator.Lexeme))
			resultType = types.Invalid
		} else {
			// Check that operand is assignable
			if ident, ok := expr.Operand.(*ast.IdentifierExpr); ok {
				symbol := a.currentScope.Lookup(ident.Name)
				if symbol != nil && !symbol.CanAssign() {
					a.error(expr.Operator.Position,
						fmt.Sprintf("cannot modify %s", ident.Name))
				}
			}
			resultType = opType
		}

	default:
		a.error(expr.Operator.Position,
			fmt.Sprintf("unknown unary operator: %s", expr.Operator.Lexeme))
		resultType = types.Invalid
	}

	a.exprTypes[expr] = resultType
	return resultType, nil
}

func (a *Analyzer) VisitLogicalExpr(expr *ast.LogicalExpr) (interface{}, error) {
	// Both operands must be boolean
	leftType, _ := expr.Left.Accept(a)
	rightType, _ := expr.Right.Accept(a)

	left := leftType.(types.Type)
	right := rightType.(types.Type)

	if !types.IsBooleanType(left) {
		a.error(expr.Left.Pos(), "left operand must be boolean")
	}
	if !types.IsBooleanType(right) {
		a.error(expr.Right.Pos(), "right operand must be boolean")
	}

	a.exprTypes[expr] = types.Bool
	return types.Bool, nil
}

func (a *Analyzer) VisitLiteralExpr(expr *ast.LiteralExpr) (interface{}, error) {
	var resultType types.Type

	switch expr.Token.Type {
	case lexer.TokenNumber:
		// Determine if int or float based on the value
		switch expr.Value.(type) {
		case int64:
			resultType = types.Int
		case float64:
			resultType = types.Float
		default:
			resultType = types.Invalid
		}

	case lexer.TokenString:
		resultType = types.String

	case lexer.TokenChar:
		resultType = types.Char

	case lexer.TokenTrue, lexer.TokenFalse:
		resultType = types.Bool

	case lexer.TokenNil:
		resultType = types.Nil

	default:
		a.error(expr.Token.Position, "unknown literal type")
		resultType = types.Invalid
	}

	a.exprTypes[expr] = resultType
	return resultType, nil
}

func (a *Analyzer) VisitIdentifierExpr(expr *ast.IdentifierExpr) (interface{}, error) {
	// Look up the symbol
	symbol := a.currentScope.Lookup(expr.Name)
	if symbol == nil {
		a.error(expr.Pos(), fmt.Sprintf("undefined: %s", expr.Name))
		a.exprTypes[expr] = types.Invalid
		return types.Invalid, nil
	}

	// Check it's not a type being used as a value
	if symbol.Kind == symtab.SymbolType {
		a.error(expr.Pos(), fmt.Sprintf("%s is a type, not a value", expr.Name))
		a.exprTypes[expr] = types.Invalid
		return types.Invalid, nil
	}

	a.exprTypes[expr] = symbol.Type
	return symbol.Type, nil
}

func (a *Analyzer) VisitCallExpr(expr *ast.CallExpr) (interface{}, error) {
	// Check callee
	calleeType, _ := expr.Callee.Accept(a)

	funcType, ok := calleeType.(*types.FunctionType)
	if !ok {
		a.error(expr.Callee.Pos(), "expression is not a function")
		a.exprTypes[expr] = types.Invalid
		return types.Invalid, nil
	}

	// Check argument count
	if len(expr.Args) != len(funcType.Parameters) {
		a.error(expr.LeftParen.Position,
			fmt.Sprintf("expected %d arguments, got %d",
				len(funcType.Parameters), len(expr.Args)))
		a.exprTypes[expr] = funcType.ReturnType
		return funcType.ReturnType, nil
	}

	// Check argument types
	for i, arg := range expr.Args {
		argType, _ := arg.Accept(a)
		expectedType := funcType.Parameters[i]
		if !a.assignable(argType.(types.Type), expectedType, arg.Pos()) {
			// Error already reported
		}
	}

	a.exprTypes[expr] = funcType.ReturnType
	return funcType.ReturnType, nil
}

func (a *Analyzer) VisitIndexExpr(expr *ast.IndexExpr) (interface{}, error) {
	// Check object
	objectType, _ := expr.Object.Accept(a)

	arrayType, ok := objectType.(*types.ArrayType)
	if !ok {
		a.error(expr.Object.Pos(), "expression is not an array")
		a.exprTypes[expr] = types.Invalid
		return types.Invalid, nil
	}

	// Check index type (must be int)
	indexType, _ := expr.Index.Accept(a)
	if !types.IsIntegerType(indexType.(types.Type)) {
		a.error(expr.Index.Pos(), "array index must be integer")
	}

	a.exprTypes[expr] = arrayType.ElementType
	return arrayType.ElementType, nil
}

func (a *Analyzer) VisitMemberExpr(expr *ast.MemberExpr) (interface{}, error) {
	// Check object
	objectType, _ := expr.Object.Accept(a)

	structType, ok := objectType.(*types.StructType)
	if !ok {
		a.error(expr.Object.Pos(), "expression is not a struct")
		a.exprTypes[expr] = types.Invalid
		return types.Invalid, nil
	}

	// Look up field
	field := structType.LookupField(expr.Member.Name)
	if field == nil {
		a.error(expr.Member.Pos(),
			fmt.Sprintf("struct %s has no field %s", structType.Name, expr.Member.Name))
		a.exprTypes[expr] = types.Invalid
		return types.Invalid, nil
	}

	a.exprTypes[expr] = field.Type
	return field.Type, nil
}

func (a *Analyzer) VisitAssignmentExpr(expr *ast.AssignmentExpr) (interface{}, error) {
	// Check target is assignable
	targetType, _ := expr.Target.Accept(a)
	valueType, _ := expr.Value.Accept(a)

	// Check target is a valid lvalue
	switch target := expr.Target.(type) {
	case *ast.IdentifierExpr:
		symbol := a.currentScope.Lookup(target.Name)
		if symbol != nil && !symbol.CanAssign() {
			a.error(expr.Target.Pos(),
				fmt.Sprintf("cannot assign to %s", target.Name))
		}

	case *ast.IndexExpr, *ast.MemberExpr:
		// These are valid lvalues

	default:
		a.error(expr.Target.Pos(), "invalid assignment target")
	}

	// Check types match
	if !a.assignable(valueType.(types.Type), targetType.(types.Type), expr.Value.Pos()) {
		// Error already reported
	}

	a.exprTypes[expr] = targetType.(types.Type)
	return targetType, nil
}

func (a *Analyzer) VisitGroupingExpr(expr *ast.GroupingExpr) (interface{}, error) {
	// Just pass through the inner expression's type
	innerType, err := expr.Expression.Accept(a)
	a.exprTypes[expr] = innerType.(types.Type)
	return innerType, err
}

func (a *Analyzer) VisitArrayLiteralExpr(expr *ast.ArrayLiteralExpr) (interface{}, error) {
	var elementType types.Type

	if expr.ElementType != nil {
		// Explicit element type
		elementType = a.resolveType(expr.ElementType)
	} else if len(expr.Elements) > 0 {
		// Infer from first element
		firstType, _ := expr.Elements[0].Accept(a)
		elementType = firstType.(types.Type)
	} else {
		a.error(expr.Pos(), "cannot infer array type from empty literal")
		elementType = types.Invalid
	}

	// Check all elements match
	for _, elem := range expr.Elements {
		elemType, _ := elem.Accept(a)
		if !a.assignable(elemType.(types.Type), elementType, elem.Pos()) {
			// Error already reported
		}
	}

	arrayType := types.NewArray(elementType, len(expr.Elements))
	a.exprTypes[expr] = arrayType
	return arrayType, nil
}

func (a *Analyzer) VisitStructLiteralExpr(expr *ast.StructLiteralExpr) (interface{}, error) {
	// Look up struct type
	symbol := a.currentScope.Lookup(expr.TypeName.Name)
	if symbol == nil {
		a.error(expr.TypeName.Pos(),
			fmt.Sprintf("undefined struct: %s", expr.TypeName.Name))
		a.exprTypes[expr] = types.Invalid
		return types.Invalid, nil
	}

	if symbol.Kind != symtab.SymbolStruct {
		a.error(expr.TypeName.Pos(),
			fmt.Sprintf("%s is not a struct", expr.TypeName.Name))
		a.exprTypes[expr] = types.Invalid
		return types.Invalid, nil
	}

	structType := symbol.Type.(*types.StructType)

	// Check fields
	providedFields := make(map[string]bool)
	for _, field := range expr.Fields {
		// Check field exists
		structField := structType.LookupField(field.Name.Name)
		if structField == nil {
			a.error(field.Name.Pos(),
				fmt.Sprintf("struct %s has no field %s",
					structType.Name, field.Name.Name))
			continue
		}

		// Check for duplicate fields
		if providedFields[field.Name.Name] {
			a.error(field.Name.Pos(),
				fmt.Sprintf("duplicate field: %s", field.Name.Name))
			continue
		}
		providedFields[field.Name.Name] = true

		// Check field value type
		valueType, _ := field.Value.Accept(a)
		if !a.assignable(valueType.(types.Type), structField.Type, field.Value.Pos()) {
			// Error already reported
		}
	}

	// Check all fields are provided
	for _, structField := range structType.Fields {
		if !providedFields[structField.Name] {
			a.error(expr.Pos(),
				fmt.Sprintf("missing field: %s", structField.Name))
		}
	}

	a.exprTypes[expr] = structType
	return structType, nil
}
