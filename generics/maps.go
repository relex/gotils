package generics

import (
	"golang.org/x/exp/constraints"
	"golang.org/x/exp/maps"
	"golang.org/x/exp/slices"
)

// MapToSlice transforms a map to a slice.
//
// The result is ordered by keys.
func MapToSlice[K constraints.Ordered, V any, R any](
	source map[K]V,
	mapPair func(key K, value V) R,
) []R {
	sortedKeys := maps.Keys(source)
	slices.Sort(sortedKeys)

	result := make([]R, 0, len(source))
	for _, key := range sortedKeys {
		result = append(result, mapPair(key, source[key]))
	}
	return result
}

// MapToSlice transforms a map to a slice.
//
// The result is ordered by keys with the "less" function.
func MapToSliceWithSortFunc[K comparable, V any, R any](
	source map[K]V,
	mapPair func(key K, value V) R,
	less func(k1, k2 K) bool,
) []R {
	sortedKeys := maps.Keys(source)
	slices.SortStableFunc(sortedKeys, less)

	result := make([]R, 0, len(source))
	for _, key := range sortedKeys {
		result = append(result, mapPair(key, source[key]))
	}
	return result
}
