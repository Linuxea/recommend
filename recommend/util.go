package recommend

import "math/rand/v2"

// 分组函数，按条件将元素分组（每个元素只属于第一个匹配的组，避免重复）
func GroupsExclusive[T any](slice []T, namedPredicate NamedPredicate[T]) [][]T {
	var results [][]T
	used := make(map[int]bool) // 记录已经被分组的元素索引

	filters := namedPredicate.Predicates
	for _, filter := range filters {
		var result []T
		for i, elem := range slice {
			if !used[i] && filter(elem) {
				result = append(result, elem)
				used[i] = true
			}
		}
		results = append(results, result)
	}

	return results
}

// 层级分组函数，支持多层分组
func GroupsHierarchical[T any](slice []T, filterGroups []NamedPredicate[T]) [][]T {
	if len(filterGroups) == 0 {
		return [][]T{slice}
	}

	// 第一层分组
	firstGroups := GroupsExclusive(slice, filterGroups[0])

	// 如果只有一层，直接返回
	if len(filterGroups) == 1 {
		return firstGroups
	}

	// 递归处理后续层级
	var finalResults [][]T
	for _, group := range firstGroups {
		subResults := GroupsHierarchical(group, filterGroups[1:])
		for _, subResult := range subResults {
			if len(subResult) == 0 {
				continue // 跳过空的子结果
			}
			finalResults = append(finalResults, subResult)
		}
	}

	return finalResults
}

// 泛型随机排序函数
func Shuffle[T any](slice []T) {
	// no need anymore
	// Go 1.20 开始，全局随机数生成器会自动使用一个真正随机的种子进行初始化，不再需要手动设置种子来获得不可预测的随机数
	// rand.Seed(time.Now().UnixNano())
	rand.Shuffle(len(slice), func(i, j int) {
		slice[i], slice[j] = slice[j], slice[i]
	})
}

func Flat[T any](slices [][]T) []T {
	var result []T
	for _, slice := range slices {
		result = append(result, slice...)
	}
	return result
}
