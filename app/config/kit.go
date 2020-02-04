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

func ResolveSemesterId(semester model.SemesterKey) (string, error) {
	if id, ok := tguids[semester]; ok {
		return id, nil
	}
	return "", errors.New("unknown semester key")
}
