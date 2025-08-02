package recommend

type FilterRule interface {
	Filter(userId int64, anchorIdList []int64) (result []int64, err error)
}
