package app

import (
	"flag"
	"github.com/n1try/kitsquid/app/admin"
	"github.com/n1try/kitsquid/app/comments"
	"github.com/n1try/kitsquid/app/config"
	"github.com/n1try/kitsquid/app/events"
	"github.com/n1try/kitsquid/app/reviews"
	"github.com/n1try/kitsquid/app/users"
	"github.com/n1try/kitsquid/app/web"
)

func init() {
	flag.Set("logtostderr", "true")
	flag.Set("stderrthreshold", "WARNING")
	flag.Set("v", "2")
	flag.Parse()

	config.Init()
	web.Init()
	events.Init(config.Db())
	users.Init(config.Db())
	comments.Init(config.Db())
	reviews.Init(config.Db())
	admin.Init(config.Db())
}

func Run() {
	web.Start()
}
