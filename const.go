package gsocket

const (
	CLIENT_MSG_TO_GATEWAY           = "client_msg_to_gateway"           //客户端发送消息给网关
	GATEWAY_MSG_TO_CLIENT           = "gateway_msg_to_client"           //网关发送消息给客户端
	GATEWAY_MSG_TO_WORKER           = "gateway_msg_to_worker"           //网关发送消息给服务端
	WORKER_MSG_TO_GATEWAY           = "worker_msg_to_gateway"           //处理器发送消息给网关
	CONN_CLOSE                      = "conn_close"                      //连接断开
	HEART_BEAT                      = "heart_beat"                      //心跳标识
	CLIENT_CONN_TO_GATEWAY          = "client_conn_to_gateway"          //客户端到网关标识
	REGISTER_BROADCAST_GATEWAY_ADDR = "register_broadcast_gateway_addr" //注册中心广播在线网关地址标识
	GATEWAY_CONN_TO_REGISTER        = "gateway_conn_to_register"        //网关连接到注册中心
	WORKER_CONN_TO_REGISTER         = "worker_conn_to_register"         //处理器连接到注册中心标识
	WORKER_CONN_TO_GATEWAY          = "worker_conn_to_gateway"          //处理器连接到网关标识
	TERMINAL_NAME_WEB               = "web"                             //web终端
	TERMINAL_NAME_H5                = "H5"                              //h5页面
	TERMINAL_NAME_ANDROID           = "android"                         //android终端
	TERMINAL_NAME_IOS               = "ios"                             //ios终端
	TERMINAL_NAME_IPAD              = "ipad"                            //ipad终端
	TERMINAL_NAME_SERVER            = "server"                          //服务器
)
