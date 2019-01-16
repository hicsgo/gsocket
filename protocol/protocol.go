package protocol

import (
	"net"
	"bufio"
	"io"
	"hash/crc32"
	"fmt"
)

type customProtocol struct {
	packInput    []byte
	packOutput   []byte
	unPackResult chan []byte
	conn         net.Conn
	isColosed    bool
	connOwner    string
}

/* ++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++
 * 设置当前连接归属
 * ++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++ */
func (c *customProtocol) SetConnOwner(connOwner string) {
	c.connOwner = connOwner
}

/* ++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++
 * 设置当前连接归属
 * ++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++ */
func (c *customProtocol) GetConnOwner() string {
	return c.connOwner
}

/* ++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++
 * 设置自定义协议封装包输入参数
 * ++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++ */
func (c *customProtocol) SetPackInput(packInput []byte) {
	c.packInput = packInput
}

/* ++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++
 * 获取自定义协议封装包输入参数的结果
 * ++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++ */
func (c *customProtocol) GetPackOutPut() (packOutput []byte) {
	return c.packOutput
}

/* ++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++
 * 设置当前连接关闭
 * ++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++ */
func (c *customProtocol) SetConnColse() {
	c.isColosed = true
}

/* ++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++
 * 设置当前的连接实例
 * ++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++ */
func (c *customProtocol) SetConn(conn net.Conn) {
	c.conn = conn
}

/* ++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++
 * 返回当前连接
 * ++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++ */
func (c *customProtocol) GetConn() (conn net.Conn) {
	return c.conn
}

/* ++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++
 * 获取长连接中的结果
 * ++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++ */
func (c *customProtocol) GetUnPackResult() (chan []byte) {
	go c.unPack()
	return c.unPackResult
}

func NewCustomerProtocol() *customProtocol {
	return &customProtocol{
		unPackResult: make(chan []byte, 0),
	}
}

/* ++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++
 * 发送数据
 * ++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++ */
func (c *customProtocol) Send() (n int, err error) {
	c.pack()
	return c.conn.Write(c.packOutput)
}

/* ++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++
 * 封装包协议
 * ++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++ */
func (c *customProtocol) pack() {
	packetLength := len(c.packInput) + 8
	result := make([]byte, packetLength)
	result[0] = 0xFF
	result[1] = 0xFF
	result[2] = byte(uint16(len(c.packInput)) >> 8)
	result[3] = byte(uint16(len(c.packInput)) & 0xFF)
	copy(result[4:], c.packInput)
	sendCrc := crc32.ChecksumIEEE(c.packInput)
	result[packetLength-4] = byte(sendCrc >> 24)
	result[packetLength-3] = byte(sendCrc >> 16 & 0xFF)
	result[packetLength-2] = 0xFF
	result[packetLength-1] = 0xFE
	c.packOutput = result
}

/* ++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++
 * 解包协议
 * ++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++ */
func (c *customProtocol) unPack() {
	defer c.conn.Close()
	// 状态机状态
	state := 0x00
	// 数据包长度
	length := uint16(0)
	// crc校验和
	crc16 := uint16(0)
	var recvBuffer []byte
	// 游标
	cursor := uint16(0)
	bufferReader := bufio.NewReader(c.conn)
	//状态机处理数据
	for {
		recvByte, err := bufferReader.ReadByte()
		if err != nil {
			//这里因为做了心跳，所以就没有加deadline时间，如果客户端断开连接
			//这里ReadByte方法返回一个io.EOF的错误，具体可考虑文档
			if err == io.EOF {
				if !c.isColosed {
					close(c.unPackResult)
					c.SetConnColse()
				}
				fmt.Printf("client %s is close!\n", c.conn.RemoteAddr().String())
			}
			//在这里直接退出goroutine，关闭由defer操作完成
			return
		}
		//进入状态机，根据不同的状态来处理
		switch state {
		case 0x00:
			if recvByte == 0xFF {
				state = 0x01
				//初始化状态机
				recvBuffer = nil
				length = 0
				crc16 = 0
			} else {
				state = 0x00
			}
			break
		case 0x01:
			if recvByte == 0xFF {
				state = 0x02
			} else {
				state = 0x00
			}
			break
		case 0x02:
			length += uint16(recvByte) * 256
			state = 0x03
			break
		case 0x03:
			length += uint16(recvByte)
			// 一次申请缓存，初始化游标，准备读数据
			recvBuffer = make([]byte, length)
			cursor = 0
			state = 0x04
			break
		case 0x04:
			//不断地在这个状态下读数据，直到满足长度为止
			recvBuffer[cursor] = recvByte
			cursor++
			if (cursor == length) {
				state = 0x05
			}
			break
		case 0x05:
			crc16 += uint16(recvByte) * 256
			state = 0x06
			break
		case 0x06:
			crc16 += uint16(recvByte)
			state = 0x07
			break
		case 0x07:
			if recvByte == 0xFF {
				state = 0x08
			} else {
				state = 0x00
			}
		case 0x08:
			if recvByte == 0xFE {
				//执行数据包校验
				if (crc32.ChecksumIEEE(recvBuffer)>>16)&0xFFFF == uint32(crc16) {
					//添加到chan中
					c.unPackResult <- recvBuffer
				} else {
					fmt.Println(string(recvBuffer))
					fmt.Println("丢弃数据!")
				}
			}
			//状态机归位,接收下一个包
			state = 0x00
		}
	}
}
