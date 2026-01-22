# MCP Server Demo

这是一个简单的 MCP (Model Context Protocol) 服务器示例，演示如何：
- 提供文件系统资源
- 暴露计算器工具
- 使用 Stdio 传输（与 Claude Desktop 兼容）

## 功能

### 资源
- **Documents**: 访问用户文档目录的文件

### 工具
- **calculator**: 执行简单的数学计算

### Prompts
- **greet**: 生成问候消息

## 运行示例

### 1. 构建

```bash
cd examples/mcp_server_demo
go build -o mcp_server_demo
```

### 2. 测试运行

```bash
./mcp_server_demo
```

服务器将监听 stdin/stdout，等待 JSON-RPC 消息。

### 3. 与 Claude Desktop 集成

#### macOS/Linux

编辑 `~/.config/claude/claude_desktop_config.json`:

```json
{
  "mcpServers": {
    "demo": {
      "command": "/absolute/path/to/mcp_server_demo"
    }
  }
}
```

#### Windows

编辑 `%APPDATA%\Claude\claude_desktop_config.json`:

```json
{
  "mcpServers": {
    "demo": {
      "command": "C:\\absolute\\path\\to\\mcp_server_demo.exe"
    }
  }
}
```

#### 重启 Claude Desktop

重启 Claude Desktop 后，它会自动连接到你的 MCP Server。

## 在 Claude 中使用

### 查询文档

```
你：列出可用的资源

Claude：我现在可以访问以下资源：
- Documents: 用户文档目录

你：读取 file:///documents/notes.txt

Claude：（读取并显示文件内容）
```

### 使用计算器

```
你：使用 calculator 工具计算 123

Claude：（调用工具）
Result: 123 = 123.00
```

### 使用 Prompt

```
你：使用 greet prompt，name 参数为 "Alice"

Claude：（使用 prompt 生成问候）
Hello Alice! Welcome! 🎉
```

## 工作原理

### 消息流程

```
Claude Desktop
      │
      │ Stdio (JSON-RPC 2.0)
      ▼
 MCP Server
      │
      ├─► Resources (FileSystemProvider)
      ├─► Tools (calculator)
      └─► Prompts (greet)
```

### JSON-RPC 示例

#### 初始化

**请求**:
```json
{
  "jsonrpc": "2.0",
  "id": "1",
  "method": "initialize",
  "params": {
    "protocolVersion": "2024-11-05",
    "clientInfo": {
      "name": "claude-desktop",
      "version": "1.0.0"
    }
  }
}
```

**响应**:
```json
{
  "jsonrpc": "2.0",
  "id": "1",
  "result": {
    "protocolVersion": "2024-11-05",
    "serverInfo": {
      "name": "demo-server",
      "version": "0.1.0"
    },
    "capabilities": {
      "resources": {"subscribe": true},
      "tools": {},
      "prompts": {}
    }
  }
}
```

#### 列出工具

**请求**:
```json
{
  "jsonrpc": "2.0",
  "id": "2",
  "method": "tools/list"
}
```

**响应**:
```json
{
  "jsonrpc": "2.0",
  "id": "2",
  "result": {
    "tools": [
      {
        "name": "calculator",
        "description": "Perform basic mathematical calculations",
        "inputSchema": {
          "type": "object",
          "properties": {
            "expression": {
              "type": "string",
              "description": "Mathematical expression"
            }
          },
          "required": ["expression"]
        }
      }
    ]
  }
}
```

## 扩展示例

### 添加自定义资源

```go
// 注册数据库资源
dbProvider := providers.NewDatabaseProvider(db)
server.RegisterResource(&mcp.Resource{
    URI:  "db://mydb",
    Name: "My Database",
}, dbProvider)
```

### 添加自定义工具

```go
searchTool := &mcp.Tool{
    Name:        "search",
    Description: "Search the web",
    InputSchema: map[string]any{
        "type": "object",
        "properties": map[string]any{
            "query": map[string]any{
                "type": "string",
            },
        },
    },
}

server.RegisterTool(searchTool, func(ctx context.Context, args map[string]any) (*mcp.ToolResult, error) {
    query := args["query"].(string)
    // 执行搜索...
    return &mcp.ToolResult{
        Content: []mcp.ContentBlock{
            {Type: "text", Text: "Search results..."},
        },
    }, nil
})
```

## 故障排查

### 问题：Claude Desktop 未显示 MCP Server

**检查清单**:
1. ✅ 配置文件路径正确
2. ✅ 可执行文件路径是绝对路径
3. ✅ 可执行文件有执行权限 (`chmod +x`)
4. ✅ 已重启 Claude Desktop

**查看日志**:

```bash
# macOS
tail -f ~/Library/Logs/Claude/mcp.log

# Linux
tail -f ~/.config/Claude/logs/mcp.log
```

### 问题：工具调用失败

**检查**:
- 参数类型是否正确
- 是否提供了所有必需参数
- 工具 Handler 是否正确处理错误

## 更多资源

- [MCP 规范](../../docs/V0.6.1_MCP_SPEC.md)
- [用户指南](../../docs/V0.6.1_USER_GUIDE.md)
- [MCP 官方网站](https://modelcontextprotocol.io/)

---

**创建日期**: 2026-01-22  
**版本**: v0.6.1
