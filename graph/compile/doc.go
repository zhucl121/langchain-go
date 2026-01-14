// Package compile 提供 LangGraph 图编译和验证功能。
//
// compile 包负责将声明式的 StateGraph 编译为可执行的图结构。
// 编译过程包括验证、优化和准备执行计划。
//
// # 编译流程
//
// 1. **验证** - 检查图的完整性和合法性
//   - 入口点存在
//   - 节点和边的有效性
//   - 无孤立节点
//   - 无无效边
//
// 2. **拓扑分析** - 分析图的结构
//   - 检测循环
//   - 检查可达性
//   - 构建执行顺序
//
// 3. **优化** - 优化执行计划（可选）
//   - 并行节点识别
//   - 边合并
//   - 死代码消除
//
// # 基本使用
//
// 编译状态图：
//
//	compiler := compile.NewCompiler[MyState]()
//	compiled, err := compiler.Compile(stateGraph)
//	if err != nil {
//	    // 处理编译错误
//	}
//
// 验证图：
//
//	validator := compile.NewValidator[MyState]()
//	if err := validator.Validate(stateGraph); err != nil {
//	    // 处理验证错误
//	}
//
// # 错误处理
//
// 编译器会返回详细的错误信息：
//   - ValidationError: 验证错误
//   - CyclicGraphError: 循环检测错误
//   - UnreachableNodeError: 不可达节点错误
//
package compile
