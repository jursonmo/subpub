package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/jursonmo/subpub/client"
	"github.com/jursonmo/subpub/message"
	"github.com/jursonmo/subpub/session"
)

var topic1 = "topic1"
var topic2 = "topic2"
var wg = sync.WaitGroup{}

func main() {
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	ctx, cancel := context.WithCancel(context.Background())
	_ = cancel

	//subscribe topic1, topic2
	subCli := subscribe(ctx)

	//publish topic1, topic2
	time.Sleep(time.Second)
	pubCli := publish(ctx)

	time.Sleep(time.Second)
	//test Unsubscribe
	err := subCli.Unsubscribe(topic1)
	if err != nil {
		log.Panic(err)
	}
	err = pubCli.Publish(topic1, []byte("shoudn't recevie this mesage"))
	if err != nil {
		log.Panic(err)
	}

	<-interrupt
	log.Println("receive interrup signal")
	//cancel()
	err = subCli.Stop(ctx)
	if err != nil {
		log.Panic(err)
	}
	err = pubCli.Stop(ctx)
	if err != nil {
		log.Panic(err)
	}
	wg.Wait()
	time.Sleep(time.Second * 2)
}

func subscribe(ctx context.Context) *client.Client {
	// cli := client.NewClient(
	// 	client.WithEndpoint("ws://localhost:8000"+client.SubscriberPath),
	// 	client.WithClientCodec("json"),
	// )
	onConnect := func(s session.Sessioner) {
		fmt.Printf("subscriber client connected, id:%s, remote:%v", s.SessionID(), s.UnderlayConn().RemoteAddr())
	}
	cli := client.NewSubcriber(
		client.WithEndpoint("ws://localhost:8000"),
		client.WithClientCodec("json"),
		client.WithOnConnect(onConnect),
	)

	err := cli.Start(ctx)
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

	handler := func(topic string, data []byte) {
		fmt.Printf("handler: receive message, topic:%s, data:%s\n", topic, string(data))
	}
	err = cli.SubscribeWithHandler(topic2, message.TopicMsgHandler(handler))
	if err != nil {
		log.Panic(err)
	}
	return cli
}

func publish(ctx context.Context) *client.Client {
	/*
		pubCli := client.NewClient(
			//client.WithEndpoint("ws://localhost:8000/publish"),
			client.WithEndpoint("ws://localhost:8000"+client.PublisherPath),
			client.WithClientCodec("json"),
		)
	*/
	pubCli := client.NewPublisher(
		client.WithEndpoint("ws://localhost:8000"),
		client.WithClientCodec("json"),
	)
	err := pubCli.Start(ctx)
	if err != nil {
		log.Panic(err)
	}
	err = pubCli.Publish(topic1, []byte("publish topic1 message"))
	if err != nil {
		log.Panic(err)
	}
	err = pubCli.Publish(topic2, []byte("publish topic2 message"))
	if err != nil {
		log.Panic(err)
	}
	return pubCli
}

func ShowSubscirbe(topic string, ch chan []byte) {
	// for {
	// 	select {
	// 	case data, ok := <-ch:
	// 		if !ok {
	// 			log.Printf("subscribe topic:%s close", topic)
	// 			return
	// 		}
	// 		log.Printf("subscriber receive topic:%s, content:%s\n", topic, string(data))
	// 	}
	// }

	for data := range ch {
		log.Printf("subscriber chan receive topic:%s, content:%s\n", topic, string(data))
	}
	log.Printf("subscribe topic:%s close", topic)
}
