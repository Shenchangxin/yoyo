package video

import (
	"os"
	"path/filepath"
)

func (e *Engine) writeRemotionProject() error {
	dir := e.remotionDir()
	if dir == "" {
		return os.ErrInvalid
	}
	if err := os.MkdirAll(filepath.Join(dir, "src"), 0o755); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Join(dir, "public"), 0o755); err != nil {
		return err
	}
	files := map[string]string{
		"package.json":       remotionPackageJSON,
		".npmrc":             remotionNPMRC,
		"tsconfig.json":      remotionTSConfig,
		"remotion.config.ts": remotionConfigTS,
		"src/index.ts":       remotionIndexTS,
		"src/Root.tsx":       remotionRootTSX,
		"src/TitleCard.tsx":  remotionTitleCardTSX,
	}
	for rel, body := range files {
		if err := os.WriteFile(filepath.Join(dir, filepath.FromSlash(rel)), []byte(body), 0o644); err != nil {
			return err
		}
	}
	return nil
}

const remotionPackageJSON = `{
  "name": "yoyo-remotion",
  "private": true,
  "version": "1.0.0",
  "scripts": {
    "render": "remotion render src/index.ts TitleCard"
  },
  "dependencies": {
    "@remotion/cli": "4.0.241",
    "react": "18.3.1",
    "react-dom": "18.3.1",
    "remotion": "4.0.241"
  }
}
`

const remotionNPMRC = `fund=false
audit=false
progress=false
`

const remotionTSConfig = `{
  "compilerOptions": {
    "target": "ES2018",
    "module": "commonjs",
    "jsx": "react-jsx",
    "strict": true,
    "noEmit": true,
    "skipLibCheck": true,
    "esModuleInterop": true
  }
}
`

const remotionConfigTS = `import { Config } from "@remotion/cli/config";
Config.setVideoImageFormat("jpeg");
Config.setOverwriteOutput(true);
`

const remotionIndexTS = `import { registerRoot } from "remotion";
import { RemotionRoot } from "./Root";
registerRoot(RemotionRoot);
`

const remotionRootTSX = `import React from "react";
import { Composition } from "remotion";
import { TitleCard } from "./TitleCard";

export const RemotionRoot: React.FC = () => {
  return (
    <Composition
      id="TitleCard"
      component={TitleCard}
      durationInFrames={90}
      fps={30}
      width={1080}
      height={1920}
      defaultProps={{
        src: "input.mp4",
        title: "Title",
        durationInFrames: 90,
        width: 1080,
        height: 1920,
      }}
      calculateMetadata={({ props }) => ({
        durationInFrames: Math.max(1, Number(props.durationInFrames) || 90),
        fps: 30,
        width: Math.max(16, Number(props.width) || 1080),
        height: Math.max(16, Number(props.height) || 1920),
      })}
    />
  );
};
`

const remotionTitleCardTSX = `import React from "react";
import { AbsoluteFill, OffthreadVideo, staticFile } from "remotion";

export type TitleCardProps = {
  src: string;
  title: string;
  durationInFrames: number;
  width: number;
  height: number;
};

export const TitleCard: React.FC<TitleCardProps> = ({ src, title }) => {
  const file = src.includes("://") || src.startsWith("/") ? src : staticFile(src.replace(/^public\//, ""));
  return (
    <AbsoluteFill style={{ backgroundColor: "#000" }}>
      <OffthreadVideo src={file} style={{ width: "100%", height: "100%", objectFit: "cover" }} />
      <AbsoluteFill
        style={{
          justifyContent: "flex-end",
          padding: 48,
          backgroundImage: "linear-gradient(to top, rgba(0,0,0,0.72) 0%, rgba(0,0,0,0) 46%)",
        }}
      >
        <div
          style={{
            color: "white",
            fontSize: 56,
            fontWeight: 700,
            lineHeight: 1.15,
            letterSpacing: "-0.03em",
            textShadow: "0 2px 18px rgba(0,0,0,0.45)",
            maxWidth: "92%",
            fontFamily: "ui-sans-serif, system-ui, sans-serif",
          }}
        >
          {title}
        </div>
      </AbsoluteFill>
    </AbsoluteFill>
  );
};
`
