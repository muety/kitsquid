package web

import (
	"github.com/foolin/goview"
	"github.com/foolin/goview/supports/ginview"
	"github.com/gin-gonic/gin"
	log "github.com/golang/glog"
	"github.com/n1try/kithub2/app/config"
	"github.com/n1try/kithub2/app/store"
	"net/http"
)

var (
	cfg    *config.Config
	router *gin.Engine
)

func Init() {
	configure()
	routes()
}

func configure() {
	cfg = config.Get()
	router = gin.Default()

	ginviewConfig := goview.DefaultConfig
	ginviewConfig.Root = "app/views"
	ginviewConfig.DisableCache = cfg.Env == "development"
	ginviewConfig.Extension = ".tpl.html"

	router.HTMLRender = ginview.New(ginviewConfig)
}

func routes() {
	router.Static("/assets", "app/public/build")

	router.GET("/", func(c *gin.Context) {
		pushAssets(c)

		lectures, err := store.FindLectures(nil)
		if err != nil {
			c.AbortWithError(http.StatusInternalServerError, err)
		}

		c.HTML(http.StatusOK, "index", gin.H{
			"lectures": lectures,
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
