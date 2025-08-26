package adaptivelist

import (
    "context"
    "fmt"

    "github.com/redis/go-redis/v9"
)

type AdaptiveRedisList struct {
    client    *redis.Client
    key       string
    threshold int64
    ctx       context.Context
}

// NewAdaptiveRedisList 생성자
func NewAdaptiveRedisList(client *redis.Client, key string, threshold int64) *AdaptiveRedisList {
    return &AdaptiveRedisList{
        client:    client,
        key:       key,
        threshold: threshold,
        ctx:       context.Background(),
    }
}

// 내부가 Tree 모드(ZSET)인지 확인
func (a *AdaptiveRedisList) isTreeMode() (bool, error) {
    typ, err := a.client.Type(a.ctx, a.key+":tree").Result()
    if err != nil {
        return false, err
    }
    return typ == "zset", nil
}

// push: 자동 전환 로직 포함
func (a *AdaptiveRedisList) Push(value string) error {
    size, err := a.client.LLen(a.ctx, a.key).Result()
    if err != nil && err != redis.Nil {
        return err
    }

    treeMode, _ := a.isTreeMode()
    if size+1 >= a.threshold && !treeMode {
        if err := a.migrateToTree(); err != nil {
            return err
        }
    }

    treeMode, _ = a.isTreeMode()
    if treeMode {
        score, _ := a.client.ZCard(a.ctx, a.key+":tree").Result()
        return a.client.ZAdd(a.ctx, a.key+":tree", redis.Z{Score: float64(score + 1), Member: value}).Err()
    }
    return a.client.RPush(a.ctx, a.key, value).Err()
}

// pop: List 또는 Tree 모드에 따라 동작
func (a *AdaptiveRedisList) Pop() (string, error) {
    treeMode, _ := a.isTreeMode()
    if treeMode {
        vals, err := a.client.ZRevRange(a.ctx, a.key+":tree", 0, 0).Result()
        if err != nil || len(vals) == 0 {
            return "", err
        }
        if err := a.client.ZRem(a.ctx, a.key+":tree", vals[0]).Err(); err != nil {
            return "", err
        }
        return vals[0], nil
    }
    return a.client.RPop(a.ctx, a.key).Result()
}

// range: List 또는 Tree 모드에 따라 동작
func (a *AdaptiveRedisList) Range(start, end int64) ([]string, error) {
    treeMode, _ := a.isTreeMode()
    if treeMode {
        return a.client.ZRange(a.ctx, a.key+":tree", start, end).Result()
    }
    return a.client.LRange(a.ctx, a.key, start, end).Result()
}

// get: 특정 인덱스 조회
func (a *AdaptiveRedisList) Get(index int64) (string, error) {
    treeMode, _ := a.isTreeMode()
    if treeMode {
        vals, err := a.client.ZRange(a.ctx, a.key+":tree", index, index).Result()
        if err != nil || len(vals) == 0 {
            return "", err
        }
        return vals[0], nil
    }
    return a.client.LIndex(a.ctx, a.key, index).Result()
}

// migrateToTree: List -> ZSET 전환
func (a *AdaptiveRedisList) migrateToTree() error {
    values, err := a.client.LRange(a.ctx, a.key, 0, -1).Result()
    if err != nil {
        return err
    }
    if err := a.client.Del(a.ctx, a.key).Err(); err != nil {
        return err
    }
    for i, v := range values {
        if err := a.client.ZAdd(a.ctx, a.key+":tree", redis.Z{Score: float64(i), Member: v}).Err(); err != nil {
            return err
        }
    }
    return nil
}
