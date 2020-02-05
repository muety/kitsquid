package web

import (
	"github.com/gin-gonic/gin"
	log "github.com/golang/glog"
	"github.com/n1try/kithub2/app/config"
	"net/http"
)

var (
	cfg    *config.Config
	router *gin.Engine
)

func Init() {
	cfg = config.Get()

	router = gin.Default()
	router.LoadHTMLGlob("app/views/*")

	router.Static("/assets", "app/public/assets")

	router.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "index.tpl", gin.H{
			"title": "Hello Gin",
		})
	})
}

func Start() {
	if err := router.Run(cfg.ListenAddr()); err != nil {
		log.Fatalf("error listening on %s – %v\n", cfg.ListenAddr(), err)
	}
	log.Infof("Listening in %s\n", cfg.ListenAddr())
}
