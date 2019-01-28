package gsocket

import (
	"sync"
)

/* ================================================================================
 *  处理器中心模块
 *  author:golang123@outlook.com
 * ================================================================================ */

type worker struct {
	base
}

/* ++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++
 * 处理中心启动
 * ++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++ */
func WorkerRun() {

	//实例化网关
	g := &worker{
		base: base{
			gatewayConns: sync.Map{},
			clientConns:  sync.Map{},
			workerConns:  sync.Map{},
			stopChan:     make(chan bool, 0),
			msgChan:      make(chan *ChanMsg, 0),
		},
	}
	//连接注册中心
	go g.dial(*registerRemoteAddr, CONN_TO_REGISTER) //注册服务

	<-g.stopChan
}

func (worker *worker) workerHandler() {
	for {
		<-worker.msgChan
		//实际业务处理
	}
}
