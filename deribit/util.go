package deribit

// chunk a slice into a slice of slices, where each sub-slice is of max length size.
// Paanics if size is 0.
func chunk[T any](items []T, size uint) [][]T {
	n := len(items) / int(size)
	if len(items)%int(size) != 0 {
		n += 1
	}
	chunks := make([][]T, n)
	for i := 0; i < n; i++ {
		start := i * int(size)
		if i == n-1 {
			chunks[i] = items[start:]
		} else {
			end := (i + 1) * int(size)
			chunks[i] = items[start:end]
		}
	}
	return chunks
}
