package server

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	_ "github.com/gin-gonic/gin/binding"
	"github.com/liyue201/tian-niu/pkg/agent"
	"github.com/liyue201/tian-niu/pkg/repository"
	"github.com/liyue201/tian-niu/pkg/service"
	"github.com/liyue201/tian-niu/pkg/shared/log"
)

type Server struct {
	svc        *service.Service
	httpServer *http.Server
	wg         sync.WaitGroup
}

func NewServer(addr string, db *repository.Repository, agent *agent.Agent) *Server {
	scv := service.NewService(db, agent)
	engine := gin.New()
	gin.SetMode(gin.ReleaseMode)
	engine.Use(gin.Recovery(), gin.Logger())

	s := &Server{
		svc:        scv,
		httpServer: &http.Server{Addr: addr, Handler: engine},
	}
	s.setupRouter(engine)
	return s
}

func (s *Server) setupRouter(g *gin.Engine) {

	api := g.Group("/api")
	api.POST("/user/register", s.register)
	api.POST("/user/login", s.login)
	api.POST("/conversation", s.createConversation)
	api.GET("/conversation", s.listConversations)
	api.PATCH("/conversation/:conversation_id", s.renameConversation)
	api.DELETE("/conversation/:conversation_id", s.deleteConversation)
	api.POST("/conversation/:conversation_id/message", s.createMessage)
	api.GET("/conversation/:conversation_id/message", s.listMessages)
}

func (s *Server) Run() {
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		err := s.httpServer.ListenAndServe()
		if err != nil {
			log.Infof("%v", err.Error())
		}
	}()
}

func (s *Server) Stop() {
	s.httpServer.Shutdown(context.Background())
	s.wg.Wait()
}

func (s *Server) setCors(r gin.IRouter) {
	corsCfg := cors.Config{
		AllowAllOrigins: true,
		AllowMethods:    []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"},
		AllowHeaders: []string{"Origin", "Access-Control-Allow-Origin", "Accept",
			"Content-Type", "Content-Length", "Accept-Encoding", "X-CSRF-Token", "Authorization", "X-Requested-With"},
		ExposeHeaders:    []string{"Content-Length", "Access-Control-Allow-Origin"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}
	r.Use(cors.New(corsCfg))
}
