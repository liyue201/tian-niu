package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/openai/openai-go/v3"
	shared2 "github.com/openai/openai-go/v3/shared"
	log "github.com/sirupsen/logrus"
	"github.com/tianniu-ai/tianniu/pkg/agent/tool"
	"github.com/tianniu-ai/tianniu/pkg/model"
	"github.com/tianniu-ai/tianniu/pkg/shared"
)

type McpStatus string

const (
	McpStatusInstalled McpStatus = "installed"
	McpStatusEnabled   McpStatus = "enabled"
	McpStatusDisabled  McpStatus = "disabled"
	McpStatusRemoved   McpStatus = "removed"
)

type McpType string

const (
	McpTypeSystem McpType = "system"
	McpTypeUser   McpType = "user"
)

type McpServer struct {
	ID          string       `json:"id"`
	Name        string       `json:"name"`
	Description string       `json:"description"`
	Status      McpStatus    `json:"status"`
	Type        McpType      `json:"type"`
	UserID      string       `json:"user_id"`
	Config      ServerConfig `json:"config"`
	InstalledAt time.Time    `json:"installed_at"`
	UpdatedAt   time.Time    `json:"updated_at"`
}

type Client struct {
	name         string
	client       *mcp.Client
	serverConfig ServerConfig

	session *mcp.ClientSession
	tools   []tool.Tool
}

func initRunningVars() map[string]string {
	runningVars := map[string]string{
		"${workspaceFolder}": shared.GetWorkspaceDir(),
	}
	return runningVars
}

func NewMcpToolProvider(name string, server ServerConfig) *Client {
	return &Client{
		name: name,
		client: mcp.NewClient(&mcp.Implementation{
			Name:    "tianniu-mcp-client",
			Title:   "TianNiu",
			Version: "v1.0.0",
		}, nil),
		serverConfig: server.ReplacePlaceholders(initRunningVars()),
		tools:        make([]tool.Tool, 0),
	}
}

func (e *Client) Name() string {
	return e.name
}

func (e *Client) connect(ctx context.Context) error {
	if e.session != nil && e.session.Ping(ctx, &mcp.PingParams{}) == nil {
		return nil
	}
	var err error
	if e.serverConfig.IsStdio() {
		cmd := exec.Command(e.serverConfig.Command, e.serverConfig.Args...)
		for k, v := range e.serverConfig.Env {
			cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
		}
		e.session, err = e.client.Connect(ctx, &mcp.CommandTransport{Command: cmd}, nil)
	} else {
		e.session, err = e.client.Connect(ctx, &mcp.StreamableClientTransport{Endpoint: e.serverConfig.Url}, nil)
	}
	if err != nil {
		return err
	}
	return nil
}

func (e *Client) RefreshTools(ctx context.Context) error {
	if err := e.connect(ctx); err != nil {
		return err
	}

	mcpToolResult, err := e.session.ListTools(ctx, &mcp.ListToolsParams{})
	if err != nil {
		return err
	}

	e.tools = make([]tool.Tool, 0)
	for _, mcpTool := range mcpToolResult.Tools {
		agentTool := &Tool{
			client:   e,
			toolName: mcpTool.Name,
			session:  e.session,
			mcpTool:  mcpTool,
		}
		e.tools = append(e.tools, agentTool)
	}
	return nil
}

func (e *Client) GetTools() []tool.Tool {
	return e.tools
}

func (e *Client) callTool(ctx context.Context, toolName string, argumentsInJSON string) (string, error) {
	if err := e.connect(ctx); err != nil {
		return "", err
	}
	mcpResult, err := e.session.CallTool(ctx, &mcp.CallToolParams{
		Name:      toolName,
		Arguments: json.RawMessage(argumentsInJSON),
	})
	if err != nil {
		log.Printf("failed to call tool: %v", err)
		return "", err
	}

	var builder strings.Builder
	for _, content := range mcpResult.Content {
		if textContent, ok := content.(*mcp.TextContent); ok {
			builder.WriteString(textContent.Text)
		}
	}
	return builder.String(), nil
}

type Tool struct {
	toolName string
	client   *Client
	session  *mcp.ClientSession
	mcpTool  *mcp.Tool
}

func (t *Tool) ToolName() string {
	return fmt.Sprintf("tianniu_mcp__%s__%s", t.client.Name(), t.toolName)
}

func (t *Tool) Info() openai.ChatCompletionToolUnionParam {
	return openai.ChatCompletionFunctionTool(shared2.FunctionDefinitionParam{
		Description: openai.String(t.mcpTool.Description),
		Name:        t.ToolName(),
		Parameters:  t.mcpTool.InputSchema.(map[string]any),
	})
}

func (t *Tool) Execute(ctx context.Context, argumentsInJSON string) (string, error) {
	return t.client.callTool(ctx, t.toolName, argumentsInJSON)
}

type McpStore interface {
	GetAll() ([]*McpServer, error)
	GetByID(id string) (*McpServer, error)
	GetByName(name string) (*McpServer, error)
	GetByUserID(userID string) ([]*McpServer, error)
	GetSystemMcpServers() ([]*McpServer, error)
	GetUserMcpServers(userID string) ([]*McpServer, error)
	GetMcpServerForUser(userID, serverName string) (*McpServer, error)
	Save(server *McpServer) error
	Delete(id string) error
	UpdateStatus(id string, status McpStatus) error
}

type InstallOptions struct {
	Force bool `json:"force"`
}

type Manager struct {
	store McpStore
}

func NewManager(store McpStore) *Manager {
	return &Manager{
		store: store,
	}
}

func (m *Manager) InstallSystemMcpServer(name string, config ServerConfig, options InstallOptions) (*McpServer, error) {
	existingServer, _ := m.store.GetByName(name)
	if existingServer != nil && !options.Force {
		return nil, fmt.Errorf("system mcp server '%s' is already installed", name)
	}

	server := &McpServer{
		ID:          uuid.NewString(),
		Name:        name,
		Description: "",
		Status:      McpStatusEnabled,
		Type:        McpTypeSystem,
		UserID:      "",
		Config:      config,
		InstalledAt: time.Now(),
		UpdatedAt:   time.Now(),
	}

	if existingServer != nil {
		server.ID = existingServer.ID
	}

	if err := m.store.Save(server); err != nil {
		return nil, fmt.Errorf("failed to save mcp server: %w", err)
	}

	log.Infof("System MCP server '%s' installed successfully", server.Name)
	return server, nil
}

func (m *Manager) InstallUserMcpServer(userID, name string, config ServerConfig, options InstallOptions) (*McpServer, error) {
	if userID == "" {
		return nil, fmt.Errorf("user ID is required")
	}

	existingServer, _ := m.store.GetMcpServerForUser(userID, name)
	if existingServer != nil && existingServer.Type == McpTypeUser && !options.Force {
		return nil, fmt.Errorf("user mcp server '%s' is already installed", name)
	}

	server := &McpServer{
		ID:          uuid.NewString(),
		Name:        name,
		Description: "",
		Status:      McpStatusEnabled,
		Type:        McpTypeUser,
		UserID:      userID,
		Config:      config,
		InstalledAt: time.Now(),
		UpdatedAt:   time.Now(),
	}

	if existingServer != nil && existingServer.Type == McpTypeUser {
		server.ID = existingServer.ID
	}

	if err := m.store.Save(server); err != nil {
		return nil, fmt.Errorf("failed to save mcp server: %w", err)
	}

	log.Infof("User MCP server '%s' installed successfully for user '%s'", server.Name, userID)
	return server, nil
}

func (m *Manager) Uninstall(serverID string) error {
	server, err := m.store.GetByID(serverID)
	if err != nil {
		return err
	}

	if err := m.store.Delete(serverID); err != nil {
		return fmt.Errorf("failed to delete mcp server from store: %w", err)
	}

	log.Infof("MCP server '%s' uninstalled successfully", server.Name)
	return nil
}

func (m *Manager) Enable(serverID string) error {
	return m.store.UpdateStatus(serverID, McpStatusEnabled)
}

func (m *Manager) Disable(serverID string) error {
	return m.store.UpdateStatus(serverID, McpStatusDisabled)
}

func (m *Manager) GetAllMcpServers() ([]*McpServer, error) {
	return m.store.GetAll()
}

func (m *Manager) GetMcpServerByID(id string) (*McpServer, error) {
	return m.store.GetByID(id)
}

func (m *Manager) GetMcpServerByName(name string) (*McpServer, error) {
	return m.store.GetByName(name)
}

func (m *Manager) GetSystemMcpServers() ([]*McpServer, error) {
	return m.store.GetSystemMcpServers()
}

func (m *Manager) GetUserMcpServers(userID string) ([]*McpServer, error) {
	return m.store.GetUserMcpServers(userID)
}

func (m *Manager) GetMcpServersForUser(userID string) ([]*McpServer, error) {
	systemServers, err := m.store.GetSystemMcpServers()
	if err != nil {
		return nil, err
	}

	userServers, err := m.store.GetUserMcpServers(userID)
	if err != nil {
		return nil, err
	}

	allServers := append(systemServers, userServers...)
	return allServers, nil
}

func (m *Manager) GetMcpServerForUser(userID, serverName string) (*McpServer, error) {
	return m.store.GetMcpServerForUser(userID, serverName)
}

func (m *Manager) LoadSystemMcpServers(configPath string) error {
	serverMap, err := LoadMcpServerConfig(configPath)
	if err != nil {
		return fmt.Errorf("failed to load MCP server configuration: %w", err)
	}

	currentServerNames := make(map[string]bool)

	for name, config := range serverMap {
		currentServerNames[name] = true

		existingServer, err := m.store.GetByName(name)
		if err != nil || existingServer == nil {
			server := &McpServer{
				ID:          uuid.NewString(),
				Name:        name,
				Description: "",
				Status:      McpStatusEnabled,
				Type:        McpTypeSystem,
				UserID:      "",
				Config:      config,
				InstalledAt: time.Now(),
				UpdatedAt:   time.Now(),
			}

			if err := m.store.Save(server); err != nil {
				log.Warnf("Failed to save system MCP server '%s': %v", name, err)
			} else {
				log.Infof("Installed system MCP server '%s'", name)
			}
			continue
		}

		if existingServer.Type != McpTypeSystem {
			continue
		}

		needUpdate := false
		if existingServer.Config.Command != config.Command ||
			!equalStringSlice(existingServer.Config.Args, config.Args) ||
			!equalStringMap(existingServer.Config.Env, config.Env) ||
			existingServer.Config.Url != config.Url {
			needUpdate = true
		}

		if needUpdate {
			existingServer.Config = config
			existingServer.UpdatedAt = time.Now()

			if err := m.store.Save(existingServer); err != nil {
				log.Warnf("Failed to update system MCP server '%s': %v", name, err)
			} else {
				log.Infof("Updated system MCP server '%s'", name)
			}
		}
	}

	existingServers, err := m.store.GetSystemMcpServers()
	if err != nil {
		log.Warnf("Failed to get existing system MCP servers: %v", err)
	} else {
		for _, existingServer := range existingServers {
			if !currentServerNames[existingServer.Name] {
				if err := m.store.Delete(existingServer.ID); err != nil {
					log.Warnf("Failed to remove missing system MCP server '%s': %v", existingServer.Name, err)
				} else {
					log.Infof("Removed missing system MCP server '%s'", existingServer.Name)
				}
			}
		}
	}

	return nil
}

func equalStringSlice(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func equalStringMap(a, b map[string]string) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if b[k] != v {
			return false
		}
	}
	return true
}

func convertMcpServerToModel(m *McpServer) *model.McpServer {
	configJSON, err := json.Marshal(m.Config)
	if err != nil {
		log.Errorf("Failed to marshal MCP server config: %v", err)
		configJSON = []byte("{}")
	}
	return &model.McpServer{
		ID:          m.ID,
		Name:        m.Name,
		Description: m.Description,
		Status:      string(m.Status),
		Type:        string(m.Type),
		UserID:      m.UserID,
		Config:      string(configJSON),
		InstalledAt: m.InstalledAt,
		UpdatedAt:   m.UpdatedAt,
	}
}

func convertModelToMcpServer(m *model.McpServer) *McpServer {
	var config ServerConfig
	if m.Config != "" {
		if err := json.Unmarshal([]byte(m.Config), &config); err != nil {
			log.Errorf("Failed to unmarshal MCP server config: %v", err)
			config = ServerConfig{}
		}
	}
	return &McpServer{
		ID:          m.ID,
		Name:        m.Name,
		Description: m.Description,
		Status:      McpStatus(m.Status),
		Type:        McpType(m.Type),
		UserID:      m.UserID,
		Config:      config,
		InstalledAt: m.InstalledAt,
		UpdatedAt:   m.UpdatedAt,
	}
}

func convertModelMcpServersToMcpServer(models []*model.McpServer) []*McpServer {
	result := make([]*McpServer, 0, len(models))
	for _, m := range models {
		result = append(result, convertModelToMcpServer(m))
	}
	return result
}
