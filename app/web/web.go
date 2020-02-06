package web

import (
	"net/http"

	"github.com/gin-gonic/gin"
	log "github.com/golang/glog"
	"github.com/n1try/kithub2/app/config"
)

var (
	cfg    *config.Config
	router *gin.Engine
)

func Init() {
	cfg = config.Get()

	router = gin.Default()
	router.LoadHTMLGlob("app/views/*")

	router.Static("/assets", "app/public/build")

	router.GET("/", func(c *gin.Context) {
		pushAssets(c)

		c.HTML(http.StatusOK, "index.tpl.html", gin.H{
			"title": "Hello Gin",
		})
	})
}

func Start() {
	cfg := config.Get()

	if cfg.Env == "production" {
		if err := router.RunTLS(cfg.ListenAddr(), cfg.Tls.CertPath, cfg.Tls.KeyPath); err != nil {
			log.Fatalf("error listening (https) on %s – %v\n", cfg.ListenAddr(), err)
		}
	} else {
		if err := router.Run(cfg.ListenAddr()); err != nil {
			log.Fatalf("error listening (http) on %s – %v\n", cfg.ListenAddr(), err)
		}
	}

	log.Infof("Listening on %s\n", cfg.ListenAddr())
}

func pushAssets(c *gin.Context) {
	if pusher := c.Writer.Pusher(); pusher != nil {
		for _, a := range config.PushAssets {
			if err := pusher.Push(a, nil); err != nil {
				log.Errorf("failed to push %s – %v", a, err)
			}
		}
	}
}
