package protocol

import "net"

/* ++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++
 * 装解包协议
 * ++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++ */
type IProtocol interface {
	pack()
	unPack()
	Send() (n int, err error)
	SetConn(conn net.Conn)
	GetConn() (conn net.Conn)
	GetUnPackResult() (chan []byte)
	SetPackInput([]byte)
	SetConnOwner(connOwner string)
	GetConnOwner() string
}
