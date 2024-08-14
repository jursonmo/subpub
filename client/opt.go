package client

import (
	"time"

	"github.com/go-kratos/kratos/v2/encoding"
	"github.com/jursonmo/subpub/common"
	"github.com/jursonmo/subpub/session"
)

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

func WithDialTimeout(t time.Duration) ClientOption {
	return func(o *Client) {
		o.dialTimeout = t
	}
}

func WithDialIntvl(t time.Duration) ClientOption {
	return func(o *Client) {
		o.dialIntvl = t
	}
}

func WithHbTimeout(t time.Duration) ClientOption {
	return func(o *Client) {
		o.hbTimeout = t
	}
}

func WithHbIntvl(t time.Duration) ClientOption {
	return func(o *Client) {
		o.hbIntvl = t
	}
}

func WithOnDialFial(h func(string, error)) ClientOption {
	return func(o *Client) {
		o.dialFialHandler = h
	}
}

func WithOnConnect(h func(session.Sessioner)) ClientOption {
	return func(o *Client) {
		o.onConnectHandler = h
	}
}

func WithOnDisconnect(h func(session.Sessioner, error)) ClientOption {
	return func(o *Client) {
		o.onDisConnHandler = h
	}
}

func WithOnStop(h func(session.Sessioner)) ClientOption {
	return func(o *Client) {
		o.onStopHandler = h
	}
}

func AddSubsrciberPath() ClientOption {
	return func(o *Client) {
		o.url += common.SubscriberPath
	}
}

func AddPublisherPath() ClientOption {
	return func(o *Client) {
		o.url += common.PublisherPath
	}
}
