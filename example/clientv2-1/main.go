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

/*
为了上层调用比较方便，不需要业务层处理connecting 逻辑。
采用 New ,Start(ctx), Stop(ctx) 的模式后, start 就负责在后台起一个任务负责连接重连的任务
这样就必须注册的方式，注册OnDialFail, OnConnected, OnDisconnect 等handler
什么时候能发送数据，在 OnConnected handler 里。
用ctx 来控制整体的生命周期， cancel 就能 stop 这个整个生命周期，
所以 Stop 只需要执行cancel
*/
var topic1 = "topic1"
var topic2 = "topic2"
var wg = sync.WaitGroup{}

func main() {
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	ctx, cancel := context.WithCancel(context.Background())
	_ = cancel

	//subscribe topic1, topic2
	subCli := SubcribeTopic(ctx)

	time.Sleep(time.Second)

	//publish topic1, topic2
	pubCli := publishTopic(ctx)

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
	//test cancel()
	err = subCli.Stop(ctx)
	if err != nil {
		log.Panic(err)
	}
	err = pubCli.Stop(ctx)
	if err != nil {
		log.Panic(err)
	}
	wg.Wait()
	time.Sleep(time.Second * 1)
}

func SubcribeTopic(ctx context.Context) *client.Client {
	SubcribeDone := make(chan struct{})
	onConnect := func(s session.Sessioner) {
		fmt.Printf("subscriber client connected, id:%s, remote:%v", s.SessionID(), s.UnderlayConn().RemoteAddr())
		go func() {
			ch1, err := s.Subscribe(topic1)
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
			err = s.SubscribeWithHandler(topic2, message.TopicMsgHandler(handler))
			if err != nil {
				log.Panic(err)
			}
			SubcribeDone <- struct{}{}
		}()
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

	<-SubcribeDone
	return cli
}
func ShowSubscirbe(topic string, ch chan []byte) {
	for data := range ch {
		log.Printf("subscriber chan receive topic:%s, content:%s\n", topic, string(data))
	}
	log.Printf("subscribe topic:%s close", topic)
}

func publishTopic(ctx context.Context) *client.Client {
	publishDone := make(chan struct{})
	onConnect := func(s session.Sessioner) {
		go func() {
			err := s.Publish(topic1, []byte("publish topic1 message"))
			if err != nil {
				log.Panic(err)
			}
			err = s.Publish(topic2, []byte("publish topic2 message"))
			if err != nil {
				log.Panic(err)
			}
			publishDone <- struct{}{}
		}()
	}
	pubCli := client.NewPublisher(
		client.WithEndpoint("ws://localhost:8000"),
		client.WithClientCodec("json"),
		client.WithOnConnect(onConnect),
	)
	err := pubCli.Start(ctx)
	if err != nil {
		log.Panic(err)
	}

	<-publishDone
	return pubCli
}
