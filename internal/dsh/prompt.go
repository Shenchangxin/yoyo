// prompt.go 把渲染后的系统提示投影成日志提交。
// 它不调用模型。in-history 为真且请求系列未换时，变化追加在历史后面；否则改写头部。

package dsh

// ProjectSystem 计算系统提示提交。nodes 是当前系统节点正文，可以含空串。
// rendered 是这一步要生效的提示。inHistory 为真表示路线能在历史里追加提示。
// startsSeries 为真表示这一步开始新的请求系列。正文与当前生效提示相同时不产生提交。
func ProjectSystem(nodes []string, rendered string, inHistory, startsSeries bool) []SystemCommit {
	if rendered == "" {
		var out []SystemCommit
		if len(nodes) == 0 {
			return []SystemCommit{{Op: "append", Text: ""}}
		}
		for i, n := range nodes {
			if n != "" {
				out = append(out, SystemCommit{Op: "replace", Index: i, Text: ""})
			}
		}
		return out
	}
	if len(nodes) == 0 {
		return []SystemCommit{{Op: "append", Text: rendered}}
	}
	effective := ""
	for _, n := range nodes {
		if n != "" {
			effective = n
		}
	}
	if effective == rendered {
		return nil
	}
	if inHistory && !startsSeries {
		return []SystemCommit{{Op: "append", Text: rendered}}
	}
	var out []SystemCommit
	for i := len(nodes) - 1; i >= 1; i-- {
		if nodes[i] != "" {
			out = append(out, SystemCommit{Op: "replace", Index: i, Text: ""})
		}
	}
	out = append(out, SystemCommit{Op: "replace", Index: 0, Text: rendered})
	return out
}
