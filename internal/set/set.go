package set

type Set[T comparable] struct {
	m map[T]struct{}
}

func New[T comparable](values ...T) Set[T] {
	m := make(map[T]struct{})
	for _, v := range values {
		m[v] = struct{}{}
	}
	return Set[T]{m}
}

func (s *Set[T]) Add(values ...T) {
	for _, v := range values {
		s.m[v] = struct{}{}
	}
}

func (s *Set[T]) Slice() []T {
	values := make([]T, 0, len(s.m))
	for v := range s.m {
		values = append(values, v)
	}
	return values
}

func (s *Set[T]) Exists(v T) bool {
	_, ok := s.m[v]
	return ok
}

func (s *Set[T]) Len() int {
	return len(s.m)
}

func (s *Set[T]) Pop(v T) bool {
	if s.Exists(v) {
		delete(s.m, v)
		return true
	}
	return false
}

func (s *Set[T]) Clear() {
	for v := range s.m {
		delete(s.m, v)
	}
}
