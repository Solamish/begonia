package dispatch

import (
	"fmt"
	"github.com/MashiroC/begonia/config"
	"github.com/MashiroC/begonia/dispatch/conn"
	"github.com/MashiroC/begonia/dispatch/frame"
	"github.com/MashiroC/begonia/tool/ids"
	"log"
	"time"
)

// dispatch_default.go something

// NewByDefaultCluster 在default cluster模式下创建一个dispatch
func NewLinkedByDefaultCluster() Dispatcher {

	d := &linkDispatch{}

	// 判断是否需要在断开连接情况下重连，hook了dispatch层的close函数
	if !config.C.Dispatch.AutoReConnection {
		// 不配置自动重连时 默认连接被关闭时panic
		d.Hook("close", func(connID string, err error) {
			panic("conn close")
		})
	} else {

		d.Hook("close", func(connID string, err error) {

			// 用一个协程跑 避免阻塞
			go func() {
				ok := false

				if config.C.Dispatch.ReConnectionRetryCount <= 0 {

					for !ok {
						log.Println("cannot link to server,retry...")
						time.Sleep(time.Duration(config.C.Dispatch.ReConnectionIntervalSecond) * time.Second)
						ok = d.ReLink()
					}

				} else {

					for i := 0; i < config.C.Dispatch.ReConnectionRetryCount && !ok; i++ {
						log.Println("cannot link to server,retry", i, "limit", config.C.Dispatch.ReConnectionRetryCount)
						time.Sleep(time.Duration(config.C.Dispatch.ReConnectionIntervalSecond) * time.Second)
						ok = d.ReLink()
					}

					if !ok {
						panic("connect closed")
					}

				}
			}()

		})

	}

	return d
}

type linkDispatch struct {
	baseDispatch

	// link模式相关变量
	linkAddr   string    // 单连接的地址
	linkedConn conn.Conn // 连接
	linkID     string    // 连接的id
}

// Link 建立连接，bgacenter cluster模式下，会开一条和center的tcp连接
func (d *linkDispatch) Link(addr string) (err error) {

	d.linkAddr = addr

	c, err := conn.Dial(addr)
	if err != nil {
		return fmt.Errorf("dispatch link error: %w", err)
	}

	d.linkedConn = c

	go d.work(c)

	return
}

func (d *linkDispatch) ReLink() bool {
	err := d.Link(d.linkAddr)
	return err == nil
}

// Send 发送一个包，在center cluster模式下直接发送到中心，中心进行调度
func (d *linkDispatch) Send(f frame.Frame) (err error) {
	// TODO:请求实现幂等 断连时排序等待连接重连 这里暂时先直接传过去
	err = d.linkedConn.Write(byte(f.Opcode()), f.Marshal())
	return
}

func (d *linkDispatch) SendTo(connID string, f frame.Frame) (err error) {
	if connID != d.linkID {
		err = fmt.Errorf("dispatch send error: in linked mode, you can't use SendTo() to another conn, please use Send() or passing manager bgacenter connID")
		return
	}

	err = d.linkedConn.Write(byte(f.Opcode()), f.Marshal())
	return
}

func (d *linkDispatch) Listen(addr string) {
	panic("link mode can't use Listen()")
}

// work 获得一个新的连接之后持续监听连接，然后把消息发送到msgCh里
func (d *linkDispatch) work(c conn.Conn) {

	id := ids.New()

	d.linkID = id
	log.Printf("link addr [%s] success, connID [%s]\n", c.Addr(), id)

	for {

		opcode, payload, err := c.Recv()
		if err != nil {
			c.Close()
			d.DoCloseHook(id, err)
			break
		}

		d.rt.Do(id, opcode, payload)
	}

}

func (d *linkDispatch) Close() {
	d.linkedConn.Close()
}
