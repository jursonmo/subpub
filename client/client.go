package client

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/url"
	"sync"
	"time"

	"github.com/go-kratos/kratos/v2/encoding"
	_ "github.com/go-kratos/kratos/v2/encoding/json"
	ws "github.com/gorilla/websocket"
	"github.com/jursonmo/subpub/common"
	"github.com/jursonmo/subpub/message"
)

//subpub client sdk
//subscribe with handler
//sync.Map

type Client struct {
	sync.Mutex
	ctx    context.Context
	cancel context.CancelFunc
	conn   *ws.Conn
	closed bool
	//log    log.Logger
	//hlog     *log.Helper
	url      string
	endpoint *url.URL

	codec encoding.Codec

	timeout time.Duration

	once        sync.Once
	chanBufSize int

	//pubHandler map[message.Topic]PushMsgHandler
	pubHandler sync.Map
	subChan    map[message.Topic]chan []byte
}

type ClientOption func(o *Client)

func WithClientCodec(c string) ClientOption {
	return func(o *Client) {
		o.codec = encoding.GetCodec(c)
	}
}

func WithEndpoint(uri string) ClientOption {
	return func(o *Client) {
		o.url = uri
	}
}

var SubscriberPath = "/subscribe"
var PublisherPath = "/publish"

func AddSubsrciberPath() ClientOption {
	return func(o *Client) {
		o.url += SubscriberPath
	}
}

func AddPublisherPath() ClientOption {
	return func(o *Client) {
		o.url += PublisherPath
	}
}

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
	go c.Heartbeat()
	return nil
}

func (c *Client) Publish(topic string, content []byte) error {
	msg := message.PubMsg{Topic: message.Topic(topic), Body: content}
	data, err := c.codec.Marshal(msg)
	if err != nil {
		return err
	}
	return c.conn.WriteMessage(ws.BinaryMessage, data)
}

func (c *Client) Disconnect() {
	c.Lock()
	defer c.Unlock()
	if c.closed {
		return
	}
	c.closed = true
	if c.cancel != nil {
		c.cancel()
	}
	if err := c.conn.Close(); err != nil {
		log.Printf("[websocket] disconnect error: %s", err.Error())
	}

	//notify subscribe channel
	for _, ch := range c.subChan {
		close(ch)
	}
	log.Println("client close over")
}

func (c *Client) Heartbeat() {
	defer c.Disconnect()
	defer log.Printf("client:%v, Heartbeat quit\n", c)
	common.SetReadDeadline(c.conn, false, time.Second*5)
	pingTimer := time.NewTicker(time.Second * 2)
	pingData := []byte("ping")
	for {
		select {
		case <-c.ctx.Done():
			return
		case <-pingTimer.C:
			err := c.conn.WriteMessage(ws.PingMessage, pingData)
			if err != nil {
				log.Printf("Error during writing ping to websocket:%v \n", err)
				return
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

func (c *Client) sendSubscribe(topic string) error {
	m := message.SubMsg{Topic: message.Topic(topic), Op: message.SubscribeOp}
	return c.sendMessage(m)
}

func (c *Client) sendUnsubscribe(topic string) error {
	m := message.SubMsg{Topic: message.Topic(topic), Op: message.UnsubscribeOp}
	return c.sendMessage(m)
}

func (c *Client) sendMessage(m message.SubMsg) error {
	data, err := c.codec.Marshal(m)
	if err != nil {
		return err
	}
	err = c.conn.WriteMessage(ws.BinaryMessage, data)
	if err != nil {
		return err
	}
	return nil
}

func (c *Client) SubscribeWithHandler(topic string, handler PushMsgHandler) error {
	if err := c.sendSubscribe(topic); err != nil {
		return err
	}

	c.pubHandler.Store(message.Topic(topic), handler)

	c.once.Do(
		func() {
			go c.readMessage()
		})
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

func (c *Client) readMessage() {
	defer c.Disconnect()
	defer log.Printf("client:%v readMessage quit\n", c)
	for {
		messageType, data, err := c.conn.ReadMessage()
		if err != nil {
			log.Printf("readMessage err:%v\n", err)
			return
		}
		switch messageType {
		case ws.CloseMessage:
			return
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
