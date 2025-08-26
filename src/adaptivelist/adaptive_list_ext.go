package adaptivelist

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
)

type AdaptiveRedisListExt struct {
	client          *redis.Client
	key             string
	threshold       int64
	reverseThreshold int64
	ctx             context.Context
}

// NewAdaptiveRedisListExt 생성자
func NewAdaptiveRedisListExt(client *redis.Client, key string, threshold, reverseThreshold int64) *AdaptiveRedisListExt {
	return &AdaptiveRedisListExt{
		client:          client,
		key:             key,
		threshold:       threshold,
		reverseThreshold: reverseThreshold,
		ctx:             context.Background(),
	}
}

// 현재 Tree 모드인지 확인
func (a *AdaptiveRedisListExt) isTreeMode() (bool, error) {
	typ, err := a.client.Type(a.ctx, a.key+":tree").Result()
	if err != nil {
		return false, err
	}
	return typ == "zset", nil
}

// Mode: 현재 모드 반환 ("list" 또는 "tree")
func (a *AdaptiveRedisListExt) Mode() string {
	treeMode, _ := a.isTreeMode()
	if treeMode {
		return "tree"
	}
	return "list"
}

// Push (RPush)
func (a *AdaptiveRedisListExt) Push(value string) error {
	return a.pushInternal(value, false)
}

// LPush (왼쪽에 삽입)
func (a *AdaptiveRedisListExt) LPush(value string) error {
	return a.pushInternal(value, true)
}

// 내부 push 로직
func (a *AdaptiveRedisListExt) pushInternal(value string, left bool) error {
	size, _ := a.client.LLen(a.ctx, a.key).Result()
	treeMode, _ := a.isTreeMode()

	// List → Tree 전환
	if size+1 >= a.threshold && !treeMode {
		if err := a.migrateToTree(); err != nil {
			return err
		}
		treeMode = true
	}

	if treeMode {
		score, _ := a.client.ZCard(a.ctx, a.key+":tree").Result()
		if left {
			// 왼쪽 삽입은 음수 score 사용
			score = -score - 1
		} else {
			score = score + 1
		}
		return a.client.ZAdd(a.ctx, a.key+":tree", redis.Z{Score: float64(score), Member: value}).Err()
	}

	if left {
		return a.client.LPush(a.ctx, a.key, value).Err()
	}
	return a.client.RPush(a.ctx, a.key, value).Err()
}

// Pop (RPop)
func (a *AdaptiveRedisListExt) Pop() (string, error) {
	return a.popInternal(false)
}

// LPop
func (a *AdaptiveRedisListExt) LPop() (string, error) {
	return a.popInternal(true)
}

// 내부 pop 로직
func (a *AdaptiveRedisListExt) popInternal(left bool) (string, error) {
	treeMode, _ := a.isTreeMode()

	if treeMode {
		var vals []string
		var err error
		if left {
			vals, err = a.client.ZRange(a.ctx, a.key+":tree", 0, 0).Result()
		} else {
			vals, err = a.client.ZRevRange(a.ctx, a.key+":tree", 0, 0).Result()
		}
		if err != nil || len(vals) == 0 {
			return "", err
		}
		if err := a.client.ZRem(a.ctx, a.key+":tree", vals[0]).Err(); err != nil {
			return "", err
		}

		// Tree → List 역전환 검사
		count, _ := a.client.ZCard(a.ctx, a.key+":tree").Result()
		if count <= a.reverseThreshold {
			_ = a.migrateToList()
		}
		return vals[0], nil
	}

	val := ""
	var err error
	if left {
		val, err = a.client.LPop(a.ctx, a.key).Result()
	} else {
		val, err = a.client.RPop(a.ctx, a.key).Result()
	}
	if err != nil {
		return "", err
	}
	return val, nil
}

// PushMany: 여러 원소 한번에 삽입
func (a *AdaptiveRedisListExt) PushMany(values []string) error {
	for _, v := range values {
		if err := a.Push(v); err != nil {
			return err
		}
	}
	return nil
}

// migrateToTree: List → Tree 변환
func (a *AdaptiveRedisListExt) migrateToTree() error {
	values, err := a.client.LRange(a.ctx, a.key, 0, -1).Result()
	if err != nil {
		return err
	}
	_ = a.client.Del(a.ctx, a.key).Err()
	for i, v := range values {
		if err := a.client.ZAdd(a.ctx, a.key+":tree", redis.Z{Score: float64(i), Member: v}).Err(); err != nil {
			return err
		}
	}
	return nil
}

// migrateToList: Tree → List 역전환
func (a *AdaptiveRedisListExt) migrateToList() error {
	values, err := a.client.ZRange(a.ctx, a.key+":tree", 0, -1).Result()
	if err != nil {
		return err
	}
	_ = a.client.Del(a.ctx, a.key+":tree").Err()
	for _, v := range values {
		if err := a.client.RPush(a.ctx, a.key, v).Err(); err != nil {
			return err
		}
	}
	return nil
}

