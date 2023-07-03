package client

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"net/url"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/go-kratos/kratos/v2/encoding"
	_ "github.com/go-kratos/kratos/v2/encoding/json"
	ws "github.com/gorilla/websocket"
	"github.com/jursonmo/subpub/common"
	"github.com/jursonmo/subpub/message"
)

//client v2.x.x
//1. client Start 后，底层连接会一直重连直到client Stopped
//2. client 可以通过 ctx 到期或者主动cancel或者主动调用client.Stop() 来停止

//todo:
//1. 有没有可能close conn 后，c.conn.ReadMessage 依然阻塞的情况？
//2. 订阅或发布消息 同步返回处理结果

type Client struct {
	sync.Mutex
	ctx         context.Context
	cancel      context.CancelFunc
	conn        *ws.Conn
	closed      bool
	isconnected bool
	//log    log.Logger
	//hlog     *log.Helper
	url      string
	endpoint *url.URL

	codec encoding.Codec

	timeout time.Duration

	chanBufSize int

	//pubHandler map[message.Topic]PushMsgHandler
	pubHandler sync.Map
	subChan    map[message.Topic]chan []byte

	sendChan chan []byte
	stopSend chan struct{}

	//每次dial 的超时时间和失败回调，什么时候放弃dial, ctx cancel或者到期或者 主动client.Stop()
	dialTimeout      time.Duration
	dialIntvl        time.Duration //有时遇到refuse 马上返回的情况，应该间隔一定的时间再发起连接
	hbTimeout        time.Duration
	hbIntvl          time.Duration
	dialFialHandler  func(endpoint string, err error)
	onConnectHandler func(endpoint string, conn net.Conn)
	onDisConnHandler func(endpoint string, err error)
	onStopHandler    func(endpoint string)
	eg               *errgroup.Group //for heartbeat, readMessage
}

var SubscriberPath = "/subscribe"
var PublisherPath = "/publish"

func NewSubcriber(opts ...ClientOption) *Client {
	return NewClient(append(opts, AddSubsrciberPath())...)
}

func NewPublisher(opts ...ClientOption) *Client {
	return NewClient(append(opts, AddPublisherPath())...)
}

func NewClient(opts ...ClientOption) *Client {
	cli := &Client{
		url:         "",
		timeout:     1 * time.Second,
		codec:       encoding.GetCodec("json"),
		chanBufSize: 256,
		subChan:     make(map[message.Topic]chan []byte),
		sendChan:    make(chan []byte),
		dialTimeout: time.Second * 5,
		dialIntvl:   time.Second * 3,
		hbIntvl:     time.Second * 3,
		hbTimeout:   time.Second * 10,
	}

	cli.init(opts...)

	return cli
}

func (c *Client) init(opts ...ClientOption) {
	for _, o := range opts {
		o(c)
	}

	c.endpoint, _ = url.Parse(c.url)
}

func (c *Client) Start(ctx context.Context) error {
	if c.endpoint == nil {
		return errors.New("endpoint is nil")
	}
	c.ctx, c.cancel = context.WithCancel(ctx)
	go func() {
		for {
			if err := c.ctx.Err(); err != nil {
				log.Printf("quit master task, endpoint:%v\n", c.endpoint.String())
				c.Stop(context.Background())
				return
			}
			log.Printf("[websocket] connecting to %s", c.endpoint.String())
			dialer := *ws.DefaultDialer
			dialer.HandshakeTimeout = c.dialTimeout
			conn, resp, err := dialer.DialContext(ctx, c.endpoint.String(), nil)
			if err != nil {
				log.Printf("err:%s, resp:[%v]", err.Error(), resp)
				if c.dialFialHandler != nil {
					c.dialFialHandler(c.endpoint.String(), err)
				}
				time.Sleep(c.dialIntvl)
				continue
			}

			c.conn = conn
			c.connected()
			if c.onConnectHandler != nil {
				c.onConnectHandler(c.endpoint.String(), conn.UnderlyingConn())
			}
			c.eg = new(errgroup.Group)
			c.eg.Go(func() error {
				return c.heartbeat()
			})
			c.eg.Go(func() error {
				return c.readMessage()
			})
			c.eg.Go(func() error {
				return c.sendMessage()
			})
			err = c.eg.Wait()
			log.Printf("wait endpoint:%v, err:%v\n", c.endpoint.String(), err)
			if c.onDisConnHandler != nil {
				c.onDisConnHandler(c.endpoint.String(), err)
			}
		}
	}()
	return nil
}

func (c *Client) Stop(ctx context.Context) error {
	c.Lock()
	defer c.Unlock()

	if c.closed {
		return nil
	}
	c.closed = true
	log.Printf("client:%v Stopping\n", c)

	//清理
	if c.onStopHandler != nil {
		c.onStopHandler(c.endpoint.String())
	}
	if c.cancel != nil {
		c.cancel()
	}

	//notify subscribe channel
	for _, ch := range c.subChan {
		close(ch)
	}
	log.Printf("client:%v Stopped\n", c)
	return nil
}

/*
func (c *Client) Connect(ctx context.Context) error {
	if c.endpoint == nil {
		return errors.New("endpoint is nil")
	}

	//c.log.Infof("[websocket] connecting to %s", c.endpoint.String())
	log.Printf("[websocket] connecting to %s", c.endpoint.String())
	conn, resp, err := ws.DefaultDialer.DialContext(ctx, c.endpoint.String(), nil)
	if err != nil {
		log.Printf("%s [%v]", err.Error(), resp)
		return err
	}
	c.conn = conn
	c.subChan = make(map[message.Topic]chan []byte)
	c.ctx, c.cancel = context.WithCancel(ctx)
	go c.heartbeat()
	return nil
}
*/

func (c *Client) Publish(topic string, content []byte) error {
	msg := message.PubMsg{Topic: message.Topic(topic), Body: content}
	data, err := c.codec.Marshal(msg)
	if err != nil {
		return err
	}
	//return c.conn.WriteMessage(ws.BinaryMessage, data)
	c.sendChan <- data
	return nil
}
func (c *Client) connected() {
	c.Lock()
	defer c.Unlock()
	c.isconnected = true
}

//处理当次连接断开的清理工作
func (c *Client) Disconnect() {
	c.Lock()
	defer c.Unlock()
	if !c.isconnected {
		return
	}
	log.Printf("closing conn:%v<->%v\n", c.conn.LocalAddr(), c.conn.RemoteAddr())
	if err := c.conn.Close(); err != nil {
		log.Printf("[websocket] disconnect error: %s", err.Error())
	}
	c.isconnected = false
	close(c.stopSend)
}

func (c *Client) heartbeat() error {
	defer c.Disconnect()
	defer log.Printf("client:%v, heartbeat quit\n", c)

	common.SetReadDeadline(c.conn, false, c.hbTimeout)
	pingTimer := time.NewTicker(c.hbIntvl)
	defer pingTimer.Stop()

	pingData := []byte("ping")
	for {
		select {
		case <-c.ctx.Done():
			return c.ctx.Err()
		case <-pingTimer.C:
			err := c.conn.WriteMessage(ws.PingMessage, pingData)
			if err != nil {
				log.Printf("Error during writing ping to websocket:%v \n", err)
				return err
			}
		}
	}

}

func (c *Client) Channel(topic message.Topic) chan []byte {
	c.Lock()
	defer c.Unlock()

	ch, ok := c.subChan[topic]
	if ok {
		return ch
	}
	ch = make(chan []byte, c.chanBufSize)
	c.subChan[topic] = ch
	return ch
}

/*
func (c *Client) putPubMsg(topic string, data []byte) {
	c.Lock()
	ch, ok := c.subChan[message.Topic(topic)]
	if !ok {
		c.Unlock()
		return
	}
	c.Unlock()
	ch <- data
}
*/

type PushMsgHandler func(string, []byte)

//异步发送, todo, 同步发送，即需要sendMessage反馈结果？
func (c *Client) sendSubscribe(topic string) error {
	m := message.SubMsg{Topic: message.Topic(topic), Op: message.SubscribeOp}
	data, err := c.codec.Marshal(m)
	if err != nil {
		return err
	}
	c.sendChan <- data
	return nil
}

//异步发送
func (c *Client) sendUnsubscribe(topic string) error {
	m := message.SubMsg{Topic: message.Topic(topic), Op: message.UnsubscribeOp}
	data, err := c.codec.Marshal(m)
	if err != nil {
		return err
	}
	c.sendChan <- data
	return nil
}

// func (c *Client) sendMessage(m message.SubMsg) error {
// 	data, err := c.codec.Marshal(m)
// 	if err != nil {
// 		return err
// 	}
// 	err = c.conn.WriteMessage(ws.BinaryMessage, data)
// 	if err != nil {
// 		return err
// 	}
// 	return nil
// }

func (c *Client) sendMessage() (err error) {
	defer c.Disconnect()
	defer func() {
		log.Printf("client:%v sendMessage quit, err:%v\n", c, err)
	}()
	c.stopSend = make(chan struct{}, 1)
	for {
		select {
		case <-c.ctx.Done():
			return c.ctx.Err()
		case <-c.stopSend:
			return errors.New("stopSend")
		case d, ok := <-c.sendChan:
			if !ok {
				return errors.New("sendChan closed")
			}
			err = c.conn.WriteMessage(ws.BinaryMessage, d)
			if err != nil {
				return err
			}
		}
	}
}

func (c *Client) SubscribeWithHandler(topic string, handler PushMsgHandler) error {
	if err := c.sendSubscribe(topic); err != nil {
		return err
	}

	c.pubHandler.Store(message.Topic(topic), handler)
	return nil
}

func (c *Client) Subscribe(topic string) (chan []byte, error) {
	// if err := c.sendSubscribe(topic); err != nil {
	// 	return nil, err
	// }

	//ch := c.Channel(message.Topic(topic))
	//c.pubHandler.Store(message.Topic(topic), c.putPubMsg)
	//c.pubHandler.Store(message.Topic(topic), PushMsgHandler(func(s string, b []byte) { ch <- b }))

	// c.once.Do(
	// 	func() {
	// 		go c.readMessage()
	// 	})
	ch := c.Channel(message.Topic(topic))
	c.SubscribeWithHandler(topic, PushMsgHandler(func(s string, b []byte) { ch <- b }))
	return ch, nil
}

func (c *Client) Unsubscribe(topic string) error {
	if err := c.sendUnsubscribe(topic); err != nil {
		return err
	}
	c.pubHandler.Delete(message.Topic(topic))
	return nil
}

func (c *Client) String() string {
	return fmt.Sprintf("%v<->%v", c.conn.LocalAddr(), c.conn.RemoteAddr())
}

func (c *Client) readMessage() error {
	defer c.Disconnect()
	defer log.Printf("client:%v readMessage quit\n", c)
	for {
		//todo:有没有可能close conn 后，c.conn.ReadMessage 依然阻塞的情况？
		messageType, data, err := c.conn.ReadMessage()
		if err != nil {
			log.Printf("readMessage err:%v\n", err)
			return err
		}
		switch messageType {
		case ws.CloseMessage:
			return errors.New("receive CloseMessage")
		case ws.BinaryMessage:
			m := message.PubMsg{}
			err = c.codec.Unmarshal(data, &m)
			if err != nil {
				log.Println(err)
				continue
			}
			if h, ok := c.pubHandler.Load(m.Topic); ok {
				h.(PushMsgHandler)(string(m.Topic), m.Body)
			}
		}
	}
}
