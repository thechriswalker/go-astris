package timezone

import (
	"testing"
)

var prefixTests = []struct {
	prefix string
	hits   []string
}{
	{
		prefix: "Africa/A",
		hits:   []string{"Africa/Abidjan", "Africa/Accra", "Africa/Addis_Ababa", "Africa/Algiers", "Africa/Asmara"},
	},
	{
		prefix: "Europe/P",
		hits:   []string{"Europe/Paris", "Europe/Podgorica", "Europe/Prague"},
	},
	{
		prefix: "Pacific/W",
		hits:   []string{"Pacific/Wake", "Pacific/Wallis"},
	},
	{
		prefix: "Europe/NonExistent",
		hits:   []string{},
	},
}

func TestSearch(t *testing.T) {
	for _, pt := range prefixTests {
		actual := PrefixSearch(pt.prefix)
		if len(actual) != len(pt.hits) {
			t.Log("Prefix Search returned wrong number of hits")
			t.Fail()
		} else {
			for i := range actual {
				if actual[i] != pt.hits[i] {
					t.Log("Prefix Search returned bad hits")
					t.Fail()
					break
				}
			}
		}
	}
}
