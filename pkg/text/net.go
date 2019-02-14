package text

import (
	"regexp"
	"strings"

	"github.com/mozillazg/go-unidecode"
)

var reserved_re = regexp.MustCompile("(\\s|\\.|@|:|\\/|\\?|#|\\[|\\]|!|\\$|&|\\(|\\)|\\*|\\+|,|;|=|\\|%|<|>|\\||\\^|~|\"|\\{|\\}|`|–|—)")

// Convert to ASCII, replace all URI reserved chars to "-", Convert to lowercase
func Slugify(value string) string {
	value = reserved_re.ReplaceAllString(unidecode.Unidecode(value), "-")
	value = strings.Replace(value, "'", "", -1)
	return strings.ToLower(value)
}
