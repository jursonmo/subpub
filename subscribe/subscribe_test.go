package subscribe

import (
	"fmt"
	"reflect"
	"testing"
	"time"
)

func sliceContains(ss []string, s string) bool {
	for _, v := range ss {
		if v == s {
			return true
		}
	}
	return false
}

var Topic1 = "topic1-test"
var Topic2 = "topic2-test"
var Topic3 = "topic3-test"

func TestBaseFunc(t *testing.T) {
	sm := NewSubscriberMgr()
	subID1 := "subID-1"
	sub1ReceiveMsg := ""
	sub1ReceiveTopic := ""
	sub1 := sm.NewSubscriber(SubscriberID(subID1), func(topic string, b []byte) error {
		t.Logf("%s receive topic:%s, msg:%s", subID1, topic, string(b))
		sub1ReceiveTopic = topic
		sub1ReceiveMsg = string(b)
		return nil
	})

	subID2 := "subID-2"
	sub2ReceiveMsg := ""
	sub2ReceiveTopic := ""
	sub2 := sm.NewSubscriber(SubscriberID(subID2), func(topic string, b []byte) error {
		t.Logf("%s receive topic:%s, msg:%s", subID2, topic, string(b))
		sub2ReceiveTopic = topic
		sub2ReceiveMsg = string(b)
		return nil
	})

	//1. 测试订阅topic
	sub1.Subscribe(Topic1)
	if !reflect.DeepEqual(sub1.Topics(), []string{Topic1}) {
		t.Fatalf("sub1.Topics():%v, topics:%v", sub1.Topics(), []string{Topic1})
	}

	//2. 测试sub2 publish 消息给 sub1
	sub2PublishMsg := "message from sub2"
	sub2.Publish(Topic1, []byte(sub2PublishMsg)) //公告消息， sub1 会收到消息，并保存在sub1ReceiveMsg

	time.Sleep(time.Millisecond * 20)

	if sub1ReceiveTopic != Topic1 {
		t.Fatalf("sub1ReceiveTopic:%s should be Topic1:%s", sub1ReceiveTopic, Topic1)
	}
	if sub1ReceiveMsg != sub2PublishMsg {
		t.Fatalf("sub1ReceiveMsg:%s, sub2PublishMsg:%s", sub1ReceiveMsg, sub2PublishMsg)
	}

	//3. 测试sm publish 消息给 sub1, sub2, 看多个subscriber 能不能收到相同消息
	sub2.Subscribe(Topic1)
	smPublishMsg := "message from sm"
	n, err := sm.Publish(nil, Topic1, []byte(smPublishMsg))
	if err != nil {
		t.Fatal(err)
	}
	//发给两个订阅者
	if n != 2 {
		t.Fatalf("n:%d should be 2,", n)
	}
	time.Sleep(time.Millisecond * 20)
	if sub1ReceiveMsg != smPublishMsg {
		t.Fatalf("sub1ReceiveMsg:%s, smPublishMsg:%s", sub1ReceiveMsg, smPublishMsg)
	}
	if sub2ReceiveMsg != smPublishMsg {
		t.Fatalf("sub2ReceiveMsg:%s, smPublishMsg:%s", sub2ReceiveMsg, smPublishMsg)
	}
	if sub2ReceiveTopic != Topic1 {
		t.Fatalf("sub2ReceiveTopic:%s should be Topic1:%s", sub2ReceiveTopic, Topic1)
	}

	//4. 测试sub1退订后，不再接受到指定的消息, 只有sub2收到消息
	sub1.UnSubscribe(Topic1)
	if sliceContains(sub1.Topics(), Topic1) {
		t.Fatalf("sub1.Topics():%v, shouldn't contains %s", sub1.Topics(), Topic1)
	}

	smPublishMsg2 := "message2 from sm"
	n, err = sm.Publish(nil, Topic1, []byte(smPublishMsg2))
	if err != nil {
		t.Fatal(err)
	}
	//发给订阅者的数量应该是1, 只剩下sub2一个订阅者
	if n != 1 {
		t.Fatalf("n:%d should be 1,", n)
	}

	time.Sleep(time.Millisecond * 20)

	//sub1 还是原来的消息
	if sub1ReceiveMsg != smPublishMsg {
		t.Fatalf("sub1ReceiveMsg:%s, smPublishMsg:%s", sub1ReceiveMsg, smPublishMsg)
	}
	//sub2 收到新的smPublishMsg2消息
	if sub2ReceiveMsg != smPublishMsg2 {
		t.Fatalf("sub2ReceiveMsg:%s, smPublishMsg2:%s", sub2ReceiveMsg, smPublishMsg2)
	}

	//5. sub1再次订阅Topic1, 是否又能接受到消息， 当前应该又有两个订阅者
	sub1.Subscribe(Topic1)
	msg := "message3 from sm"
	n, err = sm.Publish(nil, Topic1, []byte(msg))
	if err != nil {
		t.Fatal(err)
	}
	//发给订阅者的数量应该是2
	if n != 2 {
		t.Fatalf("n:%d should be 2,", n)
	}

	if sub1ReceiveMsg != msg {
		t.Fatalf("sub1ReceiveMsg:%s, smPublishMsg:%s", sub1ReceiveMsg, msg)
	}
	if sub2ReceiveMsg != msg {
		t.Fatalf("sub2ReceiveMsg:%s, smPublishMsg:%s", sub2ReceiveMsg, msg)
	}

	//6.测试订阅者订阅多个主题的情况下，能否接受到不同主题的消息
	type Msg struct {
		topic string
		msg   string
	}
	testMsgs := []Msg{{Topic1, "Topic1 msg"}, {Topic2, "Topic2 msg"}, {Topic3, "Topic3 msg"}}
	subID3 := "subID-3"
	sub3ReceiveMsgs := []Msg{}
	sub3 := sm.NewSubscriber(SubscriberID(subID3), func(topic string, b []byte) error {
		t.Logf("%s receive topic:%s, msg:%s", subID3, topic, string(b))
		sub3ReceiveMsgs = append(sub3ReceiveMsgs, Msg{topic: topic, msg: string(b)})
		return nil
	})
	sub3.Subscribe(Topic1)
	sub3.Subscribe(Topic2)
	sub3.Subscribe(Topic3)
	//公告三个不同topic的消息
	for _, msg := range testMsgs {
		sm.Publish(nil, msg.topic, []byte(msg.msg))
	}
	sm.Publish(nil, "topic-x", []byte("topic-x msg")) //发送一个sub3 没有订阅的topic 的消息
	time.Sleep(time.Millisecond * 20)

	//检查sub3是否完整接受已订阅的主题消息，
	if !reflect.DeepEqual(sub3ReceiveMsgs, testMsgs) {
		t.Fatalf("sub3ReceiveMsgs:%+v, testMsgs:%+v", sub3ReceiveMsgs, testMsgs)
	}

	//7. 测试sub3 Close(), 即退订所有topic, 应该接受不到任何新的消息
	sub3ReceiveMsgs = []Msg{} //重置
	sub3.Close()
	if len(sub3.Topics()) != 0 {
		t.Fatalf("after close, len(sub3.Topics()):%d should be 0", len(sub3.Topics()))
	}

	for _, msg := range testMsgs {
		sm.Publish(nil, msg.topic, []byte(msg.msg))
	}
	time.Sleep(time.Millisecond * 20)
	if len(sub3ReceiveMsgs) != 0 {
		t.Fatalf("sub3ReceiveMsgs:%+v, should empty", sub3ReceiveMsgs)
	}

}

func TestAddSubscribeCheck(t *testing.T) {
	subID1 := SubscriberID("subID-1")
	subID2 := SubscriberID("subID-2")

	subID1InvaildTopic := "invaild-topic"
	vaildTopic := "vaild-topic"

	//订阅者订阅主题时的回调函数
	check := func(from Subscriber, topic string) error {
		//subID1 不允许订阅subID1InvaildTopic 主题
		if from.Id() == subID1 && topic == subID1InvaildTopic {
			return fmt.Errorf("%s is not allowd to subscribe this topic", from.Id())
		}
		return nil
	}

	sm := NewSubscriberMgr(WithSubscribeCheck(check))
	sub1 := sm.NewSubscriber(subID1, func(topic string, d []byte) error { return nil })
	sub2 := sm.NewSubscriber(subID2, func(topic string, d []byte) error { return nil })

	err := sub1.Subscribe(vaildTopic)
	if err != nil {
		t.Fatal(err)
	}

	//sub1订阅 subID1InvaildTopic，应该不成功
	err = sub1.Subscribe(subID1InvaildTopic)
	if err == nil {
		t.Fatalf("%s shouldn't Subscribe subID1InvaildTopic:%s", sub1.Id(), subID1InvaildTopic)
	}

	//sub2 应该可以订阅 subID1InvaildTopic，
	err = sub2.Subscribe(subID1InvaildTopic)
	if err != nil {
		t.Fatalf("%s should Subscribe subID1InvaildTopic:%s", sub2.Id(), subID1InvaildTopic)
	}
}

func TestPublishCheck(t *testing.T) {
	subID1 := SubscriberID("subID-1")
	subID2 := SubscriberID("subID-2")

	subID1InvaildTopic := "invaild-topic"
	vaildTopic := "vaild-topic"

	publishCheck := func(from Subscriber, topic string, data []byte) error {
		//不允许 subID1 发布 subID1InvaildTopic 主题的消息
		if from.Id() == subID1 && topic == subID1InvaildTopic {
			return fmt.Errorf("%s is not allowd to publish this topic", from.Id())
		}
		return nil
	}

	sm := NewSubscriberMgr(WithPublishCheck(publishCheck))
	sub1 := sm.NewSubscriber(subID1, func(topic string, d []byte) error { return nil })
	sub2ReceiveMsg := ""
	sub2 := sm.NewSubscriber(subID2, func(topic string, d []byte) error { sub2ReceiveMsg = string(d); return nil })

	//sub2 订阅了所有主题
	err := sub2.Subscribe(subID1InvaildTopic)
	if err != nil {
		t.Fatal(err)
	}
	err = sub2.Subscribe(vaildTopic)
	if err != nil {
		t.Fatal(err)
	}

	//sub1 发布subID1InvaildTopic 主题的消息，应该不成功
	err = sub1.Publish(subID1InvaildTopic, []byte("subID1InvaildTopic msg"))
	if err == nil {
		t.Fatalf("%s shouldn't allowed to publish %s msg", sub1.Id(), subID1InvaildTopic)
	}

	//sub1 发布 vaildTopic 主题的消息，应该成功
	msg := "vaildTopic msg"
	err = sub1.Publish(vaildTopic, []byte(msg))
	if err != nil {
		t.Fatalf("%s should allowed to publish %s msg", sub1.Id(), subID1InvaildTopic)
	}
	time.Sleep(time.Millisecond * 10)
	//sub2 应该能收到 sub1 公告的msg
	if sub2ReceiveMsg != msg {
		t.Fatalf("sub2ReceiveMsg:%s should be same as  msg:%s ", sub2ReceiveMsg, msg)
	}
}
