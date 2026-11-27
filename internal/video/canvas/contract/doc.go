// Package contract 定义画布节点和生成任务之间的生成合同（GenerationSpec），不含密钥和供应商配置。
//
// 合同只有类型，没有行为：internal/video 把节点上的 generationSpec JSON 解成 GenerationSpec，再按字段取用。
// 结构体上除 json 以外的 tag 只作文档，运行时不读取，详见 generation.go 的说明。
package contract
