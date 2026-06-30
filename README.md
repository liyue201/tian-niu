# tian-niu

一个轻量级的AI聊天Agent，基于React和@assistant-ui构建，支持流式消息处理和工具调用。

## 功能特点

- **智能对话**: 支持与AI模型进行流畅的对话交互
- **多线程管理**: 支持创建、查看、重命名和删除对话线程
- **流式消息**: 基于Server-Sent Events实现实时消息推送
- **工具调用**: 支持AI调用外部工具获取信息
- **推理面板**: 展示AI的思考过程
- **模型切换**: 支持配置前端和后端两种不同的LLM模型

## 技术栈

- **前端框架**: React 19 + TypeScript
- **构建工具**: Vite 8
- **样式框架**: Tailwind CSS 4.0
- **UI组件**: @assistant-ui/react, @radix-ui/react
- **图标库**: lucide-react
- **流式处理**: @microsoft/fetch-event-source
- **状态管理**: @assistant-ui/store

## 快速开始

### 安装依赖

```bash
cd frontend
npm install
```

### 配置环境

1. 复制配置文件示例并修改：

```bash
cp config.example.json config.json
```

2. 编辑 `config.json`，配置LLM提供商：

```json
{
"llm_providers": {
    "front_model": {
    "base_url": "https://api.openai.com/v1",
    "model": "gpt-4o",
    "api_key": "your-api-key",
    "context_window": 200000
    },
    "back_model": {
    "base_url": "https://api.openai.com/v1",
    "model": "gpt-4o-mini",
    "api_key": "your-api-key",
    "context_window": 128000
    }
}
}
```

### 开发模式

```bash
cd frontend
npm run dev
```

访问 http://localhost:5173 查看应用。

### 生产构建

```bash
cd frontend
npm run build
```

构建产物将输出到 `frontend/dist` 目录。

## 项目结构

```
├── frontend/                 # 前端应用
│   ├── src/
│   │   ├── components/       # 组件目录
│   │   │   ├── ui/          # UI基础组件
│   │   │   ├── assistant-ui/ # @assistant-ui集成组件
│   │   │   ├── ChatPanel.tsx # 聊天面板
│   │   │   ├── Sidebar.tsx   # 侧边栏
│   │   │   └── MessageBubble.tsx # 消息气泡
│   │   ├── lib/             # 工具函数
│   │   ├── App.tsx          # 主应用组件
│   │   └── api.ts           # API接口
│   ├── index.html
│   ├── package.json
│   └── vite.config.ts
├── config.json              # 应用配置
├── config.example.json      # 配置示例
├── LICENSE
└── README.md
```

## 配置说明

### LLM提供商配置

| 参数 | 说明 | 默认值 |
|------|------|--------|
| `base_url` | LLM API基础地址 | - |
| `model` | 模型名称 | - |
| `api_key` | API密钥 | - |
| `context_window` | 上下文窗口大小 | 128000 |

### 支持的模型

- OpenAI: gpt-4o, gpt-4o-mini, gpt-4-turbo
- 智谱AI: GLM-5.2, GLM-4
- 其他兼容OpenAI API的模型

## 开发指南

### 添加新功能

1. 在 `frontend/src/components/` 目录下创建新组件
2. 在 `frontend/src/api.ts` 中添加API接口
3. 在 `App.tsx` 中集成新组件

### 代码规范

- 使用TypeScript进行类型检查
- 遵循ESLint规则
- 使用Tailwind CSS 4.0的CSS-first语法

## 许可证

MIT License