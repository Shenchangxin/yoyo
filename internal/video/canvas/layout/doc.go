// Package layout 是画布节点排布的确定性几何实现，供画布 Agent 做整理、归类和新增落位。
//
// 它和前端 web/src/lib/canvas/canvas-layout.ts 用同一套常量和算法，这样前端自动整理和 Agent 整理会得到同样的结果。
// 改任何一边的常量或算法，另一边都要同步改；两边各有一组黄金用例锁住结果。
//
// 这个包只做几何（Auto、Flow、ByLane、Linear、Align、Bands、FreeSlot 等），不读画布文档、不判断权限，也不写库。
package layout
