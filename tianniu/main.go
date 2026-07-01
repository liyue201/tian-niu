package main

import (
	"os"
	"os/signal"
	"syscall"

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

	db, err := repository.NewRepository("test.db")
	if err != nil {
		log.Errorf("Failed to initialize database: %v", err)
		panic(err)
	}

	a := agent.NewAgent(appConf.LLMProviders.FrontModel, agent.SystemPrompt, []tool.Tool{})
	s := server.NewServer(":8080", db, a)
	s.Run()
	defer s.Stop()

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)
	<-signalChan
}
