package main

import (
	"testing"
	"github.com/stretchr/testify/assert"
)

func TestSearchAllUserWalletFromDB(t *testing.T) {
	// 假设你的测试数据库已经设置好并连接到正确的数据库

	// 初始化测试数据
	// TODO: 在测试数据库中插入符合条件的记录，确保查询结果非空

	// 执行函数进行测试
	wallets, err := searchAllUserWalletFromDB()

	// 断言结果
	assert.NoError(t, err)
	assert.NotEmpty(t, wallets)

	// 清理测试数据
	// TODO: 在测试数据库中清理插入的测试记录
	t.Log("Wallets:", wallets)
}
