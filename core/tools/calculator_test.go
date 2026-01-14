package tools

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCalculatorTool(t *testing.T) {
	calc := NewCalculatorTool()
	assert.NotNil(t, calc)
	assert.Equal(t, "calculator", calc.GetName())
	assert.NotEmpty(t, calc.GetDescription())
}

func TestCalculatorTool_Execute(t *testing.T) {
	calc := NewCalculatorTool()

	tests := []struct {
		name       string
		expression string
		expected   float64
	}{
		// 基本运算
		{name: "addition", expression: "2 + 3", expected: 5},
		{name: "subtraction", expression: "10 - 4", expected: 6},
		{name: "multiplication", expression: "3 * 4", expected: 12},
		{name: "division", expression: "20 / 5", expected: 4},
		{name: "modulo", expression: "10 % 3", expected: 1},

		// 幂运算
		{name: "power", expression: "2 ^ 3", expected: 8},
		{name: "power2", expression: "3 ^ 2", expected: 9},

		// 复合运算
		{name: "order of operations", expression: "2 + 3 * 4", expected: 14},
		{name: "parentheses", expression: "(2 + 3) * 4", expected: 20},
		{name: "nested parentheses", expression: "((2 + 3) * 4) - 5", expected: 15},

		// 负数
		{name: "negative number", expression: "-5 + 3", expected: -2},
		{name: "subtraction negative", expression: "10 - (-5)", expected: 15},

		// 小数
		{name: "decimal addition", expression: "2.5 + 3.5", expected: 6},
		{name: "decimal division", expression: "7.5 / 2.5", expected: 3},

		// 复杂表达式
		{name: "complex", expression: "2 * (3 + 4) - 5 / 5", expected: 13},
		{name: "complex2", expression: "(10 + 5) * 2 / 3", expected: 10},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := calc.Execute(context.Background(), map[string]any{
				"expression": tt.expression,
			})

			require.NoError(t, err)
			assert.InDelta(t, tt.expected, result, 0.0001)
		})
	}
}

func TestCalculatorTool_ExecuteErrors(t *testing.T) {
	calc := NewCalculatorTool()

	tests := []struct {
		name       string
		args       map[string]any
		expectErr  bool
		errMessage string
	}{
		{
			name:      "missing expression",
			args:      map[string]any{},
			expectErr: true,
		},
		{
			name: "invalid type",
			args: map[string]any{
				"expression": 123,
			},
			expectErr: true,
		},
		{
			name: "division by zero",
			args: map[string]any{
				"expression": "10 / 0",
			},
			expectErr: true,
		},
		{
			name: "invalid expression",
			args: map[string]any{
				"expression": "2 +",
			},
			expectErr: true,
		},
		{
			name: "empty expression",
			args: map[string]any{
				"expression": "",
			},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := calc.Execute(context.Background(), tt.args)

			if tt.expectErr {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestCalculatorTool_ToTypesTool(t *testing.T) {
	calc := NewCalculatorTool()
	typesTool := calc.ToTypesTool()

	assert.Equal(t, "calculator", typesTool.Name)
	assert.NotEmpty(t, typesTool.Description)
	assert.Equal(t, "object", typesTool.Parameters.Type)
	assert.Contains(t, typesTool.Parameters.Required, "expression")
}

func TestEvaluateExpression(t *testing.T) {
	tests := []struct {
		name       string
		expression string
		expected   float64
		expectErr  bool
	}{
		{name: "simple", expression: "1+1", expected: 2, expectErr: false},
		{name: "with spaces", expression: " 1 + 1 ", expected: 2, expectErr: false},
		{name: "multiplication", expression: "2*3", expected: 6, expectErr: false},
		{name: "division", expression: "6/2", expected: 3, expectErr: false},
		{name: "power", expression: "2^3", expected: 8, expectErr: false},
		{name: "empty", expression: "", expected: 0, expectErr: true},
		{name: "invalid", expression: "abc", expected: 0, expectErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := evaluateExpression(tt.expression)

			if tt.expectErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.InDelta(t, tt.expected, result, 0.0001)
			}
		})
	}
}

func TestSplitByOperators(t *testing.T) {
	tests := []struct {
		name          string
		expr          string
		operators     []rune
		expectedParts []string
		expectedOps   []rune
	}{
		{
			name:          "simple addition",
			expr:          "1+2",
			operators:     []rune{'+'},
			expectedParts: []string{"1", "2"},
			expectedOps:   []rune{'+'},
		},
		{
			name:          "multiple operators",
			expr:          "1+2-3",
			operators:     []rune{'+', '-'},
			expectedParts: []string{"1", "2", "3"},
			expectedOps:   []rune{'+', '-'},
		},
		{
			name:          "with parentheses",
			expr:          "(1+2)*3",
			operators:     []rune{'*'},
			expectedParts: []string{"(1+2)", "3"},
			expectedOps:   []rune{'*'},
		},
		{
			name:          "negative number",
			expr:          "-5+3",
			operators:     []rune{'+'},
			expectedParts: []string{"-5", "3"},
			expectedOps:   []rune{'+'},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parts, ops := splitByOperators(tt.expr, tt.operators, false)

			assert.Equal(t, tt.expectedParts, parts)
			assert.Equal(t, tt.expectedOps, ops)
		})
	}
}
