package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/jursonmo/subpub/client"
)

var topic1 = "xxx1"
var topic2 = "xxx2"
var wg = sync.WaitGroup{}

func main() {
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	ctx, cancel := context.WithCancel(context.Background())

	//subscribe
	subscribe(ctx)

	//publish test
	time.Sleep(time.Second)
	publish(ctx)

	<-interrupt
	cancel()
	wg.Wait()
}

func subscribe(ctx context.Context) {
	cli := client.NewClient(
		client.WithEndpoint("ws://localhost:8000/subscribe"),
		client.WithClientCodec("json"),
	)
	err := cli.Connect(ctx)
	if err != nil {
		log.Panic(err)
	}

	//一个client 可以订阅不同的topic
	ch1, err := cli.Subscribe(topic1)
	if err != nil {
		log.Panic(err)
	}
	wg.Add(1)
	go func() {
		defer wg.Done()
		ShowSubscirbe(topic1, ch1)
	}()

	ch2, err := cli.Subscribe(topic2)
	if err != nil {
		log.Panic(err)
	}
	wg.Add(1)
	go func() {
		defer wg.Done()
		ShowSubscirbe(topic2, ch2)
	}()
}

func publish(ctx context.Context) {
	pubCli := client.NewClient(
		client.WithEndpoint("ws://localhost:8000/publish"),
		client.WithClientCodec("json"),
	)
	err := pubCli.Connect(ctx)
	if err != nil {
		log.Panic(err)
	}
	err = pubCli.Publish(topic1, []byte("publis test"))
	if err != nil {
		log.Panic(err)
	}
	err = pubCli.Publish(topic2, []byte("publis test"))
	if err != nil {
		log.Panic(err)
	}
}

func ShowSubscirbe(topic string, ch chan []byte) {
	for {
		select {
		case data, ok := <-ch:
			if !ok {
				log.Printf("subscribe topic:%s close", topic)
				return
			}
			log.Printf("subscriber receive topic:%s, content:%s\n", topic, string(data))
		}
	}
}
