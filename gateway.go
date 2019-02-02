package gsocket

import (
	"github.com/gorilla/websocket"
)

import (
	"sync"
	"log"
	"net/http"
	"github.com/satori/go.uuid"
	"encoding/json"
	"net/url"
	"fmt"
	"time"
)

/* ================================================================================
 *  网关中心模块
 *  author:golang123@outlook.com
 * ================================================================================ */

var local_gateway_addr = "10.1.10.70:8099" //需要能被处理器连接到(最好都处在一个内网环境,使用内网ip)

var upgrader = websocket.Upgrader{} // use default options

type gateway struct {
	clientConns           sync.Map      //客户端连接到本地网关连接集合            map["user_id"]=[]connFlag
	workerConns           sync.Map      //多个处理器连接到本地网关连接集合         map["192.168.3.7:8866"]=connFlag{Id:"cae015bb-22a8-11e9-9bdd-00163e12d392",Tag:"web"}
	connMapping           sync.Map      //连接和其他map的映射关系                map["cae015bb-22a8-11e9-9bdd-00163e12d392"]=connMap
	msgChan               chan *ChanMsg //消息
	stopChan              chan bool     //终止
	isConnectedToRegister bool          //是否连接到注册中心
}

/* ++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++
 * 启动网关
 * ++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++ */
func GatewayRun() {

	//实例化网关
	g := &gateway{
		clientConns:           sync.Map{},
		workerConns:           sync.Map{},
		connMapping:           sync.Map{},
		stopChan:              make(chan bool, 0),
		msgChan:               make(chan *ChanMsg, 0),
		isConnectedToRegister: false,
	}
	go func() {
		http.HandleFunc("/", g.server)
		log.Fatal(http.ListenAndServe(local_gateway_addr, nil))
	}() //启动监听

	go g.dialRegister(registerRemoteAddr) //注册服务

	go g.watchConnToRegister() //如果跟注册中心断开自动重连

	go g.consumerMsg() //消费消息

	go func() {
		ticker := time.NewTicker(time.Second * 10)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				fmt.Println()
				fmt.Println("------------------------------gateway sum conn begin-----------------------------------")

				workerConnCount := 0
				g.workerConns.Range(func(key, value interface{}) bool {
					workerConnCount++
					fmt.Println("gateway worker", key, value)
					return true
				})
				clientConnCount := 0
				g.clientConns.Range(func(key, value interface{}) bool {
					clientConnCount++
					fmt.Println("gateway client", key, value)
					return true
				})
				allConnCount := 0
				g.connMapping.Range(func(key, value interface{}) bool {
					allConnCount++
					return true
				})
				registerConnCount := 0
				if g.isConnectedToRegister {
					registerConnCount = 1
				}
				fmt.Println("worker_count:", workerConnCount, "client_count", clientConnCount, "register_count", registerConnCount, "all_count:", allConnCount)
				fmt.Println("------------------------------gateway sum conn end-------------------------------------")
				fmt.Println()
			}
		}
	}() //监测内存中的连接数

	go func() {
		ticker := time.NewTicker(time.Second * 20)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				g.sendMsgToWorker()
			}
		}
	}()

	<-g.stopChan
}

/* ++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++
 * 发送消息给处理器
 * ++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++ */
func (gateway *gateway) sendMsgToWorker() {
	gateway.msgChan <- &ChanMsg{
		msg: &Message{
			Type: CLIENT_MSG_TO_GATEWAY,
			Payload: &MessagePayload{
				UserId:   "QQQQQQQQQQQ",
				TargetId: "DDDDDDDDDDD",
				Msg:      "XXXXXXXXXXX",
				Extend:   "WWWWWWWWWWW",
			},
			Timestamp: time.Now().Unix(),
		},
	}
}

/* ++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++
 * 监听服务
 * ++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++ */
func (gateway *gateway) server(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("gateway upgrade:", err)
		return
	}
	defer conn.Close()
	u1, err := uuid.NewV4()
	if err != nil {
		log.Println("gateway new uuid err:", err)
		return
	}
	connId := u1.String()

	for {
		jsonResult := &Message{}
		_, message, err := conn.ReadMessage()
		if err != nil {
			jsonResult.Type = CONN_CLOSE
			gateway.msgChan <- &ChanMsg{
				msg:    jsonResult,
				conn:   conn,
				connId: connId,
			}
			return
		}

		json.Unmarshal(message, jsonResult)

		gateway.msgChan <- &ChanMsg{
			msg:    jsonResult,
			conn:   conn,
			connId: connId,
		}
	}
}

/* ++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++
 * 连接注册中心
 * ++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++ */
func (gateway *gateway) dialRegister(remoteAddr string) {
	u1, err := uuid.NewV4()
	if err != nil {
		log.Printf("gateway new uuid err:%v", err)
		return
	}

	u := url.URL{Scheme: "ws", Host: remoteAddr, Path: "/"}
	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		log.Printf("gateway dial err:%v ", err)
		panic(err)
	}
	defer c.Close()

	connId := u1.String()
	//连接注册中心成功,发送绑定指令
	msg := &Message{}
	msg.Type = GATEWAY_CONN_TO_REGISTER
	msg.Payload = &MessagePayload{
		TargetId: local_gateway_addr,
	}
	msgJson, _ := json.Marshal(msg)
	err = c.WriteMessage(websocket.TextMessage, msgJson)
	if err != nil {
		log.Printf("gateway gateway_conn_to_register err:%v,data:%s", err, string(msgJson))
		return
	}

	//成功连接注册中心
	gateway.isConnectedToRegister = true
	//将连接记录map内存中
	if _, ok := gateway.connMapping.Load(connId); !ok {
		gateway.connMapping.Store(connId, ConnMap{
			MapKey: remoteAddr,
			Tag:    GATEWAY_CONN_TO_REGISTER,
		})
	}

	//接收数据
	go func() {
		for {
			jsonResult := &Message{}
			_, message, err := c.ReadMessage()
			if err != nil {
				jsonResult.Type = CONN_CLOSE
				gateway.msgChan <- &ChanMsg{
					msg:    jsonResult,
					conn:   c,
					connId: connId,
				}
				return
			}

			json.Unmarshal(message, jsonResult)

			gateway.msgChan <- &ChanMsg{
				msg:    jsonResult,
				conn:   c,
				connId: connId,
			}
		}
	}()

	ticker := time.NewTicker(time.Minute * 1)
	defer ticker.Stop()

	//心跳
	for {
		select {
		case <-ticker.C:
			msg := &Message{}
			msg.Type = HEART_BEAT
			msgJson, _ := json.Marshal(msg)
			err := c.WriteMessage(websocket.TextMessage, msgJson)
			if err != nil {
				log.Printf("gateway heart_beat err:%v,data:%s", err, string(msgJson))
				//连接断开需要移除映射
				jsonResult := &Message{}
				jsonResult.Type = CONN_CLOSE
				gateway.msgChan <- &ChanMsg{
					msg:    jsonResult,
					conn:   c,
					connId: connId,
				}
				return
			}
		}
	}
}

/* ++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++
 * 监测是否连接到注册中心，失败自动重连
 * ++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++ */
func (gateway *gateway) watchConnToRegister() {
	ticker := time.NewTicker(time.Minute * 1)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if !gateway.isConnectedToRegister {
				go func() {
					defer func() {
						if err := recover(); err != nil {
							log.Printf("gateway conn_to_register invoke func panic error:%v", err)
						}
					}()
					gateway.dialRegister(registerRemoteAddr)
				}()
			}
		}
	}
}

/* ++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++
 * 消费消息
 * ++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++ */
func (gateway *gateway) consumerMsg() {
	for {
		result, isOk := <-gateway.msgChan
		if !isOk {
			log.Print("gateway get unpack msgChan is closed")
			break
		}

		if result != nil {
			fmt.Println()
			fmt.Println("------------------------------gateway worker_conn_to_gateway begin -------------------------------------")

			switch result.msg.Type {
			case CONN_CLOSE:
				go gateway.offlineConn(result)
			case WORKER_MSG_TO_GATEWAY:
				{
					msgJson, err := json.Marshal(result.msg)
					fmt.Println("MSG:", string(msgJson), err)
				}
			case CLIENT_MSG_TO_GATEWAY:
				{
					message := result.msg
					message.Type = GATEWAY_MSG_TO_WORKER //修改type
					msgJson, err := json.Marshal(message)
					if err == nil {
						//通知任意一个处理器
						gateway.workerConns.Range(func(key, value interface{}) bool {
							if v, ok := value.(ConnFlag); ok {
								err = v.Conn.WriteMessage(websocket.TextMessage, msgJson)
								if err != nil {
									//移除连接
									jsonResult := &Message{}
									if err != nil {
										jsonResult.Type = CONN_CLOSE
										gateway.msgChan <- &ChanMsg{
											msg:    jsonResult,
											conn:   v.Conn,
											connId: v.ConnId,
										}
									}
								} else {
									//投递给一个处理器就ok
									return false
								}
							}
							return true
						})
					}
				}
			case WORKER_CONN_TO_GATEWAY:
				{
					//connid
					connId := result.connId
					//remoteAddr
					remoteAddr := result.conn.RemoteAddr().String()
					//conn
					conn := result.conn

					//处理映射
					if _, ok := gateway.connMapping.Load(connId); !ok {
						gateway.connMapping.Store(connId, ConnMap{
							MapKey: remoteAddr,
							Tag:    WORKER_CONN_TO_GATEWAY,
						})
					}
					//处理器连接到网关
					if _, ok := gateway.workerConns.Load(remoteAddr); !ok {
						gateway.workerConns.Store(remoteAddr, ConnFlag{
							ConnId:       connId,
							TerminalName: TERMINAL_NAME_SERVER,
							Conn:         conn,
						})
					}
					fmt.Println("gateway worker_conn_to_gateway:", connId, remoteAddr)

				}
			case CLIENT_CONN_TO_GATEWAY:
				{
					//connid
					connId := result.connId
					//remoteAddr
					remoteAddr := result.conn.RemoteAddr().String()
					//conn
					conn := result.conn
					//终端名称
					terninalName := result.msg.Type

					//处理映射
					if _, ok := gateway.connMapping.Load(connId); !ok {
						gateway.connMapping.Store(connId, ConnMap{
							MapKey: remoteAddr,
							Tag:    CLIENT_CONN_TO_GATEWAY,
						})
					}
					//客户端连接到网关
					if _, ok := gateway.clientConns.Load(remoteAddr); !ok {
						gateway.clientConns.Store(remoteAddr, ConnFlag{
							ConnId:       connId,
							TerminalName: terninalName,
							Conn:         conn,
						})
					}
				}
			case HEART_BEAT:
				{
					log.Println("heart_beat")
				}
			default:
				fmt.Println("gateway unkown type code")
			}
			msgJson, err := json.Marshal(result.msg)
			fmt.Println(string(msgJson), err)
			fmt.Println("------------------------------gateway worker_conn_to_gateway   end   -------------------------------------")
			fmt.Println()
		}
	}
}

/* ++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++
 * 连接断开下线
 * ++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++ */
func (gateway *gateway) offlineConn(message *ChanMsg) {
	if message != nil {
		//connId
		connId := message.connId

		v, ok := gateway.connMapping.Load(connId)
		if ok {
			if connmap, ok := v.(ConnMap); ok {

				switch connmap.Tag {
				case CLIENT_CONN_TO_GATEWAY:
					{
						//userId
						userId := connmap.MapKey

						if value, ok := gateway.clientConns.Load(userId); ok {
							//用户的所有连接
							if userConns, ok := value.([]ConnFlag); ok {
								for i := 0; i < len(userConns); i++ {
									//移除指定关闭的连接
									if userConns[i].ConnId == connId {
										userConns = append(userConns[:i], userConns[i+1:]...)
										gateway.clientConns.Store(userId, userConns)
										break
									}
								}
							}
						}

					}
				case WORKER_CONN_TO_GATEWAY:
					{
						//remoteAdrr
						remoteAddr := connmap.MapKey

						if _, ok := gateway.workerConns.Load(remoteAddr); ok {
							//移除断开的连接
							gateway.workerConns.Delete(remoteAddr)
						}

					}
				case GATEWAY_CONN_TO_REGISTER:
					{
						gateway.isConnectedToRegister = false
					}
				}

				//移除映射
				gateway.connMapping.Delete(connId)
			}
		}
	}
}
