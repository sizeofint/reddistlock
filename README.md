# RedDistLock

Redis Distributed Lock - implemented via Redis PubSub and Lists  

## Example  
```go
package main

import (
	"context"
	"github.com/go-redis/redis/v8"
	"github.com/sizeofint/reddistlock"
	"log"
)

var redDistLock reddistlock.RedDistLock

func main() {
    redDistLock = reddistlock.New(redis.NewClient(&redis.Options{
        Addr:     "127.0.0.1:6379",
        Password: "",
        DB:       0,
    }))
    
    testExclusiveDistributedExecution(context.Background())
}

func testExclusiveDistributedExecution(ctx context.Context) {
    lk, err := redDistLock.Lock(ctx, "unique-lock-key", 5)
    if err != nil {
        log.Fatalln(err)
    }
    defer lk.Unlock(ctx)
    /**
    Do some exclusive work here
    */
}
```
Please also check out [examples](examples) folder  