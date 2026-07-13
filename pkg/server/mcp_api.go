package server

import (
	"github.com/gin-gonic/gin"
	"github.com/tianniu-ai/tianniu/pkg/agent/mcp"
)

type McpAPI struct {
	mcpManager *mcp.Manager
}

func NewMcpAPI(mcpManager *mcp.Manager) *McpAPI {
	return &McpAPI{mcpManager: mcpManager}
}

func (api *McpAPI) RegisterRoutes(g *gin.RouterGroup) {
	g.GET("/mcps", api.listMcpServersForUser)
	g.GET("/mcps/system", api.listSystemMcpServers)
	g.GET("/mcps/user", api.listUserMcpServers)
	g.GET("/mcps/:id", api.getMcpServer)
	g.POST("/mcps/install", api.installUserMcpServer)
	g.POST("/mcps/:id/uninstall", api.uninstallMcpServer)
	g.POST("/mcps/:id/enable", api.enableMcpServer)
	g.POST("/mcps/:id/disable", api.disableMcpServer)
}

func (api *McpAPI) getUserID(c *gin.Context) string {
	userID, exists := c.Get("userID")
	if !exists {
		return ""
	}
	return userID.(string)
}

func (api *McpAPI) listMcpServersForUser(c *gin.Context) {
	userID := api.getUserID(c)
	if userID == "" {
		respondError(c, StatusUsernameError, nil)
		return
	}

	servers, err := api.mcpManager.GetMcpServersForUser(userID)
	if err != nil {
		respondError(c, StatusInternalServerError, err)
		return
	}
	respondSuccess(c, servers)
}

func (api *McpAPI) listSystemMcpServers(c *gin.Context) {
	servers, err := api.mcpManager.GetSystemMcpServers()
	if err != nil {
		respondError(c, StatusInternalServerError, err)
		return
	}
	respondSuccess(c, servers)
}

func (api *McpAPI) listUserMcpServers(c *gin.Context) {
	userID := api.getUserID(c)
	if userID == "" {
		respondError(c, StatusUsernameError, nil)
		return
	}

	servers, err := api.mcpManager.GetUserMcpServers(userID)
	if err != nil {
		respondError(c, StatusInternalServerError, err)
		return
	}
	respondSuccess(c, servers)
}

func (api *McpAPI) getMcpServer(c *gin.Context) {
	id := c.Param("id")
	serverData, err := api.mcpManager.GetMcpServerByID(id)
	if err != nil {
		respondError(c, StatusInternalServerError, err)
		return
	}
	respondSuccess(c, serverData)
}

type InstallMcpServerRequest struct {
	Name    string            `json:"name"`
	Command string            `json:"command,omitempty"`
	Args    []string          `json:"args,omitempty"`
	Env     map[string]string `json:"env,omitempty"`
	Url     string            `json:"url,omitempty"`
	Force   bool              `json:"force"`
}

func (api *McpAPI) installUserMcpServer(c *gin.Context) {
	userID := api.getUserID(c)
	if userID == "" {
		respondError(c, StatusUsernameError, nil)
		return
	}

	var req InstallMcpServerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, StatusInvalidParam, err)
		return
	}

	if req.Name == "" {
		respondError(c, StatusInvalidParam, nil)
		return
	}

	config := mcp.ServerConfig{
		Command: req.Command,
		Args:    req.Args,
		Env:     req.Env,
		Url:     req.Url,
	}

	options := mcp.InstallOptions{Force: req.Force}
	installedServer, err := api.mcpManager.InstallUserMcpServer(userID, req.Name, config, options)
	if err != nil {
		respondError(c, StatusInternalServerError, err)
		return
	}
	respondSuccess(c, installedServer)
}

func (api *McpAPI) uninstallMcpServer(c *gin.Context) {
	id := c.Param("id")

	if err := api.mcpManager.Uninstall(id); err != nil {
		respondError(c, StatusInternalServerError, err)
		return
	}
	respondSuccess(c, nil)
}

func (api *McpAPI) enableMcpServer(c *gin.Context) {
	id := c.Param("id")
	if err := api.mcpManager.Enable(id); err != nil {
		respondError(c, StatusInternalServerError, err)
		return
	}
	respondSuccess(c, nil)
}

func (api *McpAPI) disableMcpServer(c *gin.Context) {
	id := c.Param("id")
	if err := api.mcpManager.Disable(id); err != nil {
		respondError(c, StatusInternalServerError, err)
		return
	}
	respondSuccess(c, nil)
}
