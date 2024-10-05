package entity

import (
	configv1 "github.com/p4lang/p4runtime/go/p4/config/v1"
	"github.com/p4lang/p4runtime/go/p4/v1"
)

// Id：这是一个 32 位无符号整数，用于唯一标识 P4 对象。所有 P4 对象的 ID 共享同一个编号空间，这意味着表的 ID 不能与计数器的 ID 重叠。
// ID 的分配方式使得可以根据 ID 推断资源类型（例如表、动作、计数器等）。ID 为 0 是保留的，表示无效 ID。
// Name：这是一个字符串，表示 P4 对象的全限定名称。例如，c1.c2.ipv4_lpm。这个名称通常是唯一的，用于明确标识 P4 对象。
// 这些字段用于确保 P4 对象在系统中的唯一性和可识别性。
// configv1.Action 描述 P4 运行时中的动作（Action）对象

type Action struct {
	Name string
	ID   uint32
}

func (a *Action) GetID() uint32 {
	return a.ID
}
func (a *Action) Type() string {
	return "ACTION"
}

func GetAction(ac *configv1.Action) Action {
	return Action{
		Name: ac.Preamble.Name,
		ID:   ac.Preamble.Id,
	}
}

// Entity 是P4 Runtime中的一个抽象概念，它代表交换机中的具体资源，如表项、计数器、Meter、Digest等。
// 控制器通过与 Entity 交互来配置和管理这些资源。
// Entity 是控制器通过P4 Runtime与交换机进行交互的核心对象，它可以用于以下操作：
//
//	插入：通过 INSERT 操作向交换机添加新的实体条目。
//	修改：通过 MODIFY 操作更改现有的实体条目。
//	删除：通过 DELETE 操作删除交换机中的某个实体条目。
//	读取：通过 READ 操作获取交换机中的实体条目信息。
//
// 常见的 Entity 类型包括：
//  1. TableEntry：表示表中的一个具体条目。交换机根据匹配字段（如IP地址）执行特定操作（如转发）。
//  2. CounterEntry：表示用于流量统计的计数器。
//  3. MeterEntry：表示用于流量控制的Meter。
//  4. DigestEntry：表示交换机发送给控制器的一种摘要数据结构，帮助交换机将某些网络事件的概要信息发送给控制器进行处理。
type Entity interface {
	Type() string
	GetID() uint32
}

// Counter stores all the information we need about a counter
type Counter struct {
	ID   uint32
	Size int64
}

// ReadValueWithIndex 用于读取特定索引处的计数器值
func (c *Counter) ReadValueWithIndex(index int64) *v1.Entity {
	entry := &v1.CounterEntry{
		CounterId: c.ID,
		Index:     &v1.Index{Index: index},
	}
	entity := &v1.Entity{
		Entity: &v1.Entity_CounterEntry{CounterEntry: entry},
	}
	return entity
}

// ReadValue 读取所有索引处的计数器值
func (c *Counter) ReadValue() *v1.Entity {
	entry := &v1.CounterEntry{
		CounterId: c.ID,
	}
	entity := &v1.Entity{
		Entity: &v1.Entity_CounterEntry{CounterEntry: entry},
	}
	return entity
}

func (c *Counter) Type() string {
	return "COUNTER"
}

func (c *Counter) GetID() uint32 {
	return c.ID
}

func GetCounter(counter *configv1.Counter) Counter {
	return Counter{
		ID:   counter.Preamble.Id,
		Size: counter.Size,
	}
}

type Digest struct {
	ID uint32
}

// Insert 插入一条digest条目
func (d *Digest) Insert(entry *v1.DigestEntry) *v1.Update {
	update := &v1.Update{
		Type: v1.Update_INSERT,
		Entity: &v1.Entity{
			Entity: &v1.Entity_DigestEntry{DigestEntry: entry},
		},
	}

	return update
}

// Modify 修改一条digest条目
func (d *Digest) Modify(entry *v1.DigestEntry) *v1.Update {
	update := &v1.Update{
		Type: v1.Update_MODIFY,
		Entity: &v1.Entity{
			Entity: &v1.Entity_DigestEntry{DigestEntry: entry},
		},
	}

	return update
}

// Delete 删除一条digest条目
func (d *Digest) Delete() *v1.Update {
	entry := &v1.DigestEntry{
		DigestId: d.ID,
	}
	update := &v1.Update{
		Type: v1.Update_DELETE,
		Entity: &v1.Entity{
			Entity: &v1.Entity_DigestEntry{DigestEntry: entry},
		},
	}

	return update
}

// Acknowledge 用于向P4 switch发送“digest确认”。
func (d *Digest) Acknowledge(digestList *v1.DigestList) *v1.StreamMessageRequest {
	return &v1.StreamMessageRequest{
		Update: &v1.StreamMessageRequest_DigestAck{DigestAck: &v1.DigestListAck{
			DigestId: d.ID,
			ListId:   digestList.ListId,
		}},
	}
}

func (d *Digest) Type() string {
	return "DIGEST"
}

func (d *Digest) GetID() uint32 {
	return d.ID
}

func GetDigest(digest *configv1.Digest) Digest {
	return Digest{
		ID: digest.Preamble.Id,
	}
}

// Table 表示一个 P4 表的实体，包含以下字段：
//
//	ID：表的唯一标识符（uint32 类型）。
//	Name：表的名称（string 类型）。
//	Transformer：类型为 TableEntryTransformer 的函数，用于将数据转换为与 P4 Runtime 兼容的格式。
type Table struct {
	ID          uint32
	Name        string
	Transformer TableEntryTransformer
}

// DirectCounterForTableEntry 获取与指定表项关联的 DirectCounter 的值。
func (t *Table) DirectCounterForTableEntry(matches []Match) *v1.Entity {
	tableEntry := &v1.TableEntry{
		TableId: t.ID,
	}
	for idx, m := range matches {
		tableEntry.Match = append(tableEntry.Match, m.get(uint32(idx+1)))
	}

	dcEntry := &v1.DirectCounterEntry{
		TableEntry: tableEntry,
	}
	entity := &v1.Entity{
		Entity: &v1.Entity_DirectCounterEntry{DirectCounterEntry: dcEntry},
	}
	return entity
}

// AllDirectCountersForTable 获取与特定表的所有条目相关的 DirectCounters 的值。
func (t *Table) AllDirectCountersForTable() *v1.Entity {
	tableEntry := &v1.TableEntry{
		TableId: t.ID,
	}
	dcEntry := &v1.DirectCounterEntry{
		TableEntry: tableEntry,
	}
	entity := &v1.Entity{
		Entity: &v1.Entity_DirectCounterEntry{DirectCounterEntry: dcEntry},
	}
	return entity
}

// AllDirectCounters 获取所有表中所有条目相关的 DirectCounters（直接计数器）的值。
func AllDirectCounters() *v1.Entity {
	tableEntry := &v1.TableEntry{
		TableId: 0,
	}
	dcEntry := &v1.DirectCounterEntry{
		TableEntry: tableEntry,
	}
	entity := &v1.Entity{
		Entity: &v1.Entity_DirectCounterEntry{DirectCounterEntry: dcEntry},
	}
	return entity
}

type Match interface {
	get(ID uint32) *v1.FieldMatch
}

// ExactMatch 代表精确匹配（Exact Match）功能，包含一个字节切片 Value 用于存储要匹配的值
type ExactMatch struct {
	Value []byte
}

// LpmMatch 代表最长前缀匹配（Longest Prefix Match）功能，包含两个字段：
//   - Value：字节切片，用于存储要匹配的值。
//   - PLen：整数，表示前缀长度。
type LpmMatch struct {
	Value []byte
	PLen  int32
}

func (m *ExactMatch) get(ID uint32) *v1.FieldMatch {
	exact := &v1.FieldMatch_Exact{
		Value: m.Value,
	}
	mf := &v1.FieldMatch{
		FieldId:        ID,
		FieldMatchType: &v1.FieldMatch_Exact_{Exact: exact},
	}
	return mf
}

func (m *LpmMatch) get(ID uint32) *v1.FieldMatch {
	lpm := &v1.FieldMatch_LPM{
		Value:     m.Value,
		PrefixLen: m.PLen,
	}

	firstByteMasked := int(m.PLen / 8)
	if firstByteMasked != len(m.Value) {
		i := firstByteMasked
		r := m.PLen % 8
		m.Value[i] = m.Value[i] & (0xff << (8 - r))
		for i = i + 1; i < len(m.Value); i++ {
			m.Value[i] = 0
		}
	}

	mf := &v1.FieldMatch{
		FieldId:        ID,
		FieldMatchType: &v1.FieldMatch_Lpm{Lpm: lpm},
	}

	return mf
}

// TableEntryTransformer 用于将 JSON 数据转换为 P4 Runtime 兼容的数据格式。
//   - 可以用于将应用层的 JSON 数据转换为底层 P4 Runtime 所需的格式。
type TableEntryTransformer func(map[string]interface{}) ([]Match, [][]byte)

// InsertEntry 插入一个条目
// 功能：该方法的目的是创建并返回一个 P4 Runtime 更新请求，表示要插入或修改表中的条目。
//   - 输入参数：
//   - actionID uint32：表示要执行的动作的唯一标识符，通常与某个特定的动作（如转发、丢弃等）相关联。
//   - mfs []Match：一个 Match 接口的切片，定义了条目的匹配条件。可以是精确匹配、最长前缀匹配等。
//   - params [][]byte：与动作相关的参数，通常是与特定操作相关的值。
func (t *Table) InsertEntry(actionID uint32, mfs []Match, params [][]byte) *v1.Update {
	directAction := &v1.Action{
		ActionId: actionID,
	}

	for idx, p := range params {
		param := &v1.Action_Param{
			ParamId: uint32(idx + 1),
			Value:   p,
		}
		directAction.Params = append(directAction.Params, param)
	}

	// 创建一个 TableAction 对象，将 directAction 包装在其中，表示要在表上执行的操作。
	tableAction := &v1.TableAction{
		Type: &v1.TableAction_Action{Action: directAction},
	}

	// 创建一个 TableEntry 对象，设置表的 ID、动作以及是否为默认动作。默认动作是指在没有其他匹配项的情况下执行的操作。
	entry := &v1.TableEntry{
		TableId:         t.ID,
		Action:          tableAction,
		IsDefaultAction: (mfs == nil),
	}

	// 遍历 mfs，将每个 Match 对象转换为 P4 Runtime 所需的格式，并添加到 entry.Match 中。
	for idx, mf := range mfs {
		entry.Match = append(entry.Match, mf.get(uint32(idx+1)))
	}

	// 根据 mfs 是否为 nil 来决定是插入新条目还是修改现有条目。若没有匹配条件，则视为修改现有条目。
	var updateType v1.Update_Type
	if mfs == nil {
		updateType = v1.Update_MODIFY
	} else {
		updateType = v1.Update_INSERT
	}
	update := &v1.Update{
		Type: updateType,
		Entity: &v1.Entity{
			Entity: &v1.Entity_TableEntry{TableEntry: entry},
		},
	}

	return update
}

func (t *Table) Type() string {
	return "TABLE"
}

func (t *Table) GetID() uint32 {
	return t.ID
}

func (t *Table) RegisterTransformer(transformer TableEntryTransformer) {
	t.Transformer = transformer
}

func GetTable(t *configv1.Table) Table {
	return Table{
		Name: t.Preamble.Name,
		ID:   t.Preamble.Id,
	}
}
