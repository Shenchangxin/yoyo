// @ts-nocheck
import type { ReactNode } from "react";

import { AppModal } from "@yingce/components/ui/product/app-modal";

export type ImageWorkbenchMode = "crop" | "split" | "upscale" | "mask" | "edit" | "text" | "annotate" | "annotate-edit" | "layers";

const TITLES: Record<ImageWorkbenchMode, string> = {
    crop: "裁剪",
    split: "切分",
    upscale: "调整尺寸",
    mask: "局部重绘",
    edit: "图片编辑",
    text: "文字编辑",
    annotate: "标注",
    "annotate-edit": "标注编辑",
    layers: "图层拆分",
};

export function ImageWorkbench({
    open,
    mode,
    onClose,
    children,
    title,
    width = "min(1120px, calc(100vw - 48px))",
}: {
    open: boolean;
    mode?: ImageWorkbenchMode;
    onClose: () => void;
    children: ReactNode;
    title?: ReactNode;
    width?: number | string;
}) {
    return (
        <AppModal
            open={open}
            title={title ?? (mode ? TITLES[mode] : null)}
            footer={null}
            onCancel={onClose}
            width={width}
            centered
            flush
            zIndex={400}
            className="sg-workbench"
            rootClassName="sg-workbench-root"
            destroyOnHidden
        >
            <div className="sg-image-workbench min-h-[min(68vh,760px)] p-4">{children}</div>
        </AppModal>
    );
}

export function ImageEditorShell({
    embedded,
    open,
    title,
    onClose,
    children,
    width,
}: {
    embedded?: boolean;
    open: boolean;
    title?: ReactNode;
    onClose: () => void;
    children: ReactNode;
    width?: number | string;
}) {
    if (embedded) return children;
    return (
        <ImageWorkbench open={open} title={title} onClose={onClose} width={width}>
            {children}
        </ImageWorkbench>
    );
}
