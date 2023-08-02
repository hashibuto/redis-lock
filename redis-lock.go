package rlock

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	redis "github.com/redis/go-redis/v9"
)

type Lock struct {
	client      *redis.Client
	key         string
	releaseChan chan struct{}
	waitGroup   *sync.WaitGroup
}

func NewLock(client *redis.Client, key string) *Lock {
	return &Lock{
		client:    client,
		key:       key,
		waitGroup: &sync.WaitGroup{},
	}
}

func (l *Lock) TryLock(interval time.Duration) (bool, error) {
	acquired, err := l.client.SetNX(context.Background(), l.key, "", interval*3*time.Second).Result()
	if err != nil {
		return false, err
	}

	if !acquired {
		return false, nil
	}

	l.releaseChan = make(chan struct{}, 1)
	l.waitGroup.Add(1)

	go l.lockThread(interval)
	return true, nil
}

func (l *Lock) Lock(interval time.Duration) error {
	sub := l.client.Subscribe(context.Background(), fmt.Sprintf("redis-lock:channel:%s", l.key))
	for {
		acquired, err := l.TryLock(interval)
		if err != nil {
			return err
		}
		if acquired {
			sub.Close()
			return nil
		}
		sub.Receive(context.Background())
	}
}

func (l *Lock) Release() error {
	close(l.releaseChan)
	l.waitGroup.Wait()
	_, err := l.client.Del(context.Background(), l.key).Result()
	l.client.Publish(context.Background(), fmt.Sprintf("redis-lock:channel:%s", l.key), "RELEASE")
	return err
}

func (l *Lock) lockThread(interval time.Duration) {
	defer l.waitGroup.Done()
	ticker := time.NewTicker(interval)
	for {
		select {
		case <-l.releaseChan:
			ticker.Stop()
			return
		case <-ticker.C:
			for i := 0; i < 3; i++ {
				_, err := l.client.Set(context.Background(), l.key, "", interval*3*time.Second).Result()
				if err == nil {
					break
				}

				if i < 2 {
					time.Sleep(3 * time.Second)
				} else {
					fmt.Fprintf(os.Stderr, "Error writing to redis client for key: %s", l.key)
				}
			}
		}
	}
}
