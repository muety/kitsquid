package web

import (
	"context"
	"fmt"
	"github.com/foolin/goview"
	"github.com/foolin/goview/supports/ginview"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	log "github.com/golang/glog"
	"github.com/n1try/kitsquid/app/config"
	"github.com/n1try/limiter/v3"
	mgin "github.com/n1try/limiter/v3/drivers/middleware/gin"
	rls "github.com/n1try/limiter/v3/drivers/store/memory"
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

	// Configure CORS middleware
	corsCfg := cors.DefaultConfig()
	corsCfg.AllowOrigins = []string{cfg.Url}
	if cfg.IsDev() {
		corsCfg.AllowOrigins = append(corsCfg.AllowOrigins, fmt.Sprintf("http://localhost:%d", cfg.Port))
	}
	corsMiddleware := cors.New(corsCfg)

	// Configure rate limiting middleware
	rate, _ := limiter.NewRateFromFormatted(cfg.Rate)
	rateLimitMiddleware := mgin.NewMiddleware(
		limiter.New(rls.NewStore(),
			rate,
			limiter.WithTrustForwardHeader(true)),
	)

	router.Use(
		gin.Recovery(),
		corsMiddleware,
		rateLimitMiddleware,
		RemoteIp(),
	)

	ginviewConfig := goview.DefaultConfig
	ginviewConfig.Root = "app/views"
	ginviewConfig.DisableCache = cfg.Env == "development"
	ginviewConfig.Extension = ".tpl.html"
	ginviewConfig.Funcs = GetFuncMap()

	router.HTMLRender = ginview.New(ginviewConfig)

	// Routes
	RegisterStaticRoutes(router)
	RegisterFallbackRoutes(router)
	RegisterMainRoutes(router)
	RegisterApiRoutes(router)
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
	if cfg.IsDev() || !cfg.Tls.Enable {
		return func() error {
			return srv.ListenAndServe()
		}
	}
	return func() error {
		return srv.ListenAndServeTLS(cfg.Tls.CertPath, cfg.Tls.KeyPath)
	}
}
