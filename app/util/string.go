package util

/*
ContainsString checks whether a list of strings contains a certain search string
*/
func ContainsString(needle string, haystack []string) bool {
	for _, item := range haystack {
		if item == needle {
			return true
		}
	}
	return false
}
