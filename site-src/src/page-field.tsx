/**
 * 拥有：整页背后的玻璃碎片场。WebGPU 起不来时什么都不画，页面仍是纸色。
 * 不拥有：标题、导航和各段正文。
 */

import { memo, useEffect, useState } from "react";
import AeroShards from "./bits/AeroShards";

type Theme = "light" | "dark";

/** useCompact 在窄屏上少画一些碎片。 */
function useCompact() {
  const [compact, setCompact] = useState(() => window.matchMedia("(max-width: 759px)").matches);
  useEffect(() => {
    const media = window.matchMedia("(max-width: 759px)");
    const onChange = () => setCompact(media.matches);
    media.addEventListener("change", onChange);
    return () => media.removeEventListener("change", onChange);
  }, []);
  return compact;
}

/** PageField 把 AeroShards 固定在整页背后。theme 决定浅色玻璃还是夜色玻璃。 */
export const PageField = memo(function PageField({ theme }: { theme: Theme }) {
  const [failed, setFailed] = useState(false);
  const compact = useCompact();
  if (failed) return null;

  const dark = theme === "dark";
  return (
    <div className="page-field" aria-hidden="true">
      <AeroShards
        shardColor={dark ? "#B7C0C8" : "#7A6856"}
        accentColor={dark ? "#E6C79B" : "#C6A36A"}
        backgroundColor="transparent"
        lightMode={!dark}
        placement="full"
        material="glass"
        markers="plus"
        count={compact ? 720 : 1200}
        shardSize={1.05}
        spread={1.2}
        twist={0.9}
        billow={1}
        speed={0.62}
        scale={1}
        depthOfField={0.7}
        glow={dark ? 0.4 : 0.22}
        grain={0.035}
        vignette={0.22}
        intro
        mouseInteraction
        mouseStrength={0.42}
        parallax={0.35}
        pulseOnClick
        holdToGather
        dpr={compact ? 1.15 : 1.35}
        onError={() => setFailed(true)}
      />
    </div>
  );
});
