// @ts-nocheck
import { type ReactNode } from "react";

/** Live2D is not part of the Yoyo desktop canvas island. */
export function Live2DAvatar({
    fallback,
    onReady,
}: {
    url?: string;
    width?: number;
    height?: number;
    reducedMotion?: boolean;
    fallback?: ReactNode;
    onReady?: () => void;
    onError?: (error: Error) => void;
}) {
    onReady?.();
    return <>{fallback || null}</>;
}
