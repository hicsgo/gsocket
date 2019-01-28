package gsocket

import "flag"

import (
	"github.com/gorilla/websocket"
)

import (
	"sync"
	"log"
	"net/http"
)

/* ================================================================================
 *  网关中心模块
 *  author:golang123@outlook.com
 * ================================================================================ */

var local_gateway_addr = flag.String("addr", "localhost:8080", "http service address")

var upgrader = websocket.Upgrader{} // use default options

type gateway struct {
	base
}

/* ++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++
 * 启动网关
 * ++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++ */
func GatewayRun() {

	//实例化网关
	g := &gateway{
		base: base{
			gatewayConns: sync.Map{},
			clientConns:  sync.Map{},
			workerConns:  sync.Map{},
			stopChan:     make(chan bool, 0),
			msgChan:      make(chan *ChanMsg, 0),
		},
	}
	go func() {
		http.HandleFunc("/server", g.server)
		log.Fatal(http.ListenAndServe(*local_gateway_addr, nil))
	}()                                        //启动监听
	go g.dial(*registerRemoteAddr, "register") //注册服务

	<-g.stopChan
}
