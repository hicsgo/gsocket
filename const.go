package gsocket

const (
	WORKER_MSG             = 20000                //服务端消息
	CLIENT_MSG             = 30000                //客户端消息
	GATEWAY_MSG            = 40000                //网关转发消息
	CONN_CLOSE             = 50000                //连接断开
	HEART_BEAT             = 60000                //心跳标识
	TAG_CLIENT_TO_GATEWAY  = "client_to_gateway"  //客户端到网关标识
	TAG_WORKER_TO_GATEWAY  = "worker_to_gateway"  //处理器到网关标识
	TAG_GATEWAY_TO_GATEWAY = "gateway_to_gateway" //网关连到网关标识
	CONN_TO_REGISTER       = "conn_to_register"   //连接到注册中心标识
)
