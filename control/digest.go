package control

import (
	"github.com/p4lang/p4runtime/go/p4/v1"
	"p4r/entity"
)

// DigestControl 用于在控制器中处理 P4 中的 DigestEntries。
// 主要功能包括在交换机中插入、修改、删除 DigestEntries 以及向交换机确认接收到的 Digest 消息。
type DigestControl struct {
	control *Controller
	digest  *entity.Digest
}

func (dc *DigestControl) getDigestEntryConfig(maxListSize int32, maxTimeoutNs, ackTimeoutNs int64) *v1.DigestEntry {
	return &v1.DigestEntry{
		DigestId: dc.digest.ID,
		Config: &v1.DigestEntry_Config{
			MaxTimeoutNs: maxTimeoutNs,
			MaxListSize:  maxListSize,
			AckTimeoutNs: ackTimeoutNs,
		},
	}
}

// Insert 向交换机插入新的 DigestEntry
func (dc DigestControl) Insert(maxListSize int32, maxTimeoutNs, ackTimeoutNs int64) error {
	entry := dc.getDigestEntryConfig(maxListSize, maxTimeoutNs, ackTimeoutNs)
	update := dc.digest.Insert(entry)
	return dc.control.Client.WriteUpdate(update)
}

// Modify 修改交换机上的现有 DigestEntry
func (dc DigestControl) Modify(maxListSize int32, maxTimeoutNs, ackTimeoutNs int64) error {
	entry := dc.getDigestEntryConfig(maxListSize, maxTimeoutNs, ackTimeoutNs)
	update := dc.digest.Modify(entry)
	return dc.control.Client.WriteUpdate(update)
}

// Delete 删除交换机中的 DigestEntry，表示控制器不再接收对应的 Digest 消息。
func (dc DigestControl) Delete() error {
	update := dc.digest.Delete()
	return dc.control.Client.WriteUpdate(update)

}

// Acknowledge 用于确认控制器已经收到一个 DigestList。通过向 OutgoingMessageChannel 发送确认消息通知交换机。
func (dc DigestControl) Acknowledge(digestList *v1.DigestList) {
	message := dc.digest.Acknowledge(digestList)
	reqChannel := dc.control.Client.GetMessageChannels().OutgoingMessageChannel
	reqChannel <- message
}
