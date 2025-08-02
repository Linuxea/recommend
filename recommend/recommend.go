package recommend

import (
	"context"
	"time"

	"github.com/go-redis/redis/v8"
)

// RecommenderBuilder 接口定义了推荐构建器的基本操作
type RecommendBuilder interface {
	BuildRecommend() Recommender
}

func NewRecommend(userid int64, recaller Recaller, filterRules []FilterRule, sorters, postSorters []Sorter) Recommender {
	return &Recommend{
		UserId:    userid,
		Recaller:  recaller,
		Rules:     filterRules,
		Sorts:     sorters,
		PostSorts: postSorters,
		rClient:   nil,
	}
}

// Recommender 接口定义了推荐的基本操作
type Recommender interface {
	Filter(anchors []int64) ([]int64, error)
	GroupSort(anchors []int64) ([][]int64, error)
	PostSort(anchors []int64) ([][]int64, error)
	Fetch(size int) ([]int64, error)
}

// Recommend 推荐框架
type Recommend struct {
	UserId int64 // 用户ID
	Recaller
	Rules     []FilterRule  // 所有过滤规则
	Sorts     []Sorter      // 所有排序规则
	PostSorts []Sorter      // 后处理排序规则
	rClient   *redis.Client // Redis客户端

}

func (m *Recommend) Filter(anchors []int64) ([]int64, error) {
	var err error
	result := anchors
	for _, rule := range m.Rules {
		result, err = rule.Filter(m.UserId, result)
		if err != nil {
			return nil, err
		}
		if len(result) == 0 { // 如果过滤后没有主播，直接返回
			return nil, nil
		}
	}
	return result, nil
}

func (m *Recommend) GroupSort(anchors []int64) ([][]int64, error) {

	var predicates []*NamedPredicate[int64]
	for _, sorter := range m.Sorts {
		predicate, err := sorter.Sort(anchors)
		if err != nil {
			return nil, err
		}
		predicates = append(predicates, predicate)
	}

	//convert
	var results []NamedPredicate[int64]
	for _, predicate := range predicates {
		results = append(results, *predicate)
	}

	return GroupsHierarchical(anchors, results), nil
}

func (m *Recommend) PostSort(anchors []int64) ([][]int64, error) {
	var predicates []*NamedPredicate[int64]
	for _, sorter := range m.PostSorts {
		predicate, err := sorter.Sort(anchors)
		if err != nil {
			return nil, err
		}
		predicates = append(predicates, predicate)
	}

	//convert
	var results []NamedPredicate[int64]
	for _, predicate := range predicates {
		results = append(results, *predicate)
	}

	return GroupsHierarchical(anchors, results), nil
}

// Fetch 获取推荐的主播列表
func (m *Recommend) Fetch(size int) ([]int64, error) {
	//1. 召回主播
	anchors, err := m.Recaller.Recall()
	if err != nil {
		return nil, err
	}

	// 没有召回到主播
	if len(anchors) == 0 {
		return nil, nil
	}

	// 打乱召回的主播顺序
	Shuffle(anchors)

	// 2. 过滤
	filteredAnchors, err := m.Filter(anchors)
	if err != nil {
		return nil, err
	}
	// 如果过滤后没有主播，直接返回
	if len(filteredAnchors) == 0 {
		return nil, nil
	}

	// 3. 排序
	groupSortedAnchors, err := m.GroupSort(filteredAnchors)
	if err != nil {
		return nil, err
	}

	// 4. 分组内乱序
	for i := range groupSortedAnchors {
		Shuffle(groupSortedAnchors[i])
	}

	// 5. 扁平化
	flat := Flat(groupSortedAnchors)

	// 6. 后处理排序
	postSortedAnchors, err := m.PostSort(flat)
	if err != nil {
		return nil, err
	}

	// 7. 再次扁平化
	flat = Flat(postSortedAnchors)

	return flat[:min(size, len(flat))], nil
}

func NewNoAnchorErrRecommend(userid int64, recaller Recaller, filterRules []FilterRule, sorters, postSorters []Sorter, err error) Recommender {
	return &NoAnchorErrRecommend{
		Recommend: &Recommend{
			UserId:    userid,
			Recaller:  recaller,
			Rules:     filterRules,
			Sorts:     sorters,
			PostSorts: postSorters,
			rClient:   nil,
		},
		err: err,
	}
}

// NoAnchorErrRecommend 召回错误推荐
type NoAnchorErrRecommend struct {
	*Recommend
	err error
}

func (f *NoAnchorErrRecommend) Fetch(size int) ([]int64, error) {
	fetch, err := f.Recommend.Fetch(size)
	if err != nil {
		return nil, err
	}

	if len(fetch) == 0 {
		return nil, f.err // 返回错误，表示没有假主播
	}

	return fetch, nil
}

func NewMemoryRecommend(recommdner Recommender) Recommender {
	return &MemoryRecommend{
		Recommender: recommdner,
		redisCli:    nil,
		MemoryKey:   "recommender:memory", // 默认的内存键
	}
}

// MemoryRecommend 记忆推荐
type MemoryRecommend struct {
	Recommender
	redisCli  *redis.Client
	MemoryKey string //下发过的key
}

// Fetch 获取推荐的主播列表
func (m *MemoryRecommend) Fetch(size int) ([]int64, error) {
	result, err := m.Recommender.Fetch(size)
	if err != nil {
		return nil, err
	}

	var zsetMembers []*redis.Z
	for _, anchorId := range result {
		zsetMembers = append(zsetMembers, &redis.Z{
			Score:  float64(time.Now().Unix()),
			Member: anchorId,
		})
	}

	if len(zsetMembers) == 0 {
		return result, nil
	}

	//使用sorted set
	err = m.redisCli.ZAdd(context.Background(), m.MemoryKey, zsetMembers...).Err()
	if err != nil {
		return nil, err
	}

	return result, nil
}

func NewRetryRecommend(recommender Recommender, retryCount int) Recommender {
	return &RetryRecommend{
		Recommender: recommender,
		retryCount:  retryCount,
	}
}

type RetryRecommend struct {
	Recommender
	retryCount int // 重试次数
}

func (r *RetryRecommend) Fetch(size int) ([]int64, error) {

	fetchSize := size
	var result []int64
	for i := 0; i < r.retryCount; i++ {
		fetch, err := r.Recommender.Fetch(fetchSize)
		if err != nil {
			return nil, err
		}
		// 找到了一些主播
		if len(fetch) > 0 {
			result = append(result, fetch...)
		}

		// 是否获取到足够的主播
		if len(result) >= size {
			return result, nil
		}

		// 如果获取的主播数量不足，继续尝试
		fetchSize = size - len(result)
		if fetchSize <= 0 {
			break
		}
	}

	return result, nil
}
