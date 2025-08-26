# Adaptive Redis List (Go)

Redis List처럼 사용할 수 있지만, 길이가 특정 임계치(기본 8)를 넘으면 내부적으로 ZSET(Tree)로 변환되는 어댑터입니다.

## Features
- 처음엔 Redis List 사용
- 원소 개수가 8 이상이면 자동으로 ZSET(Tree) 모드로 전환
- Push, Pop, Range, Get API 제공

## Install
```bash
go get github.com/example/adaptive-redis-list
```

## Usage
```go
package main

import (
    "fmt"
    "github.com/redis/go-redis/v9"
    "github.com/example/adaptive-redis-list/src/adaptivelist"
)

func main() {
    client := redis.NewClient(&redis.Options{
        Addr: "localhost:6379",
    })

    list := adaptivelist.NewAdaptiveRedisList(client, "mylist", 8)

    for i := 0; i < 10; i++ {
        list.Push(fmt.Sprintf("val%d", i))
    }

    vals, _ := list.Range(0, -1)
    fmt.Println(vals)
}
```

## License
MIT
