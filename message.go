package gsocket

import (
	"github.com/hicsgo/glib"
)

type (
	Message struct {
		Payload   *MessagePayload `form:"payload" json:"payload"`     //消息内容
		Type      string          `form:"type" json:"type"`           //类型码
		Timestamp int64           `form:"timestamp" json:"timestamp"` //unix时间戳，单位秒
	}

	//消息载荷
	MessagePayload struct {
		UserId   string `form:"user_id" json:"user_id"`     //用户id
		TargetId string `form:"target_id" json:"target_id"` //目标id
		Msg      string `form:"msg" json:"msg"`             //传递的信息
		Extend   string `form:"extend" json:"extend"`       //扩展信息
	}
)

/* ++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++
 * 对象转成Json字符串
 * ++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++ */
func (message *Message) ToJson() string {
	jsonString, err := glib.ToJson(message)
	if err != nil {
		return ""
	}

	return jsonString
}
