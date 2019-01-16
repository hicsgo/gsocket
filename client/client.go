package client

import (
	"net"
	"fmt"
	"os"
	"time"
	"log"
	"gsocket/protocol"
	"gsocket"
	"encoding/json"
)

//客户端对象
type tcpClient struct {
	connection *net.TCPConn
	hawkServer *net.TCPAddr
	stopChan   chan struct{}
}

func ClientASend() {
	//拿到服务器地址信息
	hawkServer, err := net.ResolveTCPAddr("tcp", "127.0.0.1:8099")
	if err != nil {
		log.Printf("hawk server [%s] resolve error: [%s]", "127.0.0.1:8099", err.Error())
		os.Exit(1)
	}

	//连接服务器
	connection, err := net.DialTCP("tcp", nil, hawkServer)
	if err != nil {
		fmt.Printf("connect to hawk server error: [%s]", err.Error())
		os.Exit(1)
	}
	client := &tcpClient{
		connection: connection,
		hawkServer: hawkServer,
		stopChan:   make(chan struct{}),
	}

	//启动接收
	cusProtocol := protocol.NewCustomerProtocol()

	//设置当前实例连接
	cusProtocol.SetConn(client.connection)

	//执行业务逻辑处理
	go client.receiveHandler(cusProtocol)

	//测试用的，开300个goroutine每两秒发送一个包
	for i := 0; i < 1; i++ {
		go func() {
			sendTimer := time.Tick(2 * time.Second)
			for {
				select {
				case <-sendTimer:
					client.sendMsgA(cusProtocol)
				case <-client.stopChan:
					return
				}
			}
		}()
	}

	//心跳(每秒发送一个心跳包)
	go func() {
		sendTimer := time.Tick(1 * time.Second)
		for {
			select {
			case <-sendTimer:
				client.heartbeatA(cusProtocol)
			case <-client.stopChan:
				return
			}
		}
	}()
	//等待退出
	<-client.stopChan
}

func ClientBRecive() {
	//拿到服务器地址信息
	hawkServer, err := net.ResolveTCPAddr("tcp", "127.0.0.1:8099")
	if err != nil {
		log.Printf("hawk server [%s] resolve error: [%s]", "127.0.0.1:8099", err.Error())
		os.Exit(1)
	}

	//连接服务器
	connection, err := net.DialTCP("tcp", nil, hawkServer)
	if err != nil {
		fmt.Printf("connect to hawk server error: [%s]", err.Error())
		os.Exit(1)
	}
	client := &tcpClient{
		connection: connection,
		hawkServer: hawkServer,
		stopChan:   make(chan struct{}),
	}

	//启动接收
	cusProtocol := protocol.NewCustomerProtocol()

	//设置当前实例连接
	cusProtocol.SetConn(client.connection)

	//执行业务逻辑处理
	go client.receiveHandler(cusProtocol)

	//心跳(每秒发送一个心跳包)
	go func() {
		sendTimer := time.Tick(1 * time.Second)
		for {
			select {
			case <-sendTimer:
				client.heartbeatB(cusProtocol)
			case <-client.stopChan:
				return
			}
		}
	}()

	//等待退出
	<-client.stopChan
}

func (tcpClient *tcpClient) sendMsgA(customProtocol protocol.IProtocol) {
	wResult := gsocket.Message{
		Payload: &gsocket.MessagePayload{
			UserId:   "1",
			TargetId: "2",
			Code:     "client_send",
			Extend:   "client_send",
		},
		Type:      "send_msg",
		Timestamp: time.Now().UnixNano(),
	}
	jsonResult, _ := json.Marshal(wResult)

	customProtocol.SetPackInput(jsonResult)
	fmt.Println(" i'm client a I send msg :", string(jsonResult))

	//发送数据
	_, err := customProtocol.Send()
	if err != nil {
		fmt.Println("server is closed err:", err)
		close(tcpClient.stopChan)
	}
}

//client a heartbeat to server
func (tcpClient *tcpClient) heartbeatA(customProtocol protocol.IProtocol) {
	wResult := gsocket.Message{
		Payload: &gsocket.MessagePayload{
			UserId:   "1",
			TargetId: "-",
			Code:     "-",
			Extend:   "-",
		},
		Type:      "heart_beat",
		Timestamp: time.Now().UnixNano(),
	}
	jsonResult, _ := json.Marshal(wResult)

	customProtocol.SetPackInput(jsonResult)
	fmt.Println(" i'm client a I send heartbeat msg :", string(jsonResult))

	//发送数据
	_, err := customProtocol.Send()
	if err != nil {
		fmt.Println("server is closed err:", err)
		close(tcpClient.stopChan)
	}
}

//client b heartbeat to server
func (tcpClient *tcpClient) heartbeatB(customProtocol protocol.IProtocol) {
	wResult := gsocket.Message{
		Payload: &gsocket.MessagePayload{
			UserId:   "2",
			TargetId: "-",
			Code:     "-",
			Extend:   "-",
		},
		Type:      "heart_beat",
		Timestamp: time.Now().UnixNano(),
	}
	jsonResult, _ := json.Marshal(wResult)

	customProtocol.SetPackInput(jsonResult)
	fmt.Println(" i'm client a I send heartbeat msg :", string(jsonResult))

	//发送数据
	_, err := customProtocol.Send()
	if err != nil {
		fmt.Println("server is closed err:", err)
		close(tcpClient.stopChan)
	}
}

//client receive msg
func (tcpClient *tcpClient) receiveHandler(cusProtocol protocol.IProtocol) {
	for {
		result, isOk := <-cusProtocol.GetUnPackResult()
		if !isOk {
			log.Print("get unpack result chan is closed")
			close(tcpClient.stopChan)
			break
		}
		jsonResult := &gsocket.Message{}
		err := json.Unmarshal(result, jsonResult)
		if err != nil {
			log.Printf("json.unmarshal err:%v", err)
		}
		jsonResultByte, _ := json.Marshal(jsonResult)
		fmt.Println("ggggggggggggggggggg I'm client I get msg:", string(jsonResultByte))
	}
}
