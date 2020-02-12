package common

import "errors"

var tguids = map[SemesterKey]string{
	SemesterWs1819: "0x4CB7204338AE4F67A58AFCE6C29D1488",
}

func ResolveSemesterId(semester SemesterKey) (string, error) {
	if id, ok := tguids[semester]; ok {
		return id, nil
	}
	return "", errors.New("unknown semester key")
}
