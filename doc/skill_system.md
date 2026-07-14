# TianNiu Backend System Design - Skill System

## 1. Overview

### 1.1 Purpose

The Skill System provides a plugin-based framework for extending agent capabilities through modular, reusable skills.

### 1.2 Design Goals

- **Extensibility**: Allow third-party skill development
- **Isolation**: Sandboxed execution environment
- **Security**: Permission-based access control
- **Discovery**: Automatic skill discovery and registration

### 1.3 Architecture

```
┌──────────────────────────────────────────────────────────────────────────┐
│                           Skill Manager                                │
│  [Skill Discovery] [Loading] [Execution] [Lifecycle Management]        │
├──────────────────────────────────────────────────────────────────────────┤
│                                    │                                    │
│         ┌──────────────────────────┼──────────────────────────┐         │
│         ▼                          ▼                          ▼         │
│  ┌─────────────┐         ┌─────────────┐         ┌─────────────┐       │
│  │  Built-in   │         │  User       │         │   MCP       │       │
│  │   Skills    │         │  Installed  │         │  Skills     │       │
│  └─────────────┘         └─────────────┘         └─────────────┘       │
│         │                          │                          │         │
│         └──────────────────────────┴──────────────────────────┘         │
│                                    │                                    │
│                                    ▼                                    │
│                         ┌─────────────┐                                 │
│                         │  Tool       │                                 │
│                         │  Registry   │                                 │
│                         └─────────────┘                                 │
└──────────────────────────────────────────────────────────────────────────┘
```

## 2. Skill Manager

### 2.1 Responsibilities

- Skill discovery and loading
- Skill lifecycle management
- Skill execution coordination
- User-specific skill management

### 2.2 Structure

```go
type Manager struct {
    store      SkillStore
    skillsDir  string
    skills     map[string]*Skill
    userSkills map[string][]string  // key: userId, value: skill IDs
    sync.RWMutex
}

type Skill struct {
    ID          string
    Name        string
    Description string
    Version     string
    Tool        tool.Tool
    Enabled     bool
    UserID      string
}
```

### 2.3 Key Methods

| Method | Description |
|--------|-------------|
| `NewManager(store, skillsDir)` | Create new skill manager |
| `LoadSkills()` | Load all skills from directory |
| `GetSkill(id)` | Retrieve skill by ID |
| `GetUserSkills(userId)` | Get skills enabled for user |
| `EnableSkill(userId, skillId)` | Enable skill for user |
| `DisableSkill(userId, skillId)` | Disable skill for user |
| `InstallSkill(skillArchive)` | Install new skill |
| `UninstallSkill(skillId)` | Uninstall skill |

### 2.4 Skill Loading

```go
func (m *Manager) LoadSkills() error {
    m.Lock()
    defer m.Unlock()
    
    files, err := os.ReadDir(m.skillsDir)
    if err != nil {
        return err
    }
    
    for _, file := range files {
        if file.IsDir() {
            skill, err := m.loadSkillFromDir(file.Name())
            if err != nil {
                log.Warnf("Failed to load skill %s: %v", file.Name(), err)
                continue
            }
            m.skills[skill.ID] = skill
        }
    }
    
    return nil
}
```

## 3. Skill Structure

### 3.1 Skill Manifest

Each skill must have a `skill.yaml` manifest file:

```yaml
id: "code_analyzer"
name: "Code Analyzer"
description: "Analyze and understand code files"
version: "1.0.0"
author: "TianNiu Team"
requirements:
  - python >= 3.8
dependencies:
  - ast
  - json
```

### 3.2 Skill Implementation

Skills can be implemented in multiple languages:
- Go (built-in)
- Python (via MCP)
- JavaScript (via MCP)

### 3.3 Skill Execution Interface

```go
type ExecutableSkill interface {
    Execute(ctx context.Context, args map[string]interface{}) (interface{}, error)
    GetMetadata() SkillMetadata
}
```

## 4. Skill Store

### 4.1 Interface

```go
type SkillStore interface {
    Save(ctx context.Context, skill *Skill) error
    Get(ctx context.Context, id string) (*Skill, error)
    List(ctx context.Context) ([]*Skill, error)
    Delete(ctx context.Context, id string) error
    GetUserSkills(ctx context.Context, userId string) ([]*Skill, error)
    AddUserSkill(ctx context.Context, userId, skillId string) error
    RemoveUserSkill(ctx context.Context, userId, skillId string) error
}
```

### 4.2 Database Schema

```sql
CREATE TABLE skills (
    id VARCHAR(36) PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    description TEXT,
    version VARCHAR(20),
    path VARCHAR(255),
    enabled BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE user_skills (
    user_id VARCHAR(36),
    skill_id VARCHAR(36),
    enabled BOOLEAN DEFAULT true,
    PRIMARY KEY (user_id, skill_id),
    FOREIGN KEY (skill_id) REFERENCES skills(id)
);
```

## 5. Skill Execution Flow

### 5.1 Execution Pipeline

```
Agent receives LLM response with tool call
    │
    ▼
SkillManager.GetSkill(skillId)
    │
    ▼
Skill.Validate(args)
    │
    ├─→ [Valid] → Continue
    │
    └─→ [Invalid] → Return error
    │
    ▼
Skill.Execute(ctx, args)
    │
    ├─→ [Built-in] → Direct execution
    │
    └─→ [MCP] → MCP Client call
    │
    ▼
Return result to Agent
    │
    ▼
Agent processes result and continues conversation
```

### 5.2 Skill Validation

```go
func (s *Skill) Validate(args map[string]interface{}) error {
    // 1. Check required parameters
    required := []string{"file_path"}
    for _, param := range required {
        if _, ok := args[param]; !ok {
            return fmt.Errorf("missing required parameter: %s", param)
        }
    }
    
    // 2. Validate parameter types
    if _, ok := args["file_path"].(string); !ok {
        return fmt.Errorf("file_path must be string")
    }
    
    return nil
}
```

## 6. Built-in Skills

### 6.1 Code Analysis Skills

| Skill | Description |
|-------|-------------|
| **CodeAnalyzer** | Analyze code structure and dependencies |
| **BugDetector** | Detect potential bugs in code |
| **CodeOptimizer** | Suggest code optimizations |
| **DocGenerator** | Generate documentation for code |

### 6.2 File Operation Skills

| Skill | Description |
|-------|-------------|
| **FileReader** | Read file contents |
| **FileWriter** | Write content to file |
| **FileSearch** | Search for files |
| **DirectoryList** | List directory contents |

### 6.3 Development Skills

| Skill | Description |
|-------|-------------|
| **GitSkill** | Git operations |
| **BuildSkill** | Build project |
| **TestRunner** | Run tests |
| **DependencyManager** | Manage dependencies |

## 7. Skill Installation

### 7.1 Installation Process

```
User uploads skill archive
    │
    ▼
Validate archive structure
    │
    ├─→ [Invalid] → Return error
    │
    └─→ [Valid] → Continue
    │
    ▼
Extract archive to skills directory
    │
    ▼
Parse skill.yaml manifest
    │
    ▼
Register in database
    │
    ▼
Load skill into memory
    │
    ▼
Return success
```

### 7.2 Archive Structure

```
skill_archive.zip/
├── skill.yaml          # Manifest
├── main.py             # Main implementation
├── requirements.txt    # Dependencies
└── README.md           # Documentation
```

## 8. Security Considerations

### 8.1 Permission Model

| Role | Permissions |
|------|-------------|
| **Admin** | Install, uninstall, enable, disable all skills |
| **User** | Enable/disable skills for personal use |
| **Guest** | Read-only access to public skills |

### 8.2 Sandboxing

- **File system restrictions**: Limit access to specific directories
- **Network restrictions**: Allow only approved external calls
- **Resource limits**: CPU, memory, and time limits per execution

### 8.3 Input Sanitization

- Validate all inputs before execution
- Sanitize file paths to prevent path traversal
- Restrict command execution to approved commands

## 9. Configuration

### 9.1 Configuration Structure

```yaml
skill:
  enabled: true
  skills_dir: ./skills
  max_execution_time: 30s
  sandbox_enabled: true
  allowed_directories:
    - /workspace
    - /tmp
```

### 9.2 Environment Variables

| Variable | Description |
|----------|-------------|
| `SKILLS_DIR` | Directory for skill installation |
| `SKILL_MAX_EXEC_TIME` | Maximum execution time per skill |
| `SKILL_SANDBOX_ENABLED` | Enable sandbox mode |

## 10. Error Handling

### 10.1 Error Types

| Error | Description |
|-------|-------------|
| `ErrSkillNotFound` | Skill not found |
| `ErrSkillDisabled` | Skill is disabled |
| `ErrInvalidArgs` | Invalid arguments |
| `ErrExecutionTimeout` | Execution timed out |
| `ErrPermissionDenied` | Permission denied |

### 10.2 Error Recovery

- **Timeout**: Kill process and return timeout error
- **Crash**: Log error and return graceful error message
- **Permission denied**: Return permission error