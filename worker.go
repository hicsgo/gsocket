package gsocket

import (
	"github.com/gorilla/websocket"
	"github.com/satori/go.uuid"
)

import (
	"sync"
	"fmt"
	"encoding/json"
	"log"
	"net/url"
	"time"
)

/* ================================================================================
 *  处理器中心模块
 *  author:golang123@outlook.com
 * ================================================================================ */
type worker struct {
	workerToGatewaysConns sync.Map //本地到多个网关连接集合   map["192.168.3.7:8866"]=connFlag{Id:"cae015bb-22a8-11e9-9bdd-00163e12d392",Tag:"web"}
	connMapping           sync.Map //连接和其他map的映射关系 map["cae015bb-22a8-11e9-9bdd-00163e12d392"]=connMap{"user_id":"1",Tag:"client"}
	stopChan              chan bool
	msgChan               chan *ChanMsg
	isConnectedToRegister bool //是否连接到注册中心
}

/* ++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++
 * 处理中心启动
 * ++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++ */
func WorkerRun() {

	//实例化网关
	g := &worker{
		workerToGatewaysConns: sync.Map{},
		stopChan:              make(chan bool, 0),
		msgChan:               make(chan *ChanMsg, 0),
		connMapping:           sync.Map{},
		isConnectedToRegister: false,
	}

	go g.dialRegister(registerRemoteAddr) //连接到注册中心

	go g.watchConnToRegister() //如果跟注册中心断开自动重连

	go g.workerHandler() //消费消息

	go func() {
		ticker := time.NewTicker(time.Second * 10)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				fmt.Println()
				fmt.Println("------------------------------worker sum conn begin-----------------------------------")
				gatewayConnCount := 0
				g.workerToGatewaysConns.Range(func(key, value interface{}) bool {
					gatewayConnCount++
					fmt.Println("gateway gateway ", key, value)
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
				fmt.Println("gateway_count:", gatewayConnCount, "register_count", registerConnCount, "all_count:", allConnCount)
				fmt.Println("------------------------------worker sum conn end-------------------------------------")
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
				g.TestBus()
			}
		}
	}()

	<-g.stopChan
}

/* ++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++
 * 业务逻辑处理
 * ++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++ */
func (worker *worker) workerHandler() {
	for {
		result, isOk := <-worker.msgChan
		if !isOk {
			log.Print("get unpack msgChan is closed")
			break
		}
		if result != nil {
			fmt.Println()
			fmt.Println("------------------------------worker recv register_broadcast_gateway_addr begin -------------------------------------")
			switch result.msg.Type {
			case CONN_CLOSE:
				go worker.offlineConn(result)
			case WORKER_MSG_TO_GATEWAY:
				message := result.msg
				msgJson, err := json.Marshal(message)
				if err == nil {
					//循环通知各个网关
					worker.workerToGatewaysConns.Range(func(key, value interface{}) bool {
						if v, ok := value.(ConnFlag); ok {
							err = v.Conn.WriteMessage(websocket.TextMessage, msgJson)
							if err != nil {
								//移除连接
								jsonResult := &Message{}
								if err != nil {
									jsonResult.Type = CONN_CLOSE
									worker.msgChan <- &ChanMsg{
										msg:    jsonResult,
										conn:   v.Conn,
										connId: v.ConnId,
									}
								}
							}
						}
						return true
					})
				}
			case GATEWAY_MSG_TO_WORKER:
				{
					log.Println("worker recv client msg from gateway ")
				}
			case REGISTER_BROADCAST_GATEWAY_ADDR:
				{

					//addr
					remoteAdrr := result.msg.Payload.TargetId

					if _, ok := worker.workerToGatewaysConns.Load(remoteAdrr); !ok {
						go func() {
							defer func() {
								if err := recover(); err != nil {
									log.Printf("gateway invoke func panic error:%v", err)
								}
							}()
							worker.dialGateway(remoteAdrr)
						}()
					}

				}
			default:
				log.Println("unkown type code")
			}
			jsonR, err := json.Marshal(result.msg)
			jsonR2, err := json.Marshal(worker.connMapping)
			fmt.Println("consumer msg", string(jsonR), err, result.connId, isOk, string(jsonR2))
			fmt.Println("------------------------------worker recv register_broadcast_gateway_addr end   -------------------------------------")
			fmt.Println()
		}

	}
}

//测试发送消息
func (worker *worker) TestBus() {

	worker.msgChan <- &ChanMsg{
		msg: &Message{
			Type: WORKER_MSG_TO_GATEWAY,
			Payload: &MessagePayload{
				UserId:   "1",
				TargetId: "2",
				Msg:      "哈哈",
				Extend:   "kk",
			},
			Timestamp: time.Now().Unix(),
		},
	}
}

/* ++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++
 * 连接网关
 * ++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++ */
func (worker *worker) dialGateway(remoteAddr string) {
	u1, err := uuid.NewV4()
	if err != nil {
		log.Println("new uuid err:", err)
		return
	}
	connId := u1.String()
	u := url.URL{Scheme: "ws", Host: remoteAddr, Path: "/"}

	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		fmt.Println("here", err)
		panic(err)
	}
	defer c.Close()

	//连接成功,发送绑定指令
	msg := &Message{}
	msg.Type = WORKER_CONN_TO_GATEWAY
	msgJson, _ := json.Marshal(msg)
	err = c.WriteMessage(websocket.TextMessage, msgJson)
	if err != nil {
		return
	}

	//处理器连接到网关
	if _, ok := worker.workerToGatewaysConns.Load(remoteAddr); !ok {
		worker.workerToGatewaysConns.Store(remoteAddr, ConnFlag{
			ConnId:       remoteAddr,
			TerminalName: TERMINAL_NAME_SERVER,
			Conn:         c,
		})
	}
	//连接映射
	if _, ok := worker.connMapping.Load(connId); !ok {
		worker.connMapping.Store(connId, ConnMap{
			MapKey: remoteAddr,
			Tag:    WORKER_CONN_TO_GATEWAY,
		})
	}

	//接收数据
	go func() {
		for {
			jsonResult := &Message{}
			_, message, err := c.ReadMessage()
			if err != nil {
				jsonResult.Type = CONN_CLOSE
				worker.msgChan <- &ChanMsg{
					msg:    jsonResult,
					conn:   c,
					connId: u1.String(),
				}
				return
			}

			json.Unmarshal(message, jsonResult)

			worker.msgChan <- &ChanMsg{
				msg:    jsonResult,
				conn:   c,
				connId: u1.String(),
			}
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
				log.Printf("worker heart_beat err:%v", err)
				//连接断开需要移除映射
				jsonResult := &Message{}
				jsonResult.Type = CONN_CLOSE
				worker.msgChan <- &ChanMsg{
					msg:    jsonResult,
					conn:   c,
					connId: u1.String(),
				}
				return
			}

		}
	}
}

/* ++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++
 * 连接注册中心
 * ++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++ */
func (worker *worker) dialRegister(remoteAddr string) {

	u1, err := uuid.NewV4()
	if err != nil {
		log.Println("new uuid err:", err)
		return
	}

	u := url.URL{Scheme: "ws", Host: remoteAddr, Path: "/"}

	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		log.Println("worker dial err:", err)
		panic(err)
	}
	defer c.Close()

	connId := u1.String()
	//连接成功,发送绑定指令
	msg := &Message{}
	msg.Type = WORKER_CONN_TO_REGISTER
	msgJson, _ := json.Marshal(msg)
	err = c.WriteMessage(websocket.TextMessage, msgJson)
	if err != nil {
		return
	}

	//标记已经连接到注册中心
	worker.isConnectedToRegister = true
	if _, ok := worker.connMapping.Load(connId); !ok {
		worker.connMapping.Store(connId, ConnMap{
			MapKey: remoteAddr,
			Tag:    WORKER_CONN_TO_REGISTER,
		})
	}

	//接收数据
	go func() {
		for {
			jsonResult := &Message{}
			_, message, err := c.ReadMessage()
			if err != nil {
				jsonResult.Type = CONN_CLOSE
				worker.msgChan <- &ChanMsg{
					msg:    jsonResult,
					conn:   c,
					connId: connId,
				}
				return
			}

			json.Unmarshal(message, jsonResult)

			worker.msgChan <- &ChanMsg{
				msg:    jsonResult,
				conn:   c,
				connId: connId,
			}
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
				log.Printf("worker heart_beat err:%v", err)
				//连接断开需要移除映射
				jsonResult := &Message{}
				jsonResult.Type = CONN_CLOSE
				worker.msgChan <- &ChanMsg{
					msg:    jsonResult,
					conn:   c,
					connId: u1.String(),
				}
				return
			}

		}
	}
}

/* ++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++
 * 连接断开下线
 * ++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++ */
func (worker *worker) offlineConn(message *ChanMsg) {
	if message != nil {
		v, ok := worker.connMapping.Load(message.connId)
		if ok {

			if connmap, ok := v.(ConnMap); ok {
				switch connmap.Tag {
				case WORKER_CONN_TO_GATEWAY:
					{
						if _, ok := worker.workerToGatewaysConns.Load(connmap.MapKey); ok {
							//移除断开的连接
							worker.workerToGatewaysConns.Delete(connmap.MapKey)
						}

					}
				case WORKER_CONN_TO_REGISTER:
					{
						worker.isConnectedToRegister = false
					}
				}

				//移除映射
				worker.connMapping.Delete(message.connId)
			}
		}
	}
}

/* ++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++
 * 监测是否连接到注册中心，失败自动重连
 * ++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++ */
func (worker *worker) watchConnToRegister() {
	ticker := time.NewTicker(time.Minute * 1)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C: //监测
			if !worker.isConnectedToRegister {
				go func() {
					defer func() {
						if err := recover(); err != nil {
							fmt.Sprintf("invoke func panic error:%v", err)
						}
					}()
					worker.dialRegister(registerRemoteAddr)
				}()
			}
		}
	}
}
