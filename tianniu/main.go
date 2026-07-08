package main

import (
	"context"
	"encoding/json"
	"os"
	"os/signal"
	"syscall"

	"github.com/joho/godotenv"
	"github.com/liyue201/tian-niu/pkg/agent"
	context2 "github.com/liyue201/tian-niu/pkg/agent/context"
	"github.com/liyue201/tian-niu/pkg/agent/mcp"
	"github.com/liyue201/tian-niu/pkg/agent/memory"
	"github.com/liyue201/tian-niu/pkg/agent/tool"
	"github.com/liyue201/tian-niu/pkg/repository"
	"github.com/liyue201/tian-niu/pkg/server"
	"github.com/liyue201/tian-niu/pkg/shared"
	_ "github.com/liyue201/tian-niu/pkg/shared/log"
	"github.com/liyue201/tian-niu/pkg/storage/leveldb_storage"
	log "github.com/sirupsen/logrus"
)

type AppConfig struct {
	LLMProviders struct {
		FrontModel shared.ModelConfig `json:"front_model"`
		BackModel  shared.ModelConfig `json:"back_model"`
	} `json:"llm_providers"`
	BashTool tool.BashToolConfig `json:"bash_tool"`
}

func loadAppConfig(path string) (AppConfig, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return AppConfig{}, err
	}
	var config AppConfig
	err = json.Unmarshal(content, &config)
	if err != nil {
		return AppConfig{}, err
	}
	return config, nil
}

func main() {
	_ = godotenv.Load()

	appConf, err := loadAppConfig("config.json")
	if err != nil {
		log.Errorf("Failed to load config.json: %v", err)
		panic(err)
	}

	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "test.db"
	}
	db, err := repository.NewRepository(dbPath)
	if err != nil {
		log.Errorf("Failed to initialize database: %v", err)
		panic(err)
	}

	mcpServerMap, err := mcp.LoadMcpServerConfig("mcp-server.json")
	if err != nil {
		log.Errorf("Failed to load MCP server configuration: %v", err)
	}
	ctx := context.Background()
	mcpClients := make([]*mcp.Client, 0)
	for k, v := range mcpServerMap {
		mcpClient := mcp.NewMcpToolProvider(k, v)
		if err := mcpClient.RefreshTools(ctx); err != nil {
			log.Errorf("Failed to refresh tools for MCP server %s: %v", k, err)
			continue
		}
		mcpClients = append(mcpClients, mcpClient)
	}

	// Create context engine and policies
	leveldbPath := os.Getenv("LEVELEDB_PATH")
	if leveldbPath == "" {
		leveldbPath = "leveldb_data"
	}
	storage, err := leveldb_storage.NewLevelDBStorage(leveldbPath)
	if err != nil {
		log.Errorf("Failed to create storage: %v", err)
		panic(err)
	}
	defer storage.Close()

	summarizer := context2.NewLLMSummarizer(appConf.LLMProviders.BackModel, 200)

	policies := []context2.Policy{
		context2.NewOffloadPolicy(storage, 0.4, 0, 100),
		context2.NewSummaryPolicy(summarizer, 10, 20, 0.6),
		context2.NewTruncatePolicy(0, 0.85),
	}

	memoryUpdater := memory.NewLLMMemoryUpdater(appConf.LLMProviders.BackModel)
	multiLevelMemory := memory.NewMultiLevelMemory(storage, memoryUpdater)

	mgr := agent.NewManager(
		db,
		appConf.LLMProviders.FrontModel,
		agent.SystemPrompt,
		[]tool.Tool{tool.NewBashTool(appConf.BashTool)},
		mcpClients,
		policies,
		multiLevelMemory)

	s := server.NewServer(":8080", db, mgr)
	s.Run()
	defer s.Stop()

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)
	<-signalChan
}
