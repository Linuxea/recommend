package main

import (
	"fmt"

	"linuxea.github.com/recommend/recommend"
)

func main() {

	recommend := &Builder{}
	fetch, err := recommend.Build().Fetch(100)
	if err != nil {
		panic(err)
	}
	fmt.Println("推荐结果:", fetch)
}

type isOddUser struct{}

func (f isOddUser) Filter(userId int64, anchorIdList []int64) ([]int64, error) {
	isOddUser := func(userId int64, userid []int64) ([]int64, error) {
		var result []int64
		for _, id := range userid {
			if id%2 == 1 { // 假设奇数用户ID是需要的
				result = append(result, id)
			}
		}
		return result, nil
	}
	return isOddUser(userId, anchorIdList)
}

type BiggerGroup struct {
}

func (b BiggerGroup) Sort(anchors []int64) (*recommend.NamedPredicate[int64], error) {
	predicate := &recommend.NamedPredicate[int64]{
		Name: "BiggerGroup",
		Predicates: []recommend.Predicate[int64]{
			func(item int64) bool {
				return item > 100 // 假设大于100的主播ID是需要的
			},
			func(item int64) bool {
				return item > 0 && item < 50 // 假设小于50的主播ID也是需要的
			},
			func(item int64) bool {
				//其他的
				return item < 0
			},
		},
	}
	return predicate, nil
}

type MatchRecaller struct {
}

func (m *MatchRecaller) Recall() ([]int64, error) {
	// 假设召回了一些主播ID
	return []int64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 101, 201, 301, 401, 500, 601, 700, 800, 991, 1000}, nil
}

type Builder struct {
	userid int64
}

func (b *Builder) Build() recommend.Recommender {
	return recommend.NewRecommend(b.userid, &MatchRecaller{}, []recommend.FilterRule{&isOddUser{}}, []recommend.Sorter{&BiggerGroup{}}, nil)
}
