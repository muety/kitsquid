package scraping

import "github.com/muety/kitsquid/app/config"

const (
	baseUrl       = config.KitVvzBaseURL
	mainUrl       = baseUrl + "/field.asp"
	facultiesUrl  = baseUrl + "/fields.asp?group=Vorlesungsverzeichnis"
	eventUrl      = baseUrl + "/event.asp"
	eventIliasUrl = "https://ilias.studium.kit.edu/Customizing/global/CourseDataWS.php/gguid/%s"
	maxWorkers    = 6
)

type ScrapeJob interface {
	process() (interface{}, error)
}

type Scraper interface {
	Schedule(job ScrapeJob, cronExp string)
	Run(job ScrapeJob) (interface{}, error)
}
