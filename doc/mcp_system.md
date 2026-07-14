# TianNiu Backend System Design - MCP System

## 1. Overview

### 1.1 Purpose

The MCP (Model Context Protocol) System enables integration with external plugins that extend agent capabilities through a standardized protocol.

### 1.2 Design Goals

- **Interoperability**: Support plugins written in any language
- **Isolation**: Sandboxed execution environment
- **Extensibility**: Easy plugin development
- **Dynamic Loading**: Hot-reload plugins without restart

### 1.3 Architecture

```
┌──────────────────────────────────────────────────────────────────────────┐
│                           MCP Manager                                   │
│  [Plugin Management] [Process Management] [Protocol Handling]           │
├──────────────────────────────────────────────────────────────────────────┤
│                                    │                                    │
│         ┌──────────────────────────┼──────────────────────────┐         │
│         ▼                          ▼                          ▼         │
│  ┌─────────────┐         ┌─────────────┐         ┌─────────────┐       │
│  │  Python     │         │   Node.js   │         │   Rust      │       │
│  │   Plugin    │         │   Plugin    │         │   Plugin    │       │
│  └─────────────┘         └─────────────┘         └─────────────┘       │
│         │                          │                          │         │
│         └──────────────────────────┴──────────────────────────┘         │
│                                    │                                    │
│                                    ▼                                    │
│                         ┌─────────────┐                                 │
│                         │   Socket    │                                 │
│                         │  Communication│                               │
│                         └─────────────┘                                 │
└──────────────────────────────────────────────────────────────────────────┘
```

## 2. MCP Manager

### 2.1 Responsibilities

- Plugin discovery and loading
- Process lifecycle management
- Protocol communication
- Error handling and recovery

### 2.2 Structure

```go
type Manager struct {
    store       McpStore
    pluginsDir  string
    plugins     map[string]*Client
    portManager *PortManager
    sync.RWMutex
}

type Client struct {
    ID       string
    Name     string
    Path     string
    Process  *os.Process
    Port     int
    Status   Status
    LastPing time.Time
}

type Status string

const (
    StatusRunning Status = "running"
    StatusStopped Status = "stopped"
    StatusError   Status = "error"
)
```

### 2.3 Key Methods

| Method | Description |
|--------|-------------|
| `NewManager(store, pluginsDir)` | Create new MCP manager |
| `LoadPlugins()` | Load all plugins from directory |
| `StartPlugin(id)` | Start plugin process |
| `StopPlugin(id)` | Stop plugin process |
| `RestartPlugin(id)` | Restart plugin |
| `CallPlugin(id, method, args)` | Call plugin method |
| `GetPluginStatus(id)` | Get plugin status |

### 2.4 Plugin Loading

```go
func (m *Manager) LoadPlugins() error {
    m.Lock()
    defer m.Unlock()
    
    files, err := os.ReadDir(m.pluginsDir)
    if err != nil {
        return err
    }
    
    for _, file := range files {
        if file.IsDir() {
            plugin, err := m.loadPlugin(file.Name())
            if err != nil {
                log.Warnf("Failed to load plugin %s: %v", file.Name(), err)
                continue
            }
            m.plugins[plugin.ID] = plugin
        }
    }
    
    return nil
}
```

## 3. MCP Protocol

### 3.1 Protocol Specification

MCP plugins communicate via JSON-RPC over Unix sockets or TCP:

```json
// Request
{
    "jsonrpc": "2.0",
    "id": "1",
    "method": "execute",
    "params": {
        "skill_name": "code_analyzer",
        "args": {
            "file_path": "/path/to/file.py"
        }
    }
}

// Response
{
    "jsonrpc": "2.0",
    "id": "1",
    "result": {
        "success": true,
        "data": {...}
    }
}
```

### 3.2 Protocol Methods

| Method | Description |
|--------|-------------|
| `execute` | Execute a skill |
| `list_skills` | List available skills |
| `get_skill_metadata` | Get skill metadata |
| `ping` | Health check |

### 3.3 Error Handling

```json
{
    "jsonrpc": "2.0",
    "id": "1",
    "error": {
        "code": -32601,
        "message": "Method not found",
        "data": "Skill 'unknown' not found"
    }
}
```

## 4. Plugin Structure

### 4.1 Plugin Manifest

Each plugin must have a `mcp.yaml` manifest:

```yaml
id: "python_skills"
name: "Python Skills"
description: "Python-based skills for TianNiu"
version: "1.0.0"
runtime: "python3"
main: "main.py"
requirements:
  - mcp>=0.1.0
```

### 4.2 Plugin Implementation

```python
from mcp import MCPPlugin

class MyPlugin(MCPPlugin):
    def __init__(self):
        super().__init__("my_plugin")
    
    def execute(self, skill_name, args):
        if skill_name == "analyze_code":
            return self.analyze_code(args)
        raise ValueError(f"Unknown skill: {skill_name}")
    
    def analyze_code(self, args):
        file_path = args.get("file_path")
        # Implementation...
        return {"result": "analysis complete"}
```

### 4.3 Plugin Directory Structure

```
python_skills/
├── mcp.yaml          # Manifest
├── main.py           # Main entry point
├── requirements.txt  # Dependencies
└── skills/           # Skill implementations
    └── code_analyzer.py
```

## 5. Plugin Lifecycle

### 5.1 Startup Process

```
MCP Manager starts
    │
    ▼
LoadPlugins()
    │
    ▼
For each plugin:
    │
    ├─→ Parse mcp.yaml
    │
    ├─→ Allocate port
    │
    ├─→ Start process
    │       └─→ python main.py --port <port>
    │
    ├─→ Wait for ready signal
    │
    └─→ Register in plugins map
```

### 5.2 Shutdown Process

```
MCP Manager shutdown
    │
    ▼
For each running plugin:
    │
    ├─→ Send shutdown signal
    │
    ├─→ Wait for graceful exit (10s timeout)
    │
    └─→ Force kill if needed
```

### 5.3 Health Monitoring

```go
func (m *Manager) MonitorPlugins() {
    ticker := time.NewTicker(30 * time.Second)
    defer ticker.Stop()
    
    for range ticker.C {
        m.RLock()
        for _, plugin := range m.plugins {
            if plugin.Status == StatusRunning {
                go m.checkHealth(plugin)
            }
        }
        m.RUnlock()
    }
}

func (m *Manager) checkHealth(plugin *Client) {
    _, err := m.pingPlugin(plugin)
    if err != nil {
        log.Warnf("Plugin %s is unhealthy: %v", plugin.ID, err)
        plugin.Status = StatusError
    }
}
```

## 6. Port Management

### 6.1 Port Allocation

```go
type PortManager struct {
    usedPorts map[int]bool
    minPort   int
    maxPort   int
    sync.Mutex
}

func (p *PortManager) AllocatePort() (int, error) {
    p.Lock()
    defer p.Unlock()
    
    for port := p.minPort; port <= p.maxPort; port++ {
        if !p.usedPorts[port] {
            p.usedPorts[port] = true
            return port, nil
        }
    }
    
    return 0, fmt.Errorf("no available ports")
}

func (p *PortManager) ReleasePort(port int) {
    p.Lock()
    defer p.Unlock()
    delete(p.usedPorts, port)
}
```

### 6.2 Configuration

```yaml
mcp:
  port_range:
    min: 50000
    max: 51000
```

## 7. Security Considerations

### 7.1 Plugin Isolation

- **Process isolation**: Each plugin runs in its own process
- **Network isolation**: Plugins only communicate via MCP protocol
- **Resource limits**: CPU, memory, and file system limits

### 7.2 Authentication

- **API keys**: Plugins must authenticate with the manager
- **TLS encryption**: Encrypted communication
- **Access control**: Role-based access to plugins

### 7.3 Input Validation

- **Schema validation**: Validate all inputs against schema
- **Sanitization**: Sanitize inputs to prevent injection attacks
- **Rate limiting**: Limit plugin calls per user

## 8. Error Handling

### 8.1 Error Types

| Error | Description |
|-------|-------------|
| `ErrPluginNotFound` | Plugin not found |
| `ErrPluginNotRunning` | Plugin is not running |
| `ErrPortAllocationFailed` | Failed to allocate port |
| `ErrPluginTimeout` | Plugin response timed out |
| `ErrProtocolError` | Invalid protocol message |

### 8.2 Error Recovery

- **Plugin crash**: Restart plugin automatically
- **Timeout**: Retry or return error
- **Protocol error**: Log and return error

## 9. Configuration

### 9.1 MCP Configuration

```yaml
mcp:
  enabled: true
  plugins_dir: ./mcp_plugins
  port_range:
    min: 50000
    max: 51000
  max_restart_attempts: 3
  restart_delay: 5s
  health_check_interval: 30s
```

### 9.2 Environment Variables

| Variable | Description |
|----------|-------------|
| `MCP_PLUGINS_DIR` | Directory for MCP plugins |
| `MCP_PORT_MIN` | Minimum port number |
| `MCP_PORT_MAX` | Maximum port number |

## 10. MCP Integration with Agent

### 10.1 Skill Resolution

```go
func (a *Agent) resolveSkill(skillID string) (skill.ExecutableSkill, error) {
    // 1. Check built-in skills
    if skill, ok := a.skillManager.GetSkill(skillID); ok {
        return skill, nil
    }
    
    // 2. Check MCP plugins
    for _, client := range a.mcpClients {
        if client.Status == StatusRunning {
            skills, err := client.ListSkills()
            if err != nil {
                continue
            }
            for _, skill := range skills {
                if skill.ID == skillID {
                    return &MCPSkill{client: client, skill: skill}, nil
                }
            }
        }
    }
    
    return nil, fmt.Errorf("skill not found: %s", skillID)
}
```

### 10.2 MCP Skill Execution

```go
type MCPSkill struct {
    client *mcp.Client
    skill  SkillMetadata
}

func (s *MCPSkill) Execute(ctx context.Context, args map[string]interface{}) (interface{}, error) {
    result, err := s.client.Call(ctx, "execute", map[string]interface{}{
        "skill_name": s.skill.ID,
        "args":       args,
    })
    if err != nil {
        return nil, err
    }
    
    return result, nil
}
```