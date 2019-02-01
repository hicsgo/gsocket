package gsocket

const (
	WORKER_MSG                      = "worker_msg"                      //服务端消息
	CLIENT_MSG                      = "client_msg"                      //客户端消息
	GATEWAY_MSG                     = "gateway_msg"                     //网关转发消息
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
