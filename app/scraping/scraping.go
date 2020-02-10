package scraping

import "github.com/n1try/kithub2/app/config"

const (
	baseUrl      = config.KitVvzBaseUrl
	mainUrl      = baseUrl + "/field.asp"
	facultiesUrl = baseUrl + "/fields.asp?group=Vorlesungsverzeichnis"
	eventUrl     = baseUrl + "/event.asp"
	maxWorkers   = 6
)

type ScrapeJob interface {
	process() (interface{}, error)
}

type Scraper interface {
	Schedule(job ScrapeJob, cronExp string)
	Run(job ScrapeJob) (interface{}, error)
}
