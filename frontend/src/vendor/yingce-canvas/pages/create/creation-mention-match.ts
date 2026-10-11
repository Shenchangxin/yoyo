const MENTION_LABEL_BOUNDARY = String.raw`(?=$|\s|[,.!?;:，。！？；：、)\]}】）])`;

function escapeRegExp(value: string) {
    return value.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
}

export function creationMentionLabelRegExp(label: string) {
    return new RegExp(`@${escapeRegExp(label)}${MENTION_LABEL_BOUNDARY}`, "gu");
}

export function replaceCreationMentionLabel(value: string, label: string, replacement: string) {
    return value.replace(creationMentionLabelRegExp(label), replacement);
}

export function promptContainsResolvedReference(prompt: string, label: string, token: string) {
    if (token.endsWith("]") && prompt.includes(token)) return true;
    return new RegExp(`@${escapeRegExp(label)}${MENTION_LABEL_BOUNDARY}`, "u").test(prompt);
}

export function removeResolvedReferenceTokens(value: string, references: Array<{ label: string; token: string }>) {
    // 先处理更长的名字，避免删掉 @图片1 时把 @图片10 的前缀一起吃掉。
    const ordered = [...references].sort((left, right) => Math.max(right.label.length, right.token.length) - Math.max(left.label.length, left.token.length));
    return ordered.reduce((current, reference) => {
        const withoutToken = reference.token.endsWith("]") ? current.split(reference.token).join("") : current;
        return replaceCreationMentionLabel(withoutToken, reference.label, "");
    }, value);
}
