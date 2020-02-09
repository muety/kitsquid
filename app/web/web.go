package web

import (
	"context"
	"github.com/foolin/goview"
	"github.com/foolin/goview/supports/ginview"
	"github.com/gin-gonic/gin"
	log "github.com/golang/glog"
	"github.com/n1try/kithub2/app/config"
	"github.com/n1try/kithub2/app/store"
	"github.com/n1try/kithub2/app/util"
	"html/template"
	"net/http"
	"os"
	"os/signal"
	"syscall"
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
	ginviewConfig.Funcs = template.FuncMap{
		"strIndex":    util.StrIndex,
		"randomColor": util.RandomColor,
	}

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
			"lectures":   lectures,
			"active":     "index",
			"facultyIdx": config.FacultyIdx,
		})
	})
}

func Start() {
	cfg := config.Get()

	srv := &http.Server{
		Addr:    cfg.ListenAddr(),
		Handler: router,
	}

	exited := make(chan struct{})

	go func() {
		sigint := make(chan os.Signal, 1)
		signal.Notify(sigint, syscall.SIGINT, syscall.SIGTERM)
		<-sigint

		if err := srv.Shutdown(context.Background()); err != nil {
			log.Fatalf("failed to shut down the server gracefully – %v", err)
		}

		log.Infoln("exited gracefully")
		close(exited)
	}()

	log.Infof("Listening on %s\n", cfg.ListenAddr())
	if err := getServeFunc(srv)(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("failed to start server on %s – %v\n", cfg.ListenAddr(), err)
	}

	<-exited
}

func getServeFunc(srv *http.Server) func() error {
	if cfg.Env == "development" {
		return func() error {
			return srv.ListenAndServe()
		}
	}
	return func() error {
		return srv.ListenAndServeTLS(cfg.Tls.CertPath, cfg.Tls.KeyPath)
	}
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
