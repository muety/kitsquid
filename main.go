package main

import (
	"flag"
	"fmt"
	log "github.com/golang/glog"
	"github.com/n1try/kithub2/model"
	"github.com/n1try/kithub2/scraping"
	"golang.org/x/text/language"
)

func init() {
	flag.Set("logtostderr", "true")
	flag.Set("stderrthreshold", "WARNING")
	flag.Set("v", "2")
	flag.Parse()
}

func main() {
	result, err := scraping.FetchLectures(model.SemesterWs1819, language.English)
	if err != nil {
		panic(err)
	}
	fmt.Println(result)

	log.Flush()
}
