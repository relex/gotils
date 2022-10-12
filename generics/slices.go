package generics

// IterateSlice calls the given func for each of (index, value) pair in the given slice
func IterateSlice[T any](
	list []T, action func(item T),
) {
	for _, item := range list {
		action(item)
	}
}

// IterateSliceIndexed calls the given func for each of (index, value) pair in the given slice
func IterateSliceIndexed[T any](
	list []T,
	action func(index int, item T),
) {
	for index, item := range list {
		action(index, item)
	}
}

// MapSlice transforms the given slice by mapping each item to something else
func MapSlice[T any, R any](
	list []T,
	mapper func(item T) R,
) []R {
	output := make([]R, len(list))
	for index, item := range list {
		output[index] = mapper(item)
	}
	return output
}

// ReduceSlice reduces the given slice into a single result
func ReduceSlice[T any, R any](
	list []T,
	reducer func(item T, accumulated R) R,
	initial R,
) R {
	var lastResult R
	for _, item := range list {
		lastResult = reducer(item, lastResult)
	}
	return lastResult
}

func GroupSlice[T any, K comparable](
	list []T,
	getKey func(item T) K,
) map[K][]T {
	groupMap := make(map[K][]T)
	for _, item := range list {
		key := getKey(item)
		if group, exists := groupMap[key]; exists {
			groupMap[key] = append(group, item)
		} else {
			groupMap[key] = []T{item}
		}
	}
	return groupMap
}
