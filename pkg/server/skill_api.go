package server

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/tianniu-ai/tianniu/pkg/skill"
)

type SkillAPI struct {
	skillManager *skill.Manager
}

func NewSkillAPI(skillManager *skill.Manager) *SkillAPI {
	return &SkillAPI{skillManager: skillManager}
}

func (api *SkillAPI) RegisterRoutes(g *gin.RouterGroup) {
	g.GET("/skills", api.listSkillsForUser)
	g.GET("/skills/system", api.listSystemSkills)
	g.GET("/skills/user", api.listUserSkills)
	g.GET("/skills/:id", api.getSkill)
	g.POST("/skills/install", api.installUserSkill)
	g.POST("/skills/system/install", api.installSystemSkill)
	g.POST("/skills/:id/uninstall", api.uninstallSkill)
	g.POST("/skills/:id/enable", api.enableSkill)
	g.POST("/skills/:id/disable", api.disableSkill)
}

func (api *SkillAPI) getUserID(c *gin.Context) string {
	userID, exists := c.Get("userID")
	if !exists {
		return ""
	}
	return userID.(string)
}

func (api *SkillAPI) listSkillsForUser(c *gin.Context) {
	userID := api.getUserID(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "msg": "User ID not found"})
		return
	}

	skills, err := api.skillManager.GetSkillsForUser(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 200, "data": skills})
}

func (api *SkillAPI) listSystemSkills(c *gin.Context) {
	skills, err := api.skillManager.GetSystemSkills()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 200, "data": skills})
}

func (api *SkillAPI) listUserSkills(c *gin.Context) {
	userID := api.getUserID(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "msg": "User ID not found"})
		return
	}

	skills, err := api.skillManager.GetUserSkills(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 200, "data": skills})
}

func (api *SkillAPI) getSkill(c *gin.Context) {
	id := c.Param("id")
	skillData, err := api.skillManager.GetSkillByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "msg": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 200, "data": skillData})
}

type InstallSkillRequest struct {
	SkillPath string `json:"skill_path"`
	Force     bool   `json:"force"`
}

func (api *SkillAPI) installUserSkill(c *gin.Context) {
	userID := api.getUserID(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "msg": "User ID not found"})
		return
	}

	var req InstallSkillRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "Invalid request body"})
		return
	}

	options := skill.InstallOptions{Force: req.Force}
	installedSkill, err := api.skillManager.InstallUserSkill(userID, req.SkillPath, options)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 200, "data": installedSkill})
}

func (api *SkillAPI) installSystemSkill(c *gin.Context) {
	var req InstallSkillRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "Invalid request body"})
		return
	}

	options := skill.InstallOptions{Force: req.Force}
	installedSkill, err := api.skillManager.InstallSystemSkill(req.SkillPath, options)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 200, "data": installedSkill})
}

type UninstallSkillRequest struct {
	KeepConfig bool `json:"keep_config"`
}

func (api *SkillAPI) uninstallSkill(c *gin.Context) {
	id := c.Param("id")
	var req UninstallSkillRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "Invalid request body"})
		return
	}

	options := skill.UninstallOptions{KeepConfig: req.KeepConfig}
	if err := api.skillManager.Uninstall(id, options); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 200, "msg": "Skill uninstalled successfully"})
}

func (api *SkillAPI) enableSkill(c *gin.Context) {
	id := c.Param("id")
	if err := api.skillManager.Enable(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 200, "msg": "Skill enabled successfully"})
}

func (api *SkillAPI) disableSkill(c *gin.Context) {
	id := c.Param("id")
	if err := api.skillManager.Disable(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 200, "msg": "Skill disabled successfully"})
}
