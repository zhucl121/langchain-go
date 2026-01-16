// Package templates 提供预定义的 prompt 模板。
//
// 这些模板针对常见任务进行了优化，可以直接使用。
//
// 支持的模板：
//   - RAG 模板
//   - Agent 模板
//   - QA 模板
//
package templates

import "langchain-go/core/prompts"

// ==================== RAG Templates ====================

// DefaultRAGPrompt 默认 RAG prompt  
var DefaultRAGPrompt = mustNewPrompt(prompts.PromptTemplateConfig{
	Template: `基于以下上下文回答问题。如果上下文中没有相关信息，请明确说明无法回答。

上下文:
{{.context}}

问题: {{.question}}

回答:`,
	InputVariables: []string{"context", "question"},
})

// DetailedRAGPrompt 详细的 RAG prompt
var DetailedRAGPrompt = mustNewPrompt(prompts.PromptTemplateConfig{
	Template: `你是一个专业的助手，请基于给定的上下文回答用户的问题。

指导原则：
1. 仅使用上下文中的信息回答
2. 如果上下文中没有相关信息，请明确说明
3. 尽可能引用来源文档
4. 保持客观和准确

上下文:
{{.context}}

用户问题: {{.question}}

请提供详细的回答:`,
	InputVariables: []string{"context", "question"},
})

// ConversationalRAGPrompt 对话式 RAG prompt
var ConversationalRAGPrompt = mustNewPrompt(prompts.PromptTemplateConfig{
	Template: `基于以下上下文和对话历史回答问题。

对话历史:
{{.chat_history}}

上下文:
{{.context}}

问题: {{.question}}

回答:`,
	InputVariables: []string{"chat_history", "context", "question"},
})

// MultilingualRAGPrompt 多语言 RAG prompt
var MultilingualRAGPrompt = mustNewPrompt(prompts.PromptTemplateConfig{
	Template: `Based on the following context, answer the question in the same language as the question.

Context:
{{.context}}

Question: {{.question}}

Answer:`,
	InputVariables: []string{"context", "question"},
})

// StructuredRAGPrompt 结构化 RAG prompt
var StructuredRAGPrompt = mustNewPrompt(prompts.PromptTemplateConfig{
	Template: `基于以下上下文回答问题，并以 JSON 格式返回答案。

上下文:
{{.context}}

问题: {{.question}}

请返回以下格式的 JSON:
{
  "answer": "你的答案",
  "confidence": 0.95,
  "sources": ["来源1", "来源2"]
}

JSON 回答:`,
	InputVariables: []string{"context", "question"},
})

// ConciseRAGPrompt 简洁的 RAG prompt
var ConciseRAGPrompt = mustNewPrompt(prompts.PromptTemplateConfig{
	Template: `基于以下上下文，用一到两句话简洁回答问题。

上下文:
{{.context}}

问题: {{.question}}

简洁回答:`,
	InputVariables: []string{"context", "question"},
})

// ==================== QA Templates ====================

// SimpleQAPrompt 简单的 QA prompt
var SimpleQAPrompt = mustNewPrompt(prompts.PromptTemplateConfig{
	Template: `请回答以下问题：

{{.question}}

回答:`,
	InputVariables: []string{"question"},
})

// StepByStepQAPrompt 逐步推理的 QA prompt
var StepByStepQAPrompt = mustNewPrompt(prompts.PromptTemplateConfig{
	Template: `请逐步推理并回答以下问题：

问题: {{.question}}

思考过程:`,
	InputVariables: []string{"question"},
})

// ==================== Agent Templates ====================

// ReActPrompt ReAct Agent prompt
var ReActPrompt = mustNewPrompt(prompts.PromptTemplateConfig{
	Template: `Answer the following questions as best you can. You have access to the following tools:

{{.tools}}

Use the following format:

Question: the input question you must answer
Thought: you should always think about what to do
Action: the action to take, should be one of [{{.tool_names}}]
Action Input: the input to the action (in JSON format)
Observation: the result of the action
... (this Thought/Action/Action Input/Observation can repeat N times)
Thought: I now know the final answer
Final Answer: the final answer to the original input question

Begin!

Question: {{.input}}
{{.history}}
Thought:`,
	InputVariables: []string{"tools", "tool_names", "input", "history"},
})

// ChineseReActPrompt 中文 ReAct Agent prompt
var ChineseReActPrompt = mustNewPrompt(prompts.PromptTemplateConfig{
	Template: `请尽力回答以下问题。你可以使用以下工具：

{{.tools}}

请使用以下格式：

问题: 你需要回答的输入问题
思考: 你应该思考接下来要做什么
动作: 要采取的动作，应该是 [{{.tool_names}}] 之一
动作输入: 动作的输入（JSON 格式）
观察: 动作的结果
... (思考/动作/动作输入/观察 可以重复 N 次)
思考: 我现在知道最终答案了
最终答案: 原始问题的最终答案

开始！

问题: {{.input}}
{{.history}}
思考:`,
	InputVariables: []string{"tools", "tool_names", "input", "history"},
})

// PlanExecutePrompt Plan-Execute prompt
var PlanExecutePrompt = mustNewPrompt(prompts.PromptTemplateConfig{
	Template: `Let's first understand the problem and devise a plan to solve it.
Then, let's carry out the plan step by step.

Problem: {{.input}}

Plan:`,
	InputVariables: []string{"input"},
})

// ToolCallingPrompt Tool Calling prompt
var ToolCallingPrompt = mustNewPrompt(prompts.PromptTemplateConfig{
	Template: `You are a helpful assistant with access to the following tools:

{{.tools}}

To use a tool, please use the following format:
<tool>tool_name</tool>
<tool_input>{"param": "value"}</tool_input>

Question: {{.input}}

Response:`,
	InputVariables: []string{"tools", "input"},
})

// ==================== Summarization Templates ====================

// SummarizationPrompt 摘要 prompt
var SummarizationPrompt = mustNewPrompt(prompts.PromptTemplateConfig{
	Template: `请为以下文本写一个简洁的摘要：

文本:
{{.text}}

摘要:`,
	InputVariables: []string{"text"},
})

// RefinePrompt 精炼摘要 prompt
var RefinePrompt = mustNewPrompt(prompts.PromptTemplateConfig{
	Template: `你已经生成了一个摘要：
{{.existing_summary}}

现在有新的文本：
{{.text}}

请结合新文本精炼你的摘要：`,
	InputVariables: []string{"existing_summary", "text"},
})

// ==================== Translation Templates ====================

// TranslationPrompt 翻译 prompt
var TranslationPrompt = mustNewPrompt(prompts.PromptTemplateConfig{
	Template: `请将以下文本从 {{.source_lang}} 翻译成 {{.target_lang}}：

原文:
{{.text}}

译文:`,
	InputVariables: []string{"source_lang", "target_lang", "text"},
})

// ==================== Code Templates ====================

// CodeExplanationPrompt 代码解释 prompt
var CodeExplanationPrompt = mustNewPrompt(prompts.PromptTemplateConfig{
	Template: `请解释以下 {{.language}} 代码的功能：

代码:
{{.code}}

解释:`,
	InputVariables: []string{"language", "code"},
})

// CodeReviewPrompt 代码审查 prompt
var CodeReviewPrompt = mustNewPrompt(prompts.PromptTemplateConfig{
	Template: `请审查以下 {{.language}} 代码，指出潜在问题和改进建议：

代码:
{{.code}}

审查意见:`,
	InputVariables: []string{"language", "code"},
})

// ==================== Classification Templates ====================

// ClassificationPrompt 分类 prompt
var ClassificationPrompt = mustNewPrompt(prompts.PromptTemplateConfig{
	Template: `请将以下文本分类到以下类别之一：{{.categories}}

文本:
{{.text}}

类别:`,
	InputVariables: []string{"categories", "text"},
})

// SentimentAnalysisPrompt 情感分析 prompt
var SentimentAnalysisPrompt = mustNewPrompt(prompts.PromptTemplateConfig{
	Template: `请分析以下文本的情感倾向（正面/负面/中性）：

文本:
{{.text}}

情感:`,
	InputVariables: []string{"text"},
})

// ==================== Helper Functions ====================

// mustNewPrompt 创建 prompt，如果失败则 panic
func mustNewPrompt(config prompts.PromptTemplateConfig) *prompts.PromptTemplate {
	prompt, err := prompts.NewPromptTemplate(config)
	if err != nil {
		panic(err)
	}
	return prompt
}

// GetRAGTemplate 获取 RAG 模板
func GetRAGTemplate(name string) *prompts.PromptTemplate {
	templates := map[string]*prompts.PromptTemplate{
		"default":        DefaultRAGPrompt,
		"detailed":       DetailedRAGPrompt,
		"conversational": ConversationalRAGPrompt,
		"multilingual":   MultilingualRAGPrompt,
		"structured":     StructuredRAGPrompt,
		"concise":        ConciseRAGPrompt,
	}

	if template, ok := templates[name]; ok {
		return template
	}
	return DefaultRAGPrompt
}

// GetAgentTemplate 获取 Agent 模板
func GetAgentTemplate(name string) *prompts.PromptTemplate {
	templates := map[string]*prompts.PromptTemplate{
		"react":         ReActPrompt,
		"react_chinese": ChineseReActPrompt,
		"plan_execute":  PlanExecutePrompt,
		"tool_calling":  ToolCallingPrompt,
	}

	if template, ok := templates[name]; ok {
		return template
	}
	return ReActPrompt
}
