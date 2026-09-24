// @ts-nocheck
/** Strip upstream 影策 product chrome from user-visible copy. Keep protocol ids intact. */
export function debrandYingce(text: string) {
    if (!text) return text;
    return text
        .replaceAll("影策团队", "Yoyo")
        .replaceAll("影策创作者", "Yoyo")
        .replaceAll(" / 影策", "")
        .replaceAll("影策画布", "无限画布")
        .replaceAll("接入影策", "接入画布")
        .replaceAll("影策", "Yoyo");
}
