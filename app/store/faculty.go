package store

import "github.com/n1try/kithub2/app/config"

func GetFaculties() ([]string, error) {
	cacheKey := "get:all"
	if fl, ok := facultiesCache.Get(cacheKey); ok {
		return fl.([]string), nil
	}

	facultyMap := make(map[string]bool)
	events, err := GetEvents()
	if err != nil {
		return []string{}, err
	}

	for _, l := range events {
		if len(l.Categories) > config.FacultyIdx {
			if _, ok := facultyMap[l.Categories[config.FacultyIdx]]; !ok {
				facultyMap[l.Categories[config.FacultyIdx]] = true
			}
		}
	}

	var i int
	faculties := make([]string, len(facultyMap))
	for k := range facultyMap {
		faculties[i] = k
		i++
	}

	facultiesCache.SetDefault("get:all", faculties)
	return faculties, nil
}

func CountFaculties() int {
	if fl, err := GetFaculties(); err == nil {
		return len(fl)
	}
	return 0
}
