package gsocket

import (
	"github.com/gorilla/websocket"
)

var registerRemoteAddr = "127.0.0.1:8080"

/* ++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++
 * 连接标记
 * ++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++ */
type ConnFlag struct {
	ConnId       string //"uuid"
	TerminalName string //web/H5/android/ios/ipad/server
	Conn         *websocket.Conn
}

/* ++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++
 * 连接映射
 * ++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++ */
type ConnMap struct {
	MapKey string
	Tag    string //client_conn_to_gateway,worker_conn_to_gateway,gateway_conn_to_gateway,gateway_conn_to_register,worker_conn_to_register
}

/* ++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++
 * 通道消息
 * ++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++ */
type ChanMsg struct {
	msg    *Message
	conn   *websocket.Conn
	connId string
}
