# TianNiu Backend System Design - Tool System

## 1. Overview

### 1.1 Purpose

The Tool System provides built-in utilities for interacting with the system and external services, enabling the agent to perform practical operations.

### 1.2 Design Goals

- **Simplicity**: Easy to use and extend
- **Security**: Safe execution environment
- **Consistency**: Uniform interface across tools
- **Extensibility**: Plugin-based architecture

### 1.3 Tool Interface

```go
type Tool interface {
    Name() string
    Description() string
    Parameters() []Parameter
    Execute(ctx context.Context, args map[string]interface{}) (interface{}, error)
}

type Parameter struct {
    Name        string
    Type        string
    Description string
    Required    bool
}
```

## 2. Tool Registry

### 2.1 Responsibilities

- Tool registration and discovery
- Tool metadata management
- Tool execution coordination

### 2.2 Structure

```go
type Registry struct {
    tools map[string]Tool
    sync.RWMutex
}
```

### 2.3 Key Methods

| Method | Description |
|--------|-------------|
| `NewRegistry()` | Create new tool registry |
| `Register(tool)` | Register a new tool |
| `Get(name)` | Retrieve tool by name |
| `List()` | List all registered tools |
| `Execute(name, args)` | Execute tool |

### 2.4 Registration Process

```go
func (r *Registry) Register(tool Tool) {
    r.Lock()
    defer r.Unlock()
    r.tools[tool.Name()] = tool
}
```

## 3. Built-in Tools

### 3.1 Bash Tool

**Responsibility**: Execute shell commands

```go
type BashTool struct {
    workingDir string
}

func (b *BashTool) Execute(ctx context.Context, args map[string]interface{}) (interface{}, error) {
    command, ok := args["command"].(string)
    if !ok {
        return nil, fmt.Errorf("command is required")
    }
    
    cmd := exec.CommandContext(ctx, "bash", "-c", command)
    cmd.Dir = b.workingDir
    
    output, err := cmd.CombinedOutput()
    if err != nil {
        return string(output), fmt.Errorf("command failed: %w", err)
    }
    
    return string(output), nil
}
```

**Configuration**:
```yaml
bash_tool:
  enabled: true
  working_dir: /workspace
  allowed_commands:
    - ls
    - cat
    - grep
    - git
```

### 3.2 File Tool

**Responsibility**: File operations (read, write, delete)

```go
type FileTool struct{}

func (f *FileTool) Execute(ctx context.Context, args map[string]interface{}) (interface{}, error) {
    action, ok := args["action"].(string)
    if !ok {
        return nil, fmt.Errorf("action is required")
    }
    
    switch action {
    case "read":
        return f.readFile(args)
    case "write":
        return f.writeFile(args)
    case "delete":
        return f.deleteFile(args)
    default:
        return nil, fmt.Errorf("unknown action: %s", action)
    }
}
```

**Supported Actions**:
| Action | Description |
|--------|-------------|
| `read` | Read file contents |
| `write` | Write content to file |
| `delete` | Delete file |
| `list` | List directory contents |

### 3.3 Web Search Tool

**Responsibility**: Search the web for information

```go
type WebSearchTool struct {
    apiKey string
}

func (w *WebSearchTool) Execute(ctx context.Context, args map[string]interface{}) (interface{}, error) {
    query, ok := args["query"].(string)
    if !ok {
        return nil, fmt.Errorf("query is required")
    }
    
    results, err := w.search(ctx, query)
    if err != nil {
        return nil, err
    }
    
    return results, nil
}
```

### 3.4 Calculator Tool

**Responsibility**: Perform mathematical calculations

```go
type CalculatorTool struct{}

func (c *CalculatorTool) Execute(ctx context.Context, args map[string]interface{}) (interface{}, error) {
    expression, ok := args["expression"].(string)
    if !ok {
        return nil, fmt.Errorf("expression is required")
    }
    
    result, err := evaluateExpression(expression)
    if err != nil {
        return nil, err
    }
    
    return result, nil
}
```

### 3.5 Python Runner Tool

**Responsibility**: Execute Python code

```go
type PythonRunnerTool struct{}

func (p *PythonRunnerTool) Execute(ctx context.Context, args map[string]interface{}) (interface{}, error) {
    code, ok := args["code"].(string)
    if !ok {
        return nil, fmt.Errorf("code is required")
    }
    
    result, err := p.executePython(ctx, code)
    if err != nil {
        return nil, err
    }
    
    return result, nil
}
```

## 4. Tool Execution Flow

```
LLM generates tool call
    │
    ▼
ToolRegistry.Get(toolName)
    │
    ▼
Tool.Validate(args)
    │
    ├─→ [Invalid] → Return error
    │
    └─→ [Valid] → Continue
    │
    ▼
Tool.Execute(ctx, args)
    │
    ├─→ [Success] → Return result
    │
    └─→ [Error] → Return error with details
    │
    ▼
Format result for LLM
    │
    ▼
Return to Agent
```

## 5. Security Considerations

### 5.1 Input Validation

- **Command sanitization**: Prevent command injection
- **Path validation**: Prevent path traversal
- **Resource limits**: Limit execution time and memory

### 5.2 Sandbox Environment

- **Isolated filesystem**: Restrict access to specific directories
- **Network restrictions**: Control external network access
- **Process isolation**: Run tools in isolated processes

### 5.3 Permission Model

| Role | Permissions |
|------|-------------|
| **Admin** | Full access to all tools |
| **User** | Limited access based on role |
| **Guest** | Read-only operations |

### 5.4 Allowed Commands

```yaml
security:
  allowed_commands:
    - ls
    - cat
    - grep
    - git
    - python
    - go
  allowed_directories:
    - /workspace
    - /tmp
  blocked_commands:
    - rm
    - sudo
    - chmod
```

## 6. Error Handling

### 6.1 Error Types

| Error | Description |
|-------|-------------|
| `ErrToolNotFound` | Tool not found |
| `ErrInvalidArgs` | Invalid arguments |
| `ErrExecutionFailed` | Execution failed |
| `ErrTimeout` | Execution timed out |
| `ErrPermissionDenied` | Permission denied |

### 6.2 Error Recovery

- **Timeout**: Kill process and return timeout error
- **Permission denied**: Return permission error
- **Unknown error**: Log details and return generic error

## 7. Configuration

### 7.1 Tool Configuration

```yaml
tools:
  bash:
    enabled: true
    working_dir: /workspace
  file:
    enabled: true
    allowed_directories:
      - /workspace
      - /tmp
  web_search:
    enabled: true
    api_key: ${SEARCH_API_KEY}
  calculator:
    enabled: true
  python_runner:
    enabled: true
```

### 7.2 Environment Variables

| Variable | Description |
|----------|-------------|
| `TOOL_BASH_WORKING_DIR` | Default working directory for bash |
| `SEARCH_API_KEY` | API key for web search |

## 8. Tool Integration with Agent

### 8.1 Tool Registration in Agent

```go
func NewAgent(modelConf shared.ModelConfig, systemPrompt string, 
              tools []tool.Tool, mcpClients []*mcp.Client, engine *context.Engine) *Agent {
    // Register tools in registry
    registry := tool.NewRegistry()
    for _, t := range tools {
        registry.Register(t)
    }
    
    return &Agent{
        modelConf:    modelConf,
        systemPrompt: systemPrompt,
        tools:        tools,
        mcpClients:   mcpClients,
        engine:       engine,
    }
}
```

### 8.2 Tool Calling in Conversation

```go
func (a *Agent) ExecuteTool(ctx context.Context, toolCall shared.ToolCall) (interface{}, error) {
    tool := a.findTool(toolCall.Tool)
    if tool == nil {
        return nil, fmt.Errorf("tool not found: %s", toolCall.Tool)
    }
    
    args := toolCall.ToArgs()
    result, err := tool.Execute(ctx, args)
    if err != nil {
        return nil, fmt.Errorf("tool execution failed: %w", err)
    }
    
    return result, nil
}
```

## 9. Extending the Tool System

### 9.1 Creating a New Tool

```go
type MyTool struct{}

func (m *MyTool) Name() string {
    return "my_tool"
}

func (m *MyTool) Description() string {
    return "My custom tool"
}

func (m *MyTool) Parameters() []tool.Parameter {
    return []tool.Parameter{
        {
            Name:        "param1",
            Type:        "string",
            Description: "First parameter",
            Required:    true,
        },
    }
}

func (m *MyTool) Execute(ctx context.Context, args map[string]interface{}) (interface{}, error) {
    param1, ok := args["param1"].(string)
    if !ok {
        return nil, fmt.Errorf("param1 is required")
    }
    
    // Implement tool logic
    return fmt.Sprintf("Result: %s", param1), nil
}
```

### 9.2 Registering the Tool

```go
registry := tool.NewRegistry()
registry.Register(&MyTool{})
```