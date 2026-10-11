// @ts-nocheck
// 提交锁只覆盖「点击到任务被接住」这一段。锁用令牌释放，后一次提交不会被前一次的收尾误放开。
export function createCreationSubmitGate() {
    let held = 0;
    let next = 0;

    return {
        tryAcquire() {
            if (held !== 0) return 0;
            held = ++next;
            return held;
        },
        release(token: number) {
            if (token && held === token) held = 0;
        },
    };
}
