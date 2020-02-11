package web

import (
	"context"
	"github.com/foolin/goview"
	"github.com/foolin/goview/supports/ginview"
	"github.com/gin-gonic/gin"
	log "github.com/golang/glog"
	"github.com/n1try/kithub2/app/config"
	"github.com/n1try/kithub2/app/web/util"
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
	cfg = config.Get()
	router = gin.Default()

	router.Use(gin.Recovery())
	router.Use(ErrorHandler())

	ginviewConfig := goview.DefaultConfig
	ginviewConfig.Root = "app/views"
	ginviewConfig.DisableCache = cfg.Env == "development"
	ginviewConfig.Extension = ".tpl.html"
	ginviewConfig.Funcs = util.GetFuncMap()

	router.HTMLRender = ginview.New(ginviewConfig)

	// Routes
	RegisterStaticRoutes(router)
	RegisterFallbackRoutes(router)
	RegisterMainRoutes(router)
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

	log.Infof("listening on %s\n", cfg.ListenAddr())
	if err := getServeFunc(srv)(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("failed to start server on %s – %v\n", cfg.ListenAddr(), err)
	}

	<-exited
}

func getServeFunc(srv *http.Server) func() error {
	if cfg.IsDev() {
		return func() error {
			return srv.ListenAndServe()
		}
	}
	return func() error {
		return srv.ListenAndServeTLS(cfg.Tls.CertPath, cfg.Tls.KeyPath)
	}
}
