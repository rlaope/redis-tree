package test

import (
	"fmt"
	"testing"

	"github.com/redis/go-redis/v9"
	"github.com/example/adaptive-redis-list/src/adaptivelist"
)

func setupClient() *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
		DB:   0,
	})
}

func TestAdaptiveListPushPop(t *testing.T) {
	client := setupClient()
	client.FlushAll(client.Context())

	list := adaptivelist.NewAdaptiveRedisListExt(client, "testlist", 8, 4)

	// Push 7개까지는 List 모드
	for i := 0; i < 7; i++ {
		if err := list.Push(fmt.Sprintf("val%d", i)); err != nil {
			t.Fatalf("Push failed: %v", err)
		}
	}
	if mode := list.Mode(); mode != "list" {
		t.Errorf("Expected list mode, got %s", mode)
	}

	// 8번째 추가 → Tree 모드 전환
	_ = list.Push("val7")
	if mode := list.Mode(); mode != "tree" {
		t.Errorf("Expected tree mode after threshold, got %s", mode)
	}

	// Pop 여러번 해서 reverseThreshold 이하로 줄이면 List 모드 복귀
	for i := 0; i < 5; i++ {
		_, _ = list.Pop()
	}
	if mode := list.Mode(); mode != "list" {
		t.Errorf("Expected list mode after shrinking, got %s", mode)
	}
}

func TestAdaptiveListLPushLPop(t *testing.T) {
	client := setupClient()
	client.FlushAll(client.Context())

	list := adaptivelist.NewAdaptiveRedisListExt(client, "deque", 8, 4)

	_ = list.LPush("left1")
	_ = list.LPush("left2")
	_ = list.Push("right1")

	val, _ := list.LPop()
	if val != "left2" {
		t.Errorf("Expected left2, got %s", val)
	}

	val, _ = list.Pop()
	if val != "right1" {
		t.Errorf("Expected right1, got %s", val)
	}
}

func TestAdaptiveListPushMany(t *testing.T) {
	client := setupClient()
	client.FlushAll(client.Context())

	list := adaptivelist.NewAdaptiveRedisListExt(client, "batch", 8, 4)

	values := []string{"a", "b", "c", "d"}
	if err := list.PushMany(values); err != nil {
		t.Fatalf("PushMany failed: %v", err)
	}

	result, _ := list.Range(0, -1)
	if len(result) != len(values) {
		t.Errorf("Expected %d values, got %d", len(values), len(result))
	}
}

