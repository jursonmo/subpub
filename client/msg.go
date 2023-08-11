package client

type Msg struct {
	d    []byte
	done chan struct{}
	err  error
}

func NewMsg(d []byte) *Msg {
	msg := new(Msg) //todo, sync.Pool
	msg.d = d
	msg.err = nil
	msg.done = make(chan struct{})
	return msg
}

func (m *Msg) Complete(err error) {
	if m.done != nil {
		m.err = err
		close(m.done)
	}
}

func (m *Msg) Wait() error {
	defer m.Release()
	if m.done != nil {
		<-m.done
	}
	return m.err
}

func (m *Msg) Release() {
	//todo: reset msg and put back to sync.Pool
}
