package control

import (
	"p4r/entity"
)

type TableControl struct {
	control *Controller
	table   *entity.Table
}

// InsertEntryRaw 直接插入表项的方法
func (tc TableControl) InsertEntryRaw(action string, mf []entity.Match, params [][]byte) error {
	actions := *tc.control.Client.GetEntities("ACTION")
	actionID := actions[action].(*entity.Action).ID

	insertMessage := tc.table.InsertEntry(actionID, mf, params)
	return tc.control.Client.WriteUpdate(insertMessage)
}

// InsertEntry 提供更简洁的表项插入接口
func (tc TableControl) InsertEntry(action string, data map[string]interface{}) error {
	mf, params := tc.table.Transformer(data)
	return tc.InsertEntryRaw(action, mf, params)
}

// RegisterTransformer 注册表项转换器
func (tc TableControl) RegisterTransformer(transformer entity.TableEntryTransformer) {
	tc.table.RegisterTransformer(transformer)
}
