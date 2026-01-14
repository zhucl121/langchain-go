// Package tools 提供 LangChain 工具（Tool）系统的实现。
//
// tools 包实现了可执行的工具接口和工具执行器，允许 LLM 调用外部函数。
// 工具是 Agent 系统的核心组件，让 AI 能够与外部世界交互。
//
// 核心特性：
//   - Tool 接口定义
//   - 工具执行器（ToolExecutor）
//   - 内置工具集合（计算器、HTTP 请求等）
//   - 参数验证
//   - 错误处理
//   - 超时控制
//
// 基本用法：
//
//	import (
//	    "context"
//	    "langchain-go/core/tools"
//	    "langchain-go/pkg/types"
//	)
//
//	// 创建自定义工具
//	calculator := tools.NewFunctionTool(tools.FunctionToolConfig{
//	    Name:        "calculator",
//	    Description: "Perform arithmetic calculations",
//	    Parameters: types.Schema{
//	        Type: "object",
//	        Properties: map[string]types.Schema{
//	            "expression": {Type: "string", Description: "Math expression"},
//	        },
//	        Required: []string{"expression"},
//	    },
//	    Fn: func(ctx context.Context, args map[string]any) (any, error) {
//	        expr := args["expression"].(string)
//	        // 计算表达式
//	        return result, nil
//	    },
//	})
//
//	// 执行工具
//	result, err := calculator.Execute(ctx, map[string]any{
//	    "expression": "2 + 2",
//	})
//
// 使用工具执行器：
//
//	// 创建执行器
//	executor := tools.NewToolExecutor(tools.ToolExecutorConfig{
//	    Tools:   []tools.Tool{calculator, search, database},
//	    Timeout: 30 * time.Second,
//	})
//
//	// 执行工具调用
//	result, err := executor.Execute(ctx, "calculator", map[string]any{
//	    "expression": "10 * 5",
//	})
//
// 与 ChatModel 集成：
//
//	// 将工具绑定到模型
//	modelWithTools := model.BindTools([]types.Tool{
//	    calculator.ToTypesTool(),
//	    search.ToTypesTool(),
//	})
//
//	// 调用模型（可能返回工具调用）
//	response, _ := modelWithTools.Invoke(ctx, messages)
//
//	// 如果有工具调用，执行工具
//	if len(response.ToolCalls) > 0 {
//	    for _, call := range response.ToolCalls {
//	        result, _ := executor.Execute(ctx, call.Function.Name, call.Function.Arguments)
//	        // 处理结果
//	    }
//	}
//
// 内置工具：
//
//	// 计算器工具
//	calc := tools.NewCalculatorTool()
//
//	// HTTP 请求工具
//	http := tools.NewHTTPRequestTool(tools.HTTPRequestConfig{
//	    AllowedMethods: []string{"GET", "POST"},
//	    Timeout:        10 * time.Second,
//	})
//
//	// Shell 命令工具（谨慎使用）
//	shell := tools.NewShellTool(tools.ShellToolConfig{
//	    AllowedCommands: []string{"ls", "pwd"},
//	})
//
// 错误处理：
//
//	result, err := tool.Execute(ctx, args)
//	if err != nil {
//	    if errors.Is(err, tools.ErrToolNotFound) {
//	        // 工具不存在
//	    } else if errors.Is(err, tools.ErrInvalidArguments) {
//	        // 参数无效
//	    } else if errors.Is(err, tools.ErrTimeout) {
//	        // 执行超时
//	    }
//	}
//
package tools
