package recommend

type Predicate[T any] func(item T) bool

type NamedPredicate[T any] struct {
	Name       string
	Predicates []Predicate[T]
}

// Sorter 排序器
type Sorter interface {
	Sort([]int64) (*NamedPredicate[int64], error)
}
