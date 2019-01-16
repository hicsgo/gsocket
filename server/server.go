package server

import (
	"net"
	"fmt"
	"log"
	"gsocket/protocol"
	"encoding/json"
	"gsocket"
	"time"
	"errors"
)

var (
	cusProtocol protocol.IProtocol
)

//与服务器相关的资源都放在这里面
type tcpServer struct {
	listener   *net.TCPListener
	hawkServer *net.TCPAddr
	userConns  map[string][]protocol.IProtocol
}

func Run() {

	//类似于初始化套接字，绑定端口
	hawkServer, err := net.ResolveTCPAddr("tcp", "127.0.0.1:8099")
	if err != nil {
		log.Printf("tcp resolve err:%v", err)
	}
	//侦听
	listen, err := net.ListenTCP("tcp", hawkServer)
	if err != nil {
		log.Printf("tcp listen err:%v", err)
	}

	//关闭
	defer listen.Close()

	tcpServer := &tcpServer{
		listener:   listen,
		hawkServer: hawkServer,
		userConns:  make(map[string][]protocol.IProtocol, 0),
	}

	log.Print("start server successful......")

	//开始接收请求
	for {
		//这里是阻塞的每个conn会执行一次
		conn, err := tcpServer.listener.Accept()
		if err != nil {
			log.Printf("tcp accept err:%v", err)
		}
		fmt.Println("accept tcp client %s", conn.RemoteAddr().String())

		cusProtocol = protocol.NewCustomerProtocol()

		//设置当前实例连接
		cusProtocol.SetConn(conn)

		//执行业务逻辑处理(接收所有连接传回来的value)
		go tcpServer.receiveHandler(cusProtocol)

		//发送请求(心跳)
		go tcpServer.heartbeat(cusProtocol)

	}
}

func (tcpServer *tcpServer) receiveHandler(cusProtocol protocol.IProtocol) {
	for {
		result, isOk := <-cusProtocol.GetUnPackResult()
		if !isOk {
			log.Print("get unpack result chan is closed")
			break
		}
		jsonResult := &gsocket.Message{}
		err := json.Unmarshal(result, jsonResult)
		if err != nil {
			log.Printf("json.unmarshal err:%v", err)
		}
		cusProtocol.SetConnOwner(jsonResult.Payload.UserId)
		//设置用户连接(根据连接)
		if jsonResult.Type == "heart_beat" {
			tcpServer.SetUserConn(jsonResult.Payload.UserId, cusProtocol)
		}

		//服务器转发请求
		if jsonResult.Type == "send_msg" {
			tcpServer.SenMsgToUser(jsonResult.Payload.TargetId, jsonResult)
		}

		jsonResultByte, _ := json.Marshal(jsonResult)
		fmt.Println("ggggggggggggggggggggggggg I'm server I get result:", string(jsonResultByte))

	}
}

func (tcpServer *tcpServer) heartbeat(cusProtocol protocol.IProtocol) {
	heartBeatTick := time.Tick(1 * time.Second)
	for {
		if cusProtocol.GetConnOwner() != "" {
			<-heartBeatTick
			wResult := gsocket.Message{
				Payload: &gsocket.MessagePayload{
					UserId:   "server_send",
					TargetId: "server_send",
					Code:     "server_send",
					Extend:   "server_send",
				},
				Type:      "server_heart_beat",
				Timestamp: time.Now().UnixNano(),
			}
			jsonResult, _ := json.Marshal(wResult)

			cusProtocol.SetPackInput(jsonResult)

			fmt.Println(" I'm server I send heartbeat:", string(jsonResult))
			//发送数据
			_, err := cusProtocol.Send()
			if err != nil {
				fmt.Println("client conn is closed", err)
				tcpServer.RemoveUserConn(cusProtocol.GetConnOwner(), cusProtocol)
				break
			}
		}
	}
}

func (tcpServer *tcpServer) SetUserConn(userId string, cusProtocol protocol.IProtocol) {
	conns, isOk := tcpServer.userConns[userId]
	if !isOk {
		userConns := make([]protocol.IProtocol, 0)
		userConns = append(userConns, cusProtocol)
		tcpServer.userConns[userId] = userConns
	} else {
		isExist := false
		for _, exist_conn := range conns {
			if exist_conn == cusProtocol {
				isExist = true
				break
			}
			if !isExist {
				tcpServer.userConns[userId] = append(tcpServer.userConns[userId], cusProtocol)
			}
		}
	}

}

func (tcpServer *tcpServer) GetUserConns(userId string) []protocol.IProtocol {
	conns, isOk := tcpServer.userConns[userId]
	if !isOk {
		return nil
	} else {
		return conns
	}
}

func (tcpServer *tcpServer) RemoveUserConn(userId string, cusProtocol protocol.IProtocol) (bool, error) {
	conns, isOk := tcpServer.userConns[userId]
	if isOk {
		for i, exist_conn := range conns {
			if exist_conn == cusProtocol {
				tcpServer.userConns[userId] = append(tcpServer.userConns[userId], tcpServer.userConns[userId][:i]...)
				tcpServer.userConns[userId] = append(tcpServer.userConns[userId], tcpServer.userConns[userId][i+1:]...)
			}
		}
	}
	return true, nil
}

func (tcpServer *tcpServer) SenMsgToUser(userId string, msg *gsocket.Message) (bool, error) {
	jsonMsgBytes, _ := json.Marshal(msg)
	conns, isOk := tcpServer.userConns[userId]
	if !isOk {
		return false, errors.New("没有获取到指定连接")
	} else {
		for _, cusProtocol := range conns {
			cusProtocol.SetPackInput(jsonMsgBytes)
			cusProtocol.Send()
		}
	}
	return true, nil
}
