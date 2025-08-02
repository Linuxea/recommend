package recommend

// Recaller 召回机制
type Recaller interface {
	Recall() ([]int64, error)
}
