package store

import "github.com/n1try/kithub2/app/config"

func GetFaculties() ([]string, error) {
	facultyMap := make(map[string]bool)
	lectures, err := GetLectures()
	if err != nil {
		return []string{}, err
	}

	for _, l := range lectures {
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

	return faculties, nil
}

func CountFaculties() int {
	if fl, err := GetFaculties(); err == nil {
		return len(fl)
	}
	return 0
}
