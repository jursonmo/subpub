package message

const (
	SubscribeOp   byte = 0
	UnsubscribeOp byte = 1
)

type Topic string
type SubMsg struct {
	Topic Topic
	//Encode string //没用因为在unmarshal 之前，根本不知道Encode的值是啥
	Op   byte //0:subscribe, 1 unsubscribe
	Body []byte
}

type PubMsg struct {
	Topic Topic
	Body  []byte
}

type TopicMsgHandler func(string, []byte)
