// @ts-nocheck
import { CanvasNodeAnnotationDialog, type CanvasImageAnnotationPayload } from "@yingce/components/canvas/canvas-node-annotation-dialog";
import { CanvasNodeCropDialog, type CanvasImageCropRect } from "@yingce/components/canvas/canvas-node-crop-dialog";
import { CanvasNodeMaskEditDialog, type CanvasImageMaskEditPayload } from "@yingce/components/canvas/canvas-node-mask-edit-dialog";
import { CanvasNodeUpscaleDialog, type CanvasImageUpscaleParams } from "@yingce/components/canvas/canvas-node-upscale-dialog";
import { CanvasNodeImageEditDialog, type CanvasImageEditPayload } from "@yingce/components/canvas/canvas-node-image-edit-dialog";
import { CanvasNodeLayerDecompositionDialog, type CanvasImageLayerDecompositionPayload } from "@yingce/components/canvas/canvas-node-layer-decomposition-dialog";
import { CanvasNodeTextEditDialog, type CanvasImageTextEditPayload } from "@yingce/components/canvas/canvas-node-text-edit-dialog";
import { ImageWorkbench, type ImageWorkbenchMode } from "@yingce/components/canvas/canvas-image-workbench";
import type { CanvasNodeData } from "@yingce/types/canvas";
import type { AiConfig } from "@yingce/stores/use-config-store";

type CanvasProjectMediaDialogsProps = {
    cropNode: CanvasNodeData | null;
    annotationNode: CanvasNodeData | null;
    annotationEditNode: CanvasNodeData | null;
    maskEditNode: CanvasNodeData | null;
    imageEditNode: CanvasNodeData | null;
    layerDecompositionNode: CanvasNodeData | null;
    textEditNode: CanvasNodeData | null;
    imageEditPreset?: "remove-background" | null;
    upscaleNode: CanvasNodeData | null;
    onCloseCrop: () => void;
    onCloseAnnotation: () => void;
    onCloseAnnotationEdit: () => void;
    onCloseMaskEdit: () => void;
    onCloseUpscale: () => void;
    onCloseImageEdit: () => void;
    onCloseLayerDecomposition: () => void;
    onCloseTextEdit: () => void;
    onCrop: (node: CanvasNodeData, crop: CanvasImageCropRect) => void;
    onAnnotate: (node: CanvasNodeData, dataUrl: string) => void;
    onAnnotationEdit: (node: CanvasNodeData, payload: CanvasImageAnnotationPayload) => void;
    onMaskEdit: (node: CanvasNodeData, payload: CanvasImageMaskEditPayload) => void;
    onUpscale: (node: CanvasNodeData, params: CanvasImageUpscaleParams) => void;
    onImageOperation: (node: CanvasNodeData, payload: CanvasImageEditPayload) => void;
    onLayerDecomposition: (node: CanvasNodeData, payload: CanvasImageLayerDecompositionPayload) => void;
    onDetectText: () => Promise<import("@yingce/components/canvas/canvas-node-text-edit-dialog").CanvasImageTextLine[]>;
    onTextEdit: (node: CanvasNodeData, payload: CanvasImageTextEditPayload) => void;
    config: AiConfig;
};

export function CanvasProjectMediaDialogs(props: CanvasProjectMediaDialogsProps) {
    const {
        cropNode,
        annotationNode,
        annotationEditNode,
        maskEditNode,
        imageEditNode,
        layerDecompositionNode,
        textEditNode,
        imageEditPreset,
        upscaleNode,
        onCloseCrop,
        onCloseAnnotation,
        onCloseAnnotationEdit,
        onCloseMaskEdit,
        onCloseUpscale,
        onCloseImageEdit,
        onCloseLayerDecomposition,
        onCloseTextEdit,
        onCrop,
        onAnnotate,
        onAnnotationEdit,
        onMaskEdit,
        onUpscale,
        onImageOperation,
        onLayerDecomposition,
        onDetectText,
        onTextEdit,
        config,
    } = props;

    const active = cropNode?.metadata?.content
        ? { mode: "crop" as ImageWorkbenchMode, onClose: onCloseCrop }
        : annotationNode?.metadata?.content
            ? { mode: "annotate" as ImageWorkbenchMode, onClose: onCloseAnnotation }
            : annotationEditNode?.metadata?.content
                ? { mode: "annotate-edit" as ImageWorkbenchMode, onClose: onCloseAnnotationEdit }
                : maskEditNode?.metadata?.content
                    ? { mode: "mask" as ImageWorkbenchMode, onClose: onCloseMaskEdit }
                    : upscaleNode?.metadata?.content
                        ? { mode: "upscale" as ImageWorkbenchMode, onClose: onCloseUpscale }
                        : imageEditNode?.metadata?.content
                            ? { mode: "edit" as ImageWorkbenchMode, onClose: onCloseImageEdit }
                            : layerDecompositionNode?.metadata?.content
                                ? { mode: "layers" as ImageWorkbenchMode, onClose: onCloseLayerDecomposition }
                                : textEditNode?.metadata?.content
                                    ? { mode: "text" as ImageWorkbenchMode, onClose: onCloseTextEdit }
                                    : null;

    if (!active) return null;

    return (
        <ImageWorkbench open mode={active.mode} onClose={active.onClose} title={imageEditPreset === "remove-background" && active.mode === "edit" ? "去除背景" : undefined}>
            {active.mode === "crop" && cropNode?.metadata?.content ? <CanvasNodeCropDialog embedded dataUrl={cropNode.metadata.content} open onClose={onCloseCrop} onConfirm={(crop) => onCrop(cropNode, crop)} /> : null}
            {active.mode === "annotate" && annotationNode?.metadata?.content ? <CanvasNodeAnnotationDialog embedded image={{ url: annotationNode.metadata.content, storageKey: annotationNode.metadata.storageKey }} open onClose={onCloseAnnotation} onConfirm={(dataUrl) => { if (typeof dataUrl === "string") onAnnotate(annotationNode, dataUrl); }} /> : null}
            {active.mode === "annotate-edit" && annotationEditNode?.metadata?.content ? <CanvasNodeAnnotationDialog embedded image={{ url: annotationEditNode.metadata.content, storageKey: annotationEditNode.metadata.storageKey }} editMode open onClose={onCloseAnnotationEdit} onConfirm={(payload) => { if (typeof payload !== "string") onAnnotationEdit(annotationEditNode, payload); }} /> : null}
            {active.mode === "mask" && maskEditNode?.metadata?.content ? <CanvasNodeMaskEditDialog embedded dataUrl={maskEditNode.metadata.content} config={{ ...config, model: maskEditNode.metadata.model || config.model, imageModel: maskEditNode.metadata.model || config.imageModel, size: maskEditNode.metadata.size || config.size, quality: maskEditNode.metadata.quality || config.quality, count: String(maskEditNode.metadata.count || config.count) }} open onClose={onCloseMaskEdit} onConfirm={(payload) => onMaskEdit(maskEditNode, payload)} /> : null}
            {active.mode === "upscale" && upscaleNode?.metadata?.content ? <CanvasNodeUpscaleDialog embedded dataUrl={upscaleNode.metadata.content} open onClose={onCloseUpscale} onConfirm={(params) => onUpscale(upscaleNode, params)} /> : null}
            {active.mode === "edit" && imageEditNode?.metadata?.content ? <CanvasNodeImageEditDialog embedded dataUrl={imageEditNode.metadata.content} preset={imageEditPreset} config={{ ...config, model: imageEditNode.metadata.model || config.model, imageModel: imageEditNode.metadata.model || config.imageModel, size: imageEditNode.metadata.size || config.size, quality: imageEditNode.metadata.quality || config.quality }} open onClose={onCloseImageEdit} onConfirm={(payload) => onImageOperation(imageEditNode, payload)} /> : null}
            {active.mode === "layers" && layerDecompositionNode?.metadata?.content ? <CanvasNodeLayerDecompositionDialog embedded dataUrl={layerDecompositionNode.metadata.content} config={{ ...config, model: layerDecompositionNode.metadata.model || config.model, imageModel: layerDecompositionNode.metadata.model || config.imageModel, size: layerDecompositionNode.metadata.size || config.size, quality: layerDecompositionNode.metadata.quality || config.quality }} open onClose={onCloseLayerDecomposition} onConfirm={(payload) => onLayerDecomposition(layerDecompositionNode, payload)} /> : null}
            {active.mode === "text" && textEditNode?.metadata?.content ? <CanvasNodeTextEditDialog embedded dataUrl={textEditNode.metadata.content} open onClose={onCloseTextEdit} onDetect={onDetectText} onConfirm={(payload) => onTextEdit(textEditNode, payload)} /> : null}
        </ImageWorkbench>
    );
}
