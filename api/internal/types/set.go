package types

type Set struct {
	items map[interface{}]bool
}

func NewSet() Set {
	return Set{map[interface{}]bool{}}
}

func (s Set) Add(x interface{}) {
	s.items[x] = true
}

func (s Set) AddAll(other Set) {
	for k := range other.items {
		s.Add(k)
	}
}

func (s Set) Exists(x interface{}) bool {
	if _, ok := s.items[x]; ok {
		return true
	}

	return false
}

func (s Set) Remove(x interface{}) bool {
	if s.Exists(x) {
		delete(s.items, x)
		return true
	}

	return false
}

func (s Set) Size() int {
	return len(s.items)
}

func (s Set) ToMap() map[interface{}]bool {
	return s.items
}
