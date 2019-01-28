package gsocket

import (
	"flag"
	"net/http"
	"log"
	"sync"
)

/* ================================================================================
 *  注册中心模块
 *  author:golang123@outlook.com
 * ================================================================================ */

var local_register_addr = flag.String("addr", "localhost:8080", "http service address")

type register struct {
	base
}

/* ++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++
 * 启动注册中心
 * ++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++ */
func RegisterRun() {
	//实例化注册中心
	r := &register{
		base: base{
			gatewayConns: sync.Map{},
			clientConns:  sync.Map{},
			workerConns:  sync.Map{},
			stopChan:     make(chan bool, 0),
			msgChan:      make(chan *ChanMsg, 0),
		},
	}
	go func() {
		http.HandleFunc("/server", r.server)
		log.Fatal(http.ListenAndServe(*local_gateway_addr, nil))
	}() //启动监听

	<-r.stopChan
}
