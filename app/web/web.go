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
	"github.com/n1try/kithub2/app/web/errors"
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

	router.Use(gin.Recovery())
	router.Use(ErrorHandler())

	ginviewConfig := goview.DefaultConfig
	ginviewConfig.Root = "app/views"
	ginviewConfig.DisableCache = cfg.Env == "development"
	ginviewConfig.Extension = ".tpl.html"
	ginviewConfig.Funcs = template.FuncMap{
		"strIndex":    util.StrIndex,
		"strRemove":   util.StrRemove,
		"randomColor": util.RandomColor,
		"htmlSafe":    util.HtmlSafe,
	}

	router.HTMLRender = ginview.New(ginviewConfig)
}

func routes() {
	router.Static("/assets", "app/public/build")

	router.NoMethod(func(c *gin.Context) {
		c.AbortWithError(http.StatusMethodNotAllowed, errors.NotFound{}).SetType(gin.ErrorTypePublic)
	})

	router.NoRoute(func(c *gin.Context) {
		c.AbortWithError(http.StatusNotFound, errors.NotFound{}).SetType(gin.ErrorTypePublic)
	})

	router.GET("/", AssetsPusher(), func(c *gin.Context) {
		lectures, err := store.FindLectures(nil)
		if err != nil {
			c.Error(err)
			c.AbortWithError(http.StatusInternalServerError, errors.Internal{}).SetType(gin.ErrorTypePublic)
			return
		}

		c.HTML(http.StatusOK, "index", gin.H{
			"lectures":   lectures,
			"active":     "index",
			"facultyIdx": config.FacultyIdx,
		})
	})

	router.GET("/event/:id", AssetsPusher(), func(c *gin.Context) {
		lecture, err := store.GetLecture(c.Param("id"))
		if err != nil {
			c.Error(err).SetType(gin.ErrorTypePrivate)
			c.AbortWithError(http.StatusNotFound, errors.NotFound{}).SetType(gin.ErrorTypePublic)
			return
		}

		c.HTML(http.StatusOK, "event", gin.H{
			"lecture":    lecture,
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

	log.Infof("listening on %s\n", cfg.ListenAddr())
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
