package subscribe

import (
	"fmt"
	"sync"
)

type SubscriberMgr struct {
	sync.RWMutex
	subscribers     map[string]*Subscribers //key: topic,
	subscriberCheck func(from Subscriber, topic string) error
	publishCheck    func(from Subscriber, topic string, data []byte) error
}

type subscriberCheckHandler func(from Subscriber, topic string) error
type publishCheckHandler func(from Subscriber, topic string, data []byte) error
type SubscriberMgrOpt func(*SubscriberMgr)

type Subscribers struct {
	sync.RWMutex
	subMap   map[SubscriberID]Subscriber
	subSlice []Subscriber
}

func (ss *Subscribers) getAllSubscriber() []Subscriber {
	ss.RLock()
	defer ss.RUnlock()
	return ss.subSlice
}

func (ss *Subscribers) delSubscriber(s Subscriber) {
	//fmt.Printf("delelte subscriber:%v\n", s.Id())
	ss.Lock()
	defer ss.Unlock()

	delete(ss.subMap, s.Id())

	//下面就是从slice 中删除指定的subscriber, 删除前判断下slice 占用的内存和实际保存的数据是否相差太大
	//如果slice 的长度小于 slice cap的1/3(一次append可能就会使cap翻倍了)，那么为了节约内存，应该重新生成subSlice
	if len(ss.subSlice) > 128 && len(ss.subSlice) < cap(ss.subSlice)/3 {
		ss.subSlice = make([]Subscriber, 0, len(ss.subMap))
		for _, s := range ss.subMap {
			ss.subSlice = append(ss.subSlice, s)
		}
		return
	}
	//从sub slice 中删除指定的subscriber
	removeIndex := 0
	exist := false
	for i, v := range ss.subSlice {
		if v == s {
			exist = true
			removeIndex = i
			break
		}
	}
	if exist {
		ss.subSlice = append(ss.subSlice[0:removeIndex], ss.subSlice[removeIndex+1:]...)
	}

	//check
	if len(ss.subMap) != len(ss.subSlice) {
		panic(fmt.Sprintf("len(ss.subMap):%d != len(ss.subSlice):%d", len(ss.subMap), len(ss.subSlice)))
	}
}

func (ss *Subscribers) addSubscriber(s Subscriber) {
	ss.Lock()
	defer ss.Unlock()
	ss.subMap[s.Id()] = s
	ss.subSlice = append(ss.subSlice, s) //翻倍分配内存

	//check
	if len(ss.subMap) != len(ss.subSlice) {
		panic(fmt.Sprintf("len(ss.subMap):%d != len(ss.subSlice):%d", len(ss.subMap), len(ss.subSlice)))
	}
}

// 有订阅者订阅主题时，可以检测这个订阅者，以判断是否允许这个订阅者订阅。
func WithSubscribeCheck(h subscriberCheckHandler) SubscriberMgrOpt {
	return func(sm *SubscriberMgr) {
		sm.subscriberCheck = h
	}
}

// 用于在发布消息时，上层业务可以根据自己的需求控制哪些消息可以公告发布
func WithPublishCheck(h publishCheckHandler) SubscriberMgrOpt {
	return func(sm *SubscriberMgr) {
		sm.publishCheck = h
	}
}

func NewSubscriberMgr(opts ...SubscriberMgrOpt) *SubscriberMgr {
	sm := &SubscriberMgr{subscribers: make(map[string]*Subscribers)}
	for _, opt := range opts {
		opt(sm)
	}
	return sm
}

func (sm *SubscriberMgr) NewSubscriber(id SubscriberID, h MailBoxHandler) Subscriber {
	return &Subscribe{id: id, handler: h, sm: sm}
}

func (sm *SubscriberMgr) AddSubscriber(topic string, s Subscriber) error {
	if sm.subscriberCheck != nil {
		if err := sm.subscriberCheck(s, topic); err != nil {
			return err
		}
	}

	sm.Lock()
	defer sm.Unlock()

	subscribers, ok := sm.subscribers[topic]
	if !ok {
		subscribers = &Subscribers{subMap: make(map[SubscriberID]Subscriber)}
		subscribers.addSubscriber(s)
		sm.subscribers[topic] = subscribers
		return nil
	}
	subscribers.addSubscriber(s)
	return nil
}

func (sm *SubscriberMgr) RemoveSubscriber(sub Subscriber, topic string) {
	//todo: 把锁的颗粒度改小点
	sm.Lock()
	defer sm.Unlock()
	subscribers, ok := sm.subscribers[topic]
	if !ok {
		return
	}

	subscribers.delSubscriber(sub)

	//there is no subscribers on this topic?
	if len(subscribers.subMap) == 0 {
		delete(sm.subscribers, topic)
	}
}

// 返回发送给订阅者的数量
func (sm *SubscriberMgr) Publish(from Subscriber, topic string, data []byte) (int, error) {
	// 订阅者也可以发布消息的话, 这里需要检查from是否有权限发布这个topic的消息，同时也可以检查消息内容是否合规之类的
	if sm.publishCheck != nil {
		if err := sm.publishCheck(from, topic, data); err != nil {
			return 0, err
		}
	}

	sm.Lock()
	subscribers, ok := sm.subscribers[topic]
	if !ok {
		sm.Unlock()
		return 0, fmt.Errorf("no topic:%s", topic)
	}
	sm.Unlock()

	topicSubscribers := subscribers.getAllSubscriber()
	n := 0
	for _, s := range topicSubscribers {
		if s == from {
			continue
		}
		n++
		s.MailBoxMsg(topic, data)
	}
	return n, nil
}

func (sm *SubscriberMgr) GetSubscribers(topic string) []Subscriber {
	sm.Lock()
	subscribers, ok := sm.subscribers[topic]
	if !ok {
		sm.Unlock()
		return nil
	}
	sm.Unlock()

	return subscribers.getAllSubscriber()
}
