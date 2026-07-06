package main

import (
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"github.com/liyue201/tian-niu/pkg/agent"
	"github.com/liyue201/tian-niu/pkg/agent/tool"
	"github.com/liyue201/tian-niu/pkg/repository"
	"github.com/liyue201/tian-niu/pkg/server"
	"github.com/liyue201/tian-niu/pkg/shared"
	"github.com/liyue201/tian-niu/pkg/shared/log"
)

func main() {
	_ = godotenv.Load()

	appConf, err := shared.LoadAppConfig("config.json")
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

	bashConf := appConf.BashTool
	bashToolConfig := tool.BashToolConfig{
		Timeout:        time.Duration(bashConf.TimeoutSeconds) * time.Second,
		MaxOutput:      bashConf.MaxOutputKB * 1024,
		WorkDir:        bashConf.WorkDir,
		Disabled:       bashConf.Disabled,
		AllowDangerous: bashConf.AllowDangerous,
	}
	a := agent.NewAgent(appConf.LLMProviders.FrontModel, agent.SystemPrompt, []tool.Tool{tool.NewBashTool(bashToolConfig)})
	s := server.NewServer(":8080", db, a)
	s.Run()
	defer s.Stop()

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)
	<-signalChan
}
