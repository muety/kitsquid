package app

import (
	"flag"
	"fmt"
	log "github.com/golang/glog"
	"github.com/n1try/kithub2/app/common"
	"github.com/n1try/kithub2/app/config"
	"github.com/n1try/kithub2/app/events"
	"github.com/n1try/kithub2/app/scraping"
	"github.com/n1try/kithub2/app/users"
	"github.com/n1try/kithub2/app/web"
	uuid "github.com/satori/go.uuid"
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
}

func Run() {
	//web.Start()
	//_debugScrape()
	//_debugScrapeDetails()
	//_debugGet()
	_debugMail()
}

func _debugScrape() {
	scraper := scraping.NewEventScraper()
	job := scraping.FetchEventsJob{Semester: common.SemesterWs1819}
	result, err := scraper.Run(job)
	if err != nil {
		panic(err)
	}
	log.Flush()

	if err := events.InsertMulti(result.([]*events.Event), true); err != nil {
		fmt.Printf("Error: %v\n", err)
	}
}

func _debugScrapeDetails() {
	scraper := scraping.NewEventDetailsScraper()
	ls, err := events.GetAll()
	if err != nil {
		panic(err)
	}
	job := scraping.FetchDetailsJob{Events: ls}
	result, err := scraper.Run(job)
	if err != nil {
		panic(err)
	}
	log.Flush()

	for _, l := range result.([]*events.Event) {
		if err := events.Insert(l, true); err != nil {
			log.Errorf("failed to update lecture %s – %v\n", l.Id, err)
		}
	}
}

func _debugGet() {
	ls, err := events.FindAll(&events.EventQuery{
		LecturerIdEq: "0xE4129354DF0A3A4E8EE42CD65C5BCD1C",
	})
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	}
	fmt.Println(ls)
}

func _debugMail() {
	err := users.SendConfirmationMail(&users.User{
		Id: "ferdinand@muetsch.io",
	}, uuid.NewV4().String())
	if err != nil {
		log.Fatal(err)
	}
}
