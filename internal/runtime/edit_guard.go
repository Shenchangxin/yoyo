// 占位参数守卫。上下文投影写进参数里的 _elided、_hint、_recall 不能落到文件系统。str_replace 和 office_edit 在改文件前调用 checkEditArgs。

package runtime

import (
	"fmt"
	"strings"
)

// placeholderKeys 是只有上下文投影会写的参数键：_elided、_hint、_recall。调用里带着它们，就说明参数是从摘要抄来的，不能当成本次要写的正文。
var placeholderKeys = []string{"_elided", "_hint", "_recall"}

// checkEditArgs 拒绝不是模型真实载荷的编辑。看的是键在不在，不是值空不空。缺 new_str 时 str 会把 nil 收成空串，str_replace 会把 old_str 删掉；键缺失必须先拒绝。
func checkEditArgs(tool string, args map[string]any, required ...string) error {
	for _, k := range placeholderKeys {
		if _, ok := args[k]; ok {
			return fmt.Errorf("refusing %s: argument %q is a context placeholder, not part of the call. Re-issue %s with the full %s value", tool, k, tool, strings.Join(required, " and "))
		}
	}
	for _, k := range required {
		if _, ok := args[k]; !ok {
			return fmt.Errorf("refusing %s: required argument %q is missing. Send it explicitly; use an empty string only when you mean an empty value", tool, k)
		}
	}
	return nil
}
