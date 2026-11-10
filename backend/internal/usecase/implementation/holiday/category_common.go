package holiday

import "regexp"

var hexColorRegex = regexp.MustCompile(`^#[0-9A-Fa-f]{6}$`)

func isValidHexColor(c string) bool {
	return hexColorRegex.MatchString(c)
}
