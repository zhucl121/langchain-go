package tools

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"strings"
	"unicode"

	"github.com/zhucl121/langchain-go/pkg/types"
)

// CalculatorTool 是计算器工具。
//
// 支持基本的算术运算：加、减、乘、除、取模、幂运算。
type CalculatorTool struct {
	name        string
	description string
}

// NewCalculatorTool 创建计算器工具。
//
// 返回：
//   - *CalculatorTool: 计算器工具实例
//
func NewCalculatorTool() *CalculatorTool {
	return &CalculatorTool{
		name:        "calculator",
		description: "Perform arithmetic calculations. Supports +, -, *, /, %, ^ (power).",
	}
}

// GetName 实现 Tool 接口。
func (t *CalculatorTool) GetName() string {
	return t.name
}

// GetDescription 实现 Tool 接口。
func (t *CalculatorTool) GetDescription() string {
	return t.description
}

// GetParameters 实现 Tool 接口。
func (t *CalculatorTool) GetParameters() types.Schema {
	return types.Schema{
		Type: "object",
		Properties: map[string]types.Schema{
			"expression": {
				Type:        "string",
				Description: "The mathematical expression to calculate (e.g., '2 + 2', '10 * 5', '2^3')",
			},
		},
		Required: []string{"expression"},
	}
}

// Execute 实现 Tool 接口。
func (t *CalculatorTool) Execute(ctx context.Context, args map[string]any) (any, error) {
	expr, ok := args["expression"]
	if !ok {
		return nil, fmt.Errorf("%w: missing 'expression' argument", ErrInvalidArguments)
	}

	exprStr, ok := expr.(string)
	if !ok {
		return nil, fmt.Errorf("%w: 'expression' must be a string", ErrInvalidArguments)
	}

	result, err := evaluateExpression(exprStr)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrExecutionFailed, err)
	}

	return result, nil
}

// ToTypesTool 实现 Tool 接口。
func (t *CalculatorTool) ToTypesTool() types.Tool {
	return types.Tool{
		Name:        t.name,
		Description: t.description,
		Parameters:  t.GetParameters(),
	}
}

// evaluateExpression 计算数学表达式（增强版本）
//
// 支持特性：
//   - 基础运算：+, -, *, /, %, ^
//   - 括号优先级
//   - 数学函数：sqrt, sin, cos, tan, abs, log, ln, exp
//   - 常量：pi, e
//
func evaluateExpression(expr string) (float64, error) {
	// 去除空格
	expr = strings.ReplaceAll(expr, " ", "")

	if expr == "" {
		return 0, fmt.Errorf("empty expression")
	}

	// 替换常量
	expr = strings.ReplaceAll(expr, "pi", "3.14159265358979323846")
	expr = strings.ReplaceAll(expr, "e", "2.71828182845904523536")

	// 处理函数调用
	expr, err := processFunctions(expr)
	if err != nil {
		return 0, err
	}

	// 解析并计算表达式
	return parseExpression(expr)
}

// processFunctions 处理数学函数
func processFunctions(expr string) (string, error) {
	functions := map[string]func(float64) float64{
		"sqrt": math.Sqrt,
		"abs":  math.Abs,
		"sin":  math.Sin,
		"cos":  math.Cos,
		"tan":  math.Tan,
		"log":  math.Log10,
		"ln":   math.Log,
		"exp":  math.Exp,
	}

	result := expr
	for funcName, funcImpl := range functions {
		for {
			idx := strings.Index(result, funcName+"(")
			if idx == -1 {
				break
			}

			// 找到对应的右括号
			parenCount := 1
			start := idx + len(funcName) + 1
			end := start

			for end < len(result) && parenCount > 0 {
				if result[end] == '(' {
					parenCount++
				} else if result[end] == ')' {
					parenCount--
				}
				end++
			}

			if parenCount != 0 {
				return "", fmt.Errorf("unmatched parentheses in %s", funcName)
			}

			// 提取参数并计算
			arg := result[start : end-1]
			argValue, err := parseExpression(arg)
			if err != nil {
				return "", fmt.Errorf("error in %s argument: %v", funcName, err)
			}

			// 应用函数
			resultValue := funcImpl(argValue)

			// 替换函数调用为结果
			result = result[:idx] + fmt.Sprintf("%f", resultValue) + result[end:]
		}
	}

	return result, nil
}

// parseExpression 解析表达式（支持 +, -, *, /, %, ^）
func parseExpression(expr string) (float64, error) {
	// 处理加法和减法（最低优先级）
	return parseAddSub(expr)
}

// parseAddSub 解析加减法
func parseAddSub(expr string) (float64, error) {
	parts, ops := splitByOperators(expr, []rune{'+', '-'}, false)

	if len(parts) == 0 {
		return 0, fmt.Errorf("invalid expression")
	}

	result, err := parseMulDiv(parts[0])
	if err != nil {
		return 0, err
	}

	for i, op := range ops {
		right, err := parseMulDiv(parts[i+1])
		if err != nil {
			return 0, err
		}

		switch op {
		case '+':
			result += right
		case '-':
			result -= right
		}
	}

	return result, nil
}

// parseMulDiv 解析乘除法
func parseMulDiv(expr string) (float64, error) {
	parts, ops := splitByOperators(expr, []rune{'*', '/', '%'}, false)

	if len(parts) == 0 {
		return 0, fmt.Errorf("invalid expression")
	}

	result, err := parsePower(parts[0])
	if err != nil {
		return 0, err
	}

	for i, op := range ops {
		right, err := parsePower(parts[i+1])
		if err != nil {
			return 0, err
		}

		switch op {
		case '*':
			result *= right
		case '/':
			if right == 0 {
				return 0, fmt.Errorf("division by zero")
			}
			result /= right
		case '%':
			if right == 0 {
				return 0, fmt.Errorf("modulo by zero")
			}
			result = float64(int(result) % int(right))
		}
	}

	return result, nil
}

// parsePower 解析幂运算
func parsePower(expr string) (float64, error) {
	parts, _ := splitByOperators(expr, []rune{'^'}, false)

	if len(parts) == 1 {
		return parsePrimary(parts[0])
	}

	// 幂运算是右结合的
	base, err := parsePrimary(parts[0])
	if err != nil {
		return 0, err
	}

	exponent, err := parsePower(strings.Join(parts[1:], "^"))
	if err != nil {
		return 0, err
	}

	return math.Pow(base, exponent), nil
}

// parsePrimary 解析基本值（数字或括号表达式）
func parsePrimary(expr string) (float64, error) {
	expr = strings.TrimSpace(expr)

	if expr == "" {
		return 0, fmt.Errorf("empty expression")
	}

	// 处理括号
	if expr[0] == '(' && expr[len(expr)-1] == ')' {
		return parseExpression(expr[1 : len(expr)-1])
	}

	// 处理负号
	if expr[0] == '-' {
		val, err := parsePrimary(expr[1:])
		if err != nil {
			return 0, err
		}
		return -val, nil
	}

	// 解析数字
	val, err := strconv.ParseFloat(expr, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid number: %s", expr)
	}

	return val, nil
}

// splitByOperators 按操作符分割表达式（考虑括号）
func splitByOperators(expr string, operators []rune, skipParentheses bool) ([]string, []rune) {
	var parts []string
	var ops []rune
	var currentPart strings.Builder
	parenDepth := 0

	for i, ch := range expr {
		if ch == '(' {
			parenDepth++
			currentPart.WriteRune(ch)
			continue
		}

		if ch == ')' {
			parenDepth--
			currentPart.WriteRune(ch)
			continue
		}

		if parenDepth == 0 && contains(operators, ch) {
			// 检查是否是负号（而不是减号）
			if ch == '-' && i == 0 {
				currentPart.WriteRune(ch)
				continue
			}
			if ch == '-' && i > 0 && isOperator(rune(expr[i-1])) {
				currentPart.WriteRune(ch)
				continue
			}

			parts = append(parts, currentPart.String())
			ops = append(ops, ch)
			currentPart.Reset()
		} else {
			currentPart.WriteRune(ch)
		}
	}

	parts = append(parts, currentPart.String())
	return parts, ops
}

// contains 检查切片是否包含元素
func contains(slice []rune, item rune) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// isOperator 检查字符是否是操作符
func isOperator(ch rune) bool {
	return ch == '+' || ch == '-' || ch == '*' || ch == '/' || ch == '%' || ch == '^' || ch == '('
}

// isDigit 检查字符是否是数字
func isDigit(ch rune) bool {
	return unicode.IsDigit(ch) || ch == '.'
}
