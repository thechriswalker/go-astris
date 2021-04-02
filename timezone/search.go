package timezone

//go:generate go run generate_zonelist.go

import (
	"sort"
	"strings"
)

func PrefixSearch(prefix string) []string {
	// we do a binary search to find the lowest index where
	// the strings compare >= to our prefix
	lo := sort.Search(len(List), func(i int) bool {
		return strings.Compare(List[i], prefix) != -1
	})
	// there is NO not found, just "where it would be inserted"
	// which could be the beginning or the end or the middle.
	// but by finding the place after that where we stop matching
	// we find the slice. Both values will be the same if there is no match
	// so we would end up with an empty slice (no hits)
	hi := sort.Search(len(List[lo:]), func(i int) bool {
		return !strings.HasPrefix(List[lo+i], prefix)
	})
	// this will contain all valid entries
	return List[lo : lo+hi]
}

func IsValidZone(zone string) bool {
	list := PrefixSearch(zone) // should be a single entry, which matches the query
	if len(list) != 1 {
		return false
	}
	return zone == list[0]
}
