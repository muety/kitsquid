package config

import (
	"errors"
	"github.com/n1try/kithub2/app/model"
)

const (
	KitVvzBaseUrl = "https://campus.kit.edu/live-stud/campus/all"
)

var tguids = map[model.SemesterKey]string{
	model.SemesterWs1819: "0x4CB7204338AE4F67A58AFCE6C29D1488",
}

var palette = []string{"#9e9e9e", "#009688", "#f44336", "#9c27b0", "#3f51b5", "#03a9f4", "#8bc34a", "#ffeb3b", "#ff9800", "#ff5722", "#607d8b", "#e91e63", "#673ab7", "#2196f3", "#00bcd4", "#4caf50", "#cddc39", "#ffc107", "#e57373", "#7986cb", "#81c784", "#ffb74d", "#ff8a65", "#f06292"}

var colorMap map[string]int

func ResolveSemesterId(semester model.SemesterKey) (string, error) {
	if id, ok := tguids[semester]; ok {
		return id, nil
	}
	return "", errors.New("unknown semester key")
}

func SetColorDomain(keys []string) {
	colorMap = make(map[string]int)
	for i, k := range keys {
		if i < len(palette)-1 {
			colorMap[k] = i + 1
		} else {
			colorMap[k] = 0
		}
	}
}

func ResolveColor(key string) string {
	if idx, ok := colorMap[key]; ok {
		return palette[idx]
	}
	return palette[0]
}
