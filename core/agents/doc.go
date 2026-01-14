// Package agents 提供 Agent 系统实现。
//
// Agent 是能够使用工具并自主决策的 LLM 应用。
//
// # 核心概念
//
// Agent 通过观察-思考-行动（ReAct）循环来完成任务：
//   1. 观察（Observe）- 接收输入和环境信息
//   2. 思考（Think）- 使用 LLM 分析和规划
//   3. 行动（Act）- 调用工具执行操作
//   4. 重复直到完成任务
//
// # Agent 类型
//
//   - ReAct Agent: 推理和行动结合
//   - ToolCalling Agent: 原生工具调用
//   - Conversational Agent: 对话式 Agent
//
// # 基本使用
//
// 创建 Agent：
//
//	agent, err := agents.CreateAgent(agents.AgentConfig{
//	    Type:      agents.ReActAgent,
//	    LLM:       chatModel,
//	    Tools:     toolList,
//	    MaxSteps:  10,
//	})
//
// 执行 Agent：
//
//	executor := agents.NewExecutor(agent)
//	result, err := executor.Execute(ctx, "帮我查询今天的天气")
//
package agents
