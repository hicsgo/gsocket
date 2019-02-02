package gsocket

import (
	"github.com/gorilla/websocket"
)

import (
	"net/http"
	"log"
	"sync"
	"github.com/satori/go.uuid"
	"encoding/json"
	"fmt"
	"time"
)

/* ================================================================================
 *  注册中心模块
 *  author:golang123@outlook.com
 * ================================================================================ */

var local_register_addr = "127.0.0.1:8080"

type register struct {
	gatewaysToRegisterConns sync.Map      //多个网关到本地注册中心集合     map["192.168.1.1:1234"]=ConnFlag{ConnId:"cae015bb-22a8-11e9-9bdd-00163e12d392",TerminalName:"server",conn:websocket.Conn}
	workersToRegisterConns  sync.Map      //多个处理器到本地注册中心集合   map["192.168.3.7:8866"]=ConnFlag{ConnId:"cae015bb-22a8-11e9-9bdd-00163e12d377",TerminalName:"server",conn:websocket.Conn}
	connMapping             sync.Map      //连接和其他map的映射关系       map["cae015bb-22a8-11e9-9bdd-00163e12d392"]=ConnMap{"MapKey":"192.168.1.1:1234",Tag:"gateway_conn_to_register"}
	msgChan                 chan *ChanMsg //消息
	stopChan                chan bool     //终止
}

/* ++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++
 * 启动注册中心
 * ++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++ */
func RegisterRun() {
	//实例化注册中心
	r := &register{
		gatewaysToRegisterConns: sync.Map{},
		workersToRegisterConns:  sync.Map{},
		stopChan:                make(chan bool, 0),
		msgChan:                 make(chan *ChanMsg, 0),
	}
	go func() {
		http.HandleFunc("/", r.server)
		log.Fatal(http.ListenAndServe(local_register_addr, nil))
	}() //启动监听

	go r.consumerMsg() //消费消息

	go func() {
		ticker := time.NewTicker(time.Minute * 1)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				fmt.Println()
				fmt.Println("------------------------------ register sum conn begin -----------------------------------")
				gatewayConnCount := 0
				r.gatewaysToRegisterConns.Range(func(key, value interface{}) bool {
					gatewayConnCount++
					fmt.Println("register gateway ", key, value)
					return true
				})
				workerConnCount := 0
				r.workersToRegisterConns.Range(func(key, value interface{}) bool {
					workerConnCount++
					fmt.Println("register worker", key, value)
					return true
				})
				allConnCount := 0
				r.connMapping.Range(func(key, value interface{}) bool {
					allConnCount++
					return true
				})
				fmt.Println("gateway_count:", gatewayConnCount, "worker_count:", workerConnCount, "all_count:", allConnCount)
				fmt.Println("------------------------------ register sum conn end  -------------------------------------")
				fmt.Println()
			}
		}
	}()

	<-r.stopChan
}

/* ++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++
 * 监听服务
 * ++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++ */
func (register *register) server(w http.ResponseWriter, r *http.Request) {
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

	//connId
	connId := u1.String()

	//接收数据
	go func() {
		for {
			jsonResult := &Message{}
			_, message, err := c.ReadMessage()
			if err != nil {
				jsonResult.Type = CONN_CLOSE
				register.msgChan <- &ChanMsg{
					msg:    jsonResult,
					conn:   c,
					connId: connId,
				}
				return
			}

			json.Unmarshal(message, jsonResult)

			register.msgChan <- &ChanMsg{
				msg:    jsonResult,
				conn:   c,
				connId: connId,
			}
		}
	}()

	ticker := time.NewTicker(time.Second * 10)
	defer ticker.Stop()
	//发送数据
	for {
		select {
		case <-ticker.C:
			if v, ok := register.connMapping.Load(connId); ok {
				if connmap, ok := v.(ConnMap); ok {
					remoteAddr := connmap.MapKey
					//如果当前连接是处理器连接,广播网关地址
					if connmap.Tag == WORKER_CONN_TO_REGISTER {
						fmt.Println()
						fmt.Println("------------------------------register register_broadcast_gateway_addr begin -------------------------------------")
						register.gatewaysToRegisterConns.Range(func(key, value interface{}) bool {
							if onlineGatewayAddr, ok := key.(string); ok {
								msg := &Message{}
								msg.Type = REGISTER_BROADCAST_GATEWAY_ADDR
								msg.Payload = &MessagePayload{
									TargetId: onlineGatewayAddr,
								}
								msgJson, _ := json.Marshal(msg)
								err := c.WriteMessage(websocket.TextMessage, msgJson)
								if err != nil {
									log.Println("write:", err)
									//连接断开需要移除映射
									jsonResult := &Message{}
									fmt.Println("发送在线网关地址失败", time.Now().Nanosecond())
									jsonResult.Type = CONN_CLOSE
									register.msgChan <- &ChanMsg{
										msg:    jsonResult,
										conn:   c,
										connId: connId,
									}
									return false
								}
								fmt.Println("** register register_broadcast_gateway_addr,remoteAddr:", remoteAddr, "notify gateway addr:", onlineGatewayAddr)
							}
							return true
						})
						fmt.Println("------------------------------register register_broadcast_gateway_addr end -------------------------------------")
						fmt.Println()
					}
				}
			}
		}
	}
}

/* ++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++
 * 消费消息
 * ++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++ */
func (register *register) consumerMsg() {
	for {
		result, isOk := <-register.msgChan
		if !isOk {
			log.Print("get unpack msgChan is closed")
			break
		}
		fmt.Println()
		fmt.Println("-------------------------register begin recv -------------------------")
		fmt.Println("register consumerMsg rev", result)
		if result != nil {

			switch result.msg.Type {
			case CONN_CLOSE:
				//移除连接以及连接映射
				go register.offlineConn(result)
			case WORKER_CONN_TO_REGISTER:
				{
					//connId
					connId := result.connId
					//remoteAddr
					remoteAddr := result.conn.RemoteAddr().String()
					//conn
					conn := result.conn

					fmt.Println("register worker_conn_to_register")
					if _, ok := register.connMapping.Load(connId); !ok {
						register.connMapping.Store(connId, ConnMap{
							MapKey: remoteAddr,
							Tag:    WORKER_CONN_TO_REGISTER,
						})
						json, er := json.Marshal(register.connMapping)
						fmt.Println(string(json), er, "HHHHHHHHHHHHHHHHHHHHHHHHHHHHHHHHHHH", register.connMapping)

						if _, ok := register.workersToRegisterConns.Load(remoteAddr); !ok {
							register.workersToRegisterConns.Store(remoteAddr, ConnFlag{
								ConnId:       connId,
								TerminalName: TERMINAL_NAME_SERVER,
								Conn:         conn,
							})
						}
					}
				}
			case HEART_BEAT:
				{
					fmt.Println("register heart_beat")
				}
			case GATEWAY_CONN_TO_REGISTER:
				{
					//connId
					connId := result.connId
					//remoteAddr
					remoteAddr := result.msg.Payload.TargetId
					//conn
					conn := result.conn

					fmt.Println("register gateway_conn_to_register")
					if _, ok := register.connMapping.Load(connId); !ok {
						register.connMapping.Store(connId, ConnMap{
							MapKey: remoteAddr,
							Tag:    GATEWAY_CONN_TO_REGISTER,
						})
						json, er := json.Marshal(register.connMapping)
						fmt.Println(string(json), er, "GGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGG", register.connMapping)

						if _, ok := register.gatewaysToRegisterConns.Load(remoteAddr); !ok {
							register.gatewaysToRegisterConns.Store(remoteAddr, ConnFlag{
								ConnId:       connId,
								TerminalName: TERMINAL_NAME_SERVER,
								Conn:         conn,
							})
						}
					}
				}
			default:
				fmt.Println("unkown type code")
			}
		}
		jsonR, err := json.Marshal(result.msg)
		jsonR2, err := json.Marshal(register.connMapping)
		fmt.Println("consumer msg", string(jsonR), err, result.connId, isOk, string(jsonR2))
		fmt.Println("-------------------------register recv  end  -------------------------")
		fmt.Println()

	}
}

/* ++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++
 * 连接断开下线
 * ++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++ */
func (register *register) offlineConn(message *ChanMsg) {
	if message != nil {

		v, ok := register.connMapping.Load(message.connId)

		if ok {
			if connmap, ok := v.(ConnMap); ok {
				//remoteAddr
				remoteAddr := connmap.MapKey
				//connid
				connId := message.connId

				switch connmap.Tag {
				case WORKER_CONN_TO_REGISTER:
					{
						if _, ok := register.workersToRegisterConns.Load(remoteAddr); ok {
							//移除断开的处理器连接
							register.workersToRegisterConns.Delete(remoteAddr)
						}
					}
				case GATEWAY_CONN_TO_REGISTER:
					{
						if _, ok := register.gatewaysToRegisterConns.Load(remoteAddr); ok {
							//移除断开的网关连接
							register.gatewaysToRegisterConns.Delete(remoteAddr)
						}
					}
				}

				//移除映射
				register.connMapping.Delete(connId)
			}
		}
	}
}
