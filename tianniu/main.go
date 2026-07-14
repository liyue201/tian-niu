package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/jinzhu/configor"
	"github.com/joho/godotenv"
	log "github.com/sirupsen/logrus"
	"github.com/tianniu-ai/tianniu/pkg/agent"
	context2 "github.com/tianniu-ai/tianniu/pkg/agent/context"
	"github.com/tianniu-ai/tianniu/pkg/agent/mcp"
	"github.com/tianniu-ai/tianniu/pkg/agent/memory"
	"github.com/tianniu-ai/tianniu/pkg/agent/rag"
	skill2 "github.com/tianniu-ai/tianniu/pkg/agent/skill"
	"github.com/tianniu-ai/tianniu/pkg/agent/tool"
	"github.com/tianniu-ai/tianniu/pkg/repository"
	"github.com/tianniu-ai/tianniu/pkg/server"
	"github.com/tianniu-ai/tianniu/pkg/shared"
	_ "github.com/tianniu-ai/tianniu/pkg/shared/log"
)

type AppConfig struct {
	ServerAddress string `yaml:"server_address"`
	Database      struct {
		Type string `yaml:"type"`
		DSN  string `yaml:"dsn"`
	} `yaml:"database"`
	LLMProviders struct {
		FrontModel shared.ModelConfig `yaml:"front_model"`
		BackModel  shared.ModelConfig `yaml:"back_model"`
	} `yaml:"llm_providers"`
	BashTool       tool.BashToolConfig         `yaml:"bash_tool"`
	LongTermMemory shared.LongTermMemoryConfig `yaml:"long_term_memory"`
}

func loadConfig() (AppConfig, error) {
	if err := godotenv.Load(); err != nil {
		log.Debug("No .env file found, using environment variables from system")
	}

	var config AppConfig
	if err := configor.Load(&config, "config.yaml"); err != nil {
		return AppConfig{}, fmt.Errorf("failed to load config.yaml: %w", err)
	}

	if err := validateConfig(&config); err != nil {
		return AppConfig{}, fmt.Errorf("invalid config: %w", err)
	}

	return config, nil
}

func validateConfig(config *AppConfig) error {
	if config.ServerAddress == "" {
		return errors.New("server_address is required")
	}
	if config.LLMProviders.FrontModel.ApiKey == "" && os.Getenv("FRONT_MODEL_API_KEY") == "" {
		log.Warn("Front model API key not set")
	}
	if config.LLMProviders.BackModel.ApiKey == "" && os.Getenv("BACK_MODEL_API_KEY") == "" {
		log.Warn("Back model API key not set")
	}
	return nil
}

func initDatabase(config AppConfig) (*repository.SQLStore, error) {
	dbType := getEnvOrDefault("DB_TYPE", config.Database.Type, "sqlite")
	dbDSN := getEnvOrDefault("DB_DSN", config.Database.DSN, "test.db")

	log.Infof("Initializing database: type=%s", dbType)
	db, err := repository.NewSQLStore(repository.DBConfig{
		Type: dbType,
		DSN:  dbDSN,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}
	return db, nil
}

func initMCP(db *repository.SQLStore) ([]*mcp.Client, error) {
	mcpStore := mcp.NewSQLMcpStore(db)
	mcpManager := mcp.NewManager(mcpStore)

	if err := mcpManager.LoadSystemMcpServers("mcp-server.json"); err != nil {
		log.Warnf("Failed to load system MCP servers: %v", err)
		return nil, nil
	}

	var clients []*mcp.Client
	systemMcps, _ := mcpManager.GetSystemMcpServers()
	for _, mcpServer := range systemMcps {
		if mcpServer.Status != mcp.McpStatusEnabled {
			continue
		}

		client := mcp.NewMcpToolProvider(mcpServer.Name, mcpServer.Config)
		if err := client.RefreshTools(context.Background()); err != nil {
			log.Warnf("Failed to refresh tools for MCP server %s: %v", mcpServer.Name, err)
			continue
		}
		clients = append(clients, client)
	}

	log.Infof("Loaded %d MCP clients", len(clients))
	return clients, nil
}

func initContextPolicies(backModel shared.ModelConfig, db *repository.SQLStore) []context2.Policy {
	summarizer := context2.NewLLMSummarizer(backModel, 200)
	return []context2.Policy{
		context2.NewOffloadPolicy(db, 0.4, 0, 100),
		context2.NewSummaryPolicy(summarizer, 10, 20, 0.6),
		context2.NewTruncatePolicy(0, 0.85),
	}
}

func initMemory(db *repository.SQLStore, backModel shared.ModelConfig, longTermMemory memory.LongTermMemoryProvider) *memory.MultiLevelMemory {
	memoryUpdater := memory.NewSmartMemoryUpdater(backModel)
	return memory.NewMultiLevelMemory(db, memoryUpdater, longTermMemory)
}

func initSkillManager(db *repository.SQLStore) (*skill2.Manager, error) {
	skillsDir := getEnvOrDefault("SKILLS_DIR", "", "skills")

	skillStore := skill2.NewSQLSkillStore(db)
	skillManager := skill2.NewManager(skillStore, skillsDir)

	if err := skillManager.LoadInstalledSkills(); err != nil {
		return nil, fmt.Errorf("failed to load installed skills: %w", err)
	}

	return skillManager, nil
}

func initLongTermMemory(config shared.LongTermMemoryConfig, backModel shared.ModelConfig) *memory.LongTermMemoryManager {
	if !config.Enabled {
		log.Info("Long-term memory is disabled")
		return nil
	}

	log.Info("Initializing long-term memory system...")

	vectorDBConfig := config.VectorDB
	vectorStore, err := rag.NewPGVectorStore(rag.Config{
		Host:      vectorDBConfig.Host,
		Port:      vectorDBConfig.Port,
		User:      vectorDBConfig.User,
		Password:  vectorDBConfig.Password,
		Database:  vectorDBConfig.Database,
		Dimension: vectorDBConfig.Dimension,
	})
	if err != nil {
		log.Warnf("Failed to initialize vector store: %v. Long-term memory will be disabled.", err)
		return nil
	}

	embeddingConfig := config.EmbeddingService
	embeddingService := rag.NewHTTPEmbeddingService(rag.HTTPEmbeddingConfig{
		APIKey:     embeddingConfig.APIKey,
		BaseURL:    embeddingConfig.BaseURL,
		Model:      embeddingConfig.Model,
		Dimensions: embeddingConfig.Dimensions,
	})

	rerankConfig := config.RerankService
	rerankService := rag.NewHTTPRerankService(rag.HTTPRerankConfig{
		APIKey:  rerankConfig.APIKey,
		BaseURL: rerankConfig.BaseURL,
		Model:   rerankConfig.Model,
	})

	strategyConfig := config.Strategy
	manager := memory.NewLongTermMemoryManager(
		vectorStore,
		embeddingService,
		rerankService,
		backModel,
		memory.StrategyConfig{
			QuickSaveRounds:          strategyConfig.QuickSaveRounds,
			RegularSaveRounds:        strategyConfig.RegularSaveRounds,
			ForceSaveRounds:          strategyConfig.ForceSaveRounds,
			MinTokenThreshold:        strategyConfig.MinTokenThreshold,
			TopicSimilarityThreshold: strategyConfig.TopicSimilarityThreshold,
		},
	)

	log.Info("Long-term memory system initialized successfully")
	return manager
}

func getEnvOrDefault(envKey, configValue, defaultValue string) string {
	if envValue := os.Getenv(envKey); envValue != "" {
		return envValue
	}
	if configValue != "" {
		return configValue
	}
	return defaultValue
}

func waitForShutdown() {
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)
	<-signalChan
}

func main() {

	log.Info("Starting TianNiu AI Assistant...")

	config, err := loadConfig()
	if err != nil {
		log.Fatalf("Config error: %v", err)
	}

	db, err := initDatabase(config)
	if err != nil {
		log.Fatalf("Database initialization failed: %v", err)
	}

	mcpClients, err := initMCP(db)
	if err != nil {
		log.Fatalf("MCP initialization failed: %v", err)
	}

	policies := initContextPolicies(config.LLMProviders.BackModel, db)

	skillManager, err := initSkillManager(db)
	if err != nil {
		log.Fatalf("Skill manager initialization failed: %v", err)
	}

	longTermMemoryManager := initLongTermMemory(config.LongTermMemory, config.LLMProviders.BackModel)

	multiLevelMemory := initMemory(db, config.LLMProviders.BackModel, longTermMemoryManager)

	mgr := agent.NewManager(
		db,
		config.LLMProviders.FrontModel,
		agent.SystemPrompt,
		[]tool.Tool{tool.NewBashTool(config.BashTool)},
		mcpClients,
		policies,
		multiLevelMemory,
		skillManager,
	)

	skillAPI := server.NewSkillAPI(skillManager)
	mcpAPI := server.NewMcpAPI(mcp.NewManager(mcp.NewSQLMcpStore(db)))

	s := server.NewServer(config.ServerAddress, db, mgr, skillAPI, mcpAPI)

	s.Run()
	defer s.Stop()

	log.Infof("Server started on %s", config.ServerAddress)

	waitForShutdown()

	log.Info("Shutting down gracefully...")
}
