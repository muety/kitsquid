package app

import (
	"flag"
	"github.com/muety/kitsquid/app/admin"
	"github.com/muety/kitsquid/app/config"
	"github.com/muety/kitsquid/app/events"
	"github.com/muety/kitsquid/app/migrations"
	"github.com/muety/kitsquid/app/users"
	"github.com/muety/kitsquid/app/web"
)

func init() {
	flag.Set("logtostderr", "true")
	flag.Set("stderrthreshold", "WARNING")
	flag.Set("v", "2")
	flag.Parse()

	config.Init()
	web.Init()
	events.Init(config.Db(), config.EventBus())
	users.Init(config.Db(), config.EventBus())
	admin.Init(config.Db(), config.EventBus())
}

/*
Run runs everything!
*/
func Run() {
	if !config.Get().QuickStart {
		migrations.RunAll()
	}
	web.Start()
}
