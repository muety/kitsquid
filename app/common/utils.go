package common

import "errors"

var tguids = map[SemesterKey]string{
	SemesterWs1920: "0x4CB7204338AE4F67A58AFCE6C29D1488",
	SemesterWs1718: "0x29DCBC00ADFA894292650C3990E6F2BF",
}

func ResolveSemesterId(semester SemesterKey) (string, error) {
	if id, ok := tguids[semester]; ok {
		return id, nil
	}
	return "", errors.New("unknown semester key")
}
