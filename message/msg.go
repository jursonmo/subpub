package message

type Topic string
type SubMsg struct {
	Topic Topic
	//Encode string //没用因为在unmarshal 之前，根本不知道Encode的值是啥
	Body []byte
}

type PubMsg struct {
	Topic Topic
	Body  []byte
}
