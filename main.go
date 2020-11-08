package main

import (
    "fmt"
    "github.com/go-redis/redis/v8"
)

func main() {

	fmt.Println("Hello World")
	
	client := redis.NewClient(&redis.Options{
        Addr:     "redis:6379",
        Password: "", // no password set
        DB:       0,  // use default DB
    })
    pong, err := client.Ping().Result()
	fmt.Println(pong, err)
	
}