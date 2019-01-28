package gsocket

import (
	"github.com/gorilla/websocket"
	"github.com/satori/go.uuid"
)

import (
	"sync"
	"net/url"
	"encoding/json"
	"time"
	"fmt"
	"log"
	"net/http"
	"math/rand"
	"flag"
)

var registerRemoteAddr = flag.String("register", "192.168.1.1", "http register address")

func init() {
	//以时间作为初始化种子
	rand.Seed(time.Now().UnixNano())
}

/* ++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++
 * 连接标记
 * ++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++ */
type connFlag struct {
	Id   string //"uuid"
	Tag  string //client_to_gateway,worker_to_gateway,gateway_to_register
	Conn *websocket.Conn
}

/* ++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++
 * 连接映射
 * ++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++ */
type connMap struct {
	MapKey string
	Tag    string //client_to_gateway,worker_to_gateway,gateway_to_register
}

/* ++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++
 * 通道消息
 * ++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++ */
type ChanMsg struct {
	mt     int
	msg    *Message
	conn   *websocket.Conn
	connId string
}

/* ++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++
 * 通道消息
 * ++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++ */
type base struct {
	gatewayConns          sync.Map      //其他网关连接集合  map["192.168.1.1:12345"]=connFlag
	clientConns           sync.Map      //客户端连接集合   map["user_id"]=[]connFlag
	workerConns           sync.Map      //服务端连接集合   map["192.168.3.7:8866"]=connFlag{Id:"cae015bb-22a8-11e9-9bdd-00163e12d392",Tag:"web"}
	connMapping           sync.Map      //连接和其他map的映射关系 map["cae015bb-22a8-11e9-9bdd-00163e12d392"]=connMap{"user_id":"1",Tag:"client"}
	msgChan               chan *ChanMsg //消息
	stopChan              chan bool     //终止
	isConnectedToRegister bool          //是否连接到注册中心
}

/* ++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++
 * 连接其他服务器
 * ++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++ */
func (base *base) dial(addr, tag string) {
	u1, err := uuid.NewV4()
	if err != nil {
		log.Println("new uuid err:", err)
		return
	}

	u := url.URL{Scheme: "ws", Host: addr, Path: "/"}
	log.Printf("connecting to %s", u.String())

	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		log.Fatal("dial:", err)
		return
	}
	defer c.Close()

	//连接成功,记录连接
	switch tag {
	case TAG_GATEWAY_TO_GATEWAY:
		{
			if _, ok := base.connMapping.Load(u1); !ok {
				base.connMapping.Store(u1, connMap{
					MapKey: c.RemoteAddr().String(),
					Tag:    TAG_GATEWAY_TO_GATEWAY,
				})
			}
			if _, ok := base.gatewayConns.Load(c.RemoteAddr().String()); !ok {
				base.gatewayConns.Store(c.RemoteAddr().String(), connFlag{
					Id:   u1.String(),
					Tag:  TAG_GATEWAY_TO_GATEWAY,
					Conn: c,
				})
			}
		}
	case CONN_TO_REGISTER:
		{
			base.isConnectedToRegister = true
			if _, ok := base.connMapping.Load(u1); !ok {
				base.connMapping.Store(u1, connMap{
					MapKey: c.RemoteAddr().String(),
					Tag:    CONN_TO_REGISTER,
				})
			}
		}
	case TAG_WORKER_TO_GATEWAY:
		{
			//处理器连接到网关
		}
	case TAG_CLIENT_TO_GATEWAY:
		{
			//客户端连接到网关
		}
	}

	go func() {
		for {
			jsonResult := &Message{}
			mt, message, err := c.ReadMessage()
			if err != nil {
				log.Println("read:", err)
				jsonResult.Type = CONN_CLOSE
				base.msgChan <- &ChanMsg{
					mt:     mt,
					msg:    jsonResult,
					conn:   c,
					connId: u1.String(),
				}
				return
			}

			json.Unmarshal(message, jsonResult)

			base.msgChan <- &ChanMsg{
				mt:     mt,
				msg:    jsonResult,
				conn:   c,
				connId: u1.String(),
			}

			log.Printf("recv: %s", message)

		}
	}()

	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C: //心跳
			msg := &Message{}
			msg.Type = HEART_BEAT
			msgJson, _ := json.Marshal(msg)
			err := c.WriteMessage(websocket.TextMessage, msgJson)
			if err != nil {
				log.Println("write:", err)
				return
			}

		}
	}
}

/* ++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++
 * 监听服务
 * ++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++ */
func (base *base) server(w http.ResponseWriter, r *http.Request) {
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Print("upgrade:", err)
		return
	}
	defer c.Close()
	u1, err := uuid.NewV4()
	if err != nil {
		log.Println("new uuid err:", err)
		return
	}

	for {
		jsonResult := &Message{}
		mt, message, err := c.ReadMessage()
		if err != nil {
			log.Println("read:", err)
			jsonResult.Type = CONN_CLOSE
			base.msgChan <- &ChanMsg{
				mt:     mt,
				msg:    jsonResult,
				conn:   c,
				connId: u1.String(),
			}
			break
		}

		json.Unmarshal(message, jsonResult)

		base.msgChan <- &ChanMsg{
			mt:     mt,
			msg:    jsonResult,
			conn:   c,
			connId: u1.String(),
		}

		log.Printf("recv: %s", message)

	}
}

/* ++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++
 * 消费消息
 * ++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++ */
func (base *base) consumerMsg() {
	for {
		result, isOk := <-base.msgChan
		if !isOk {
			log.Print("get unpack msgChan is closed")
			break
		}
		if result != nil {
			switch result.msg.Type {
			case CONN_CLOSE:
				go base.offlineConn(result)
				fmt.Println("连接关闭")
			case WORKER_MSG:
				fmt.Println("register_gateway_offline")
			default:
				fmt.Println("unkown type code")
			}
			fmt.Println("ggggggggggggggggggg I'm client I get msg:")
		}

	}
}

/* ++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++
 * 连接断开下线
 * ++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++ */
func (base *base) offlineConn(message *ChanMsg) {
	if message != nil {
		if v, ok := base.connMapping.Load(message.connId); ok {

			if connmap, ok := v.(connMap); ok {

				switch connmap.Tag {
				case TAG_CLIENT_TO_GATEWAY:
					{
						if value, ok := base.clientConns.Load(connmap.MapKey); ok {
							//用户的所有连接
							if userConns, ok := value.([]connFlag); ok {
								for i := 0; i < len(userConns); i++ {
									//移除指定关闭的连接
									if userConns[i].Id == message.connId {
										userConns = append(userConns[:i], userConns[i+1:]...)
										base.clientConns.Store(connmap.MapKey, userConns)
										break
									}
								}
							}
						}

					}
				case TAG_WORKER_TO_GATEWAY:
					{
						if _, ok := base.workerConns.Load(connmap.MapKey); ok {
							//移除断开的连接
							base.workerConns.Delete(connmap.MapKey)
						}

					}
				case TAG_GATEWAY_TO_GATEWAY:
					{
						if _, ok := base.gatewayConns.Load(connmap.MapKey); ok {
							//移除断开的连接
							base.gatewayConns.Delete(connmap.MapKey)
						}
					}
				case CONN_TO_REGISTER:
					{
						base.isConnectedToRegister = false
					}
				}

				base.connMapping.Delete(message.connId)
			}
		}
	}
}

/* ++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++
 * 发送消息给服务端处理器
 * ++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++ */
func (base *base) sendToWorkerHandler(message *ChanMsg) {
	//从服务端连接池中随机拿一个连接
	// (因为map无序性直接取第一个)
	msgJson, err := json.Marshal(message.msg)
	if err != nil {
		return
	}
	base.workerConns.Range(func(key, value interface{}) bool {
		if connflag, ok := value.(connFlag); ok {
			err = connflag.Conn.WriteMessage(message.mt, msgJson)
			if err != nil {
				//发送失败

				//移除映射
				base.connMapping.Delete(connflag.Id)

				//移除连接
				base.workerConns.Delete(key)

				return true //由后面的worker继续发送直到重试所有worker
			}

		}
		return false //第一个发送成功就终止
	})
}

/* ++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++
 * 转发消息给其他网关处理器
 * ++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++ */
func (base *base) sendToOtherGatewaysHandler(message *ChanMsg) {

	//广播其他网关地址
	base.gatewayConns.Range(func(k, v interface{}) bool {

		if connflag, ok := v.(connFlag); ok {

			msg, err := json.Marshal(message.msg)
			if err != nil {
				log.Println("send to other gateway json marsharl err:", err)
			} else {
				err := connflag.Conn.WriteMessage(message.mt, msg)

				if err != nil {
					log.Println("write:", err)
					//移除当前网关
					base.gatewayConns.Delete(k)
					//移除映射关系
					base.connMapping.Delete(connflag.Id)

				}
			}
		}

		return true
	})
}

/* ++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++
 * 发送消息给客户端
 * ++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++ */
func (base *base) sendToClient(message *ChanMsg) {
	if message != nil {
		if message.msg.Payload != nil {
			if message.msg.Payload.TargetId != "" {
				//发送给指定用户
				if conns, ok := base.clientConns.Load(message.msg.Payload.TargetId); ok {
					if clientConns, ok := conns.([]connFlag); ok {
						for i, clientConn := range clientConns { //多个终端都可以同时收到消息

							//此处直接调用ws，不使用自定义封包协议
							msg, err := json.Marshal(message)
							if err != nil {
								log.Println("send to client json marsharl err:", err)
							} else {
								err := clientConn.Conn.WriteMessage(message.mt, msg)
								if err != nil {
									//移除当前连接
									clientConns = append(clientConns[:i], clientConns[i+1:]...)
									base.clientConns.Store(message.msg.Payload.TargetId, clientConns)
								}
							}
						}
					}
				}
			} else {
				//广播所有用户
				base.clientConns.Range(func(k, v interface{}) bool {

					if clientConns, ok := v.([]connFlag); ok {
						for i, clientConn := range clientConns { //多个终端都可以同时收到消息

							//此处直接调用ws，不使用自定义封包协议
							msg, err := json.Marshal(message)
							if err != nil {
								log.Println("send to client json marsharl err:", err)
							} else {
								err := clientConn.Conn.WriteMessage(message.mt, msg)
								if err != nil {
									//移除当前连接
									clientConns = append(clientConns[:i], clientConns[i+1:]...)
									base.clientConns.Store(message.msg.Payload.TargetId, clientConns)
								}
							}
						}
					}
					return true
				})

			}
		}
	}
}

/* ++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++
 * 监测是否连接到注册中心，失败自动重连
 * ++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++ */
func (base *base) watchConnToRegister() {
	ticker := time.NewTicker(time.Minute * 2)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C: //监测
			if !base.isConnectedToRegister {
				go base.dial(*registerRemoteAddr, CONN_TO_REGISTER)
			}
		}
	}
}
