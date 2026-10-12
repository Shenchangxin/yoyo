import { useEffect, useRef } from "react";
import * as THREE from "three";

const VERT = `
precision mediump float;
in float aScale;
in float aKind;
in float aIndex;
uniform float uTime;
uniform float uChapter;
out float vKind;
out float vFade;
void main() {
  vec3 p = position;
  float ang = atan(p.z, p.x);
  float radius = length(p.xz);
  float spin = 0.16;
  float lift = 0.11;
  if (aKind > 0.5) {
    if (uChapter > 0.45 && uChapter < 1.55) {
      radius = mod(aIndex, 2.0) < 0.5 ? 1.18 : 2.08;
      spin = 0.07;
    } else if (uChapter >= 1.55 && uChapter < 2.55) {
      radius = 1.9;
      spin = 0.045;
      lift = 0.02;
    } else if (uChapter >= 2.55) {
      radius = mix(1.35, 0.72, smoothstep(2.55, 3.6, uChapter));
      spin = 0.035;
    }
    ang += uTime * spin;
    p.x = cos(ang) * radius;
    p.z = sin(ang) * radius * 0.62;
    p.y = sin(ang * 2.0) * lift;
  } else {
    p.y += sin(uTime * 0.22 + p.x * 1.4) * 0.06;
    p.x += sin(uTime * 0.08 + aIndex) * 0.04 * uChapter;
  }
  vec4 mv = modelViewMatrix * vec4(p, 1.0);
  gl_PointSize = clamp(aScale * (80.0 / max(1.0, -mv.z)), 1.0, 11.0);
  gl_Position = projectionMatrix * mv;
  vKind = aKind;
  vFade = smoothstep(6.5, 2.2, -mv.z);
}
`;

const FRAG = `
precision mediump float;
in float vKind;
in float vFade;
uniform float uChapter;
out vec4 fragColor;
void main() {
  vec2 uv = gl_PointCoord - vec2(0.5);
  float d = length(uv);
  if (d > 0.5) discard;
  float soft = smoothstep(0.5, 0.08, d);
  vec3 dust = vec3(0.55, 0.54, 0.50);
  vec3 evidence = vec3(0.36, 0.60, 0.84);
  vec3 lamp = vec3(0.83, 0.54, 0.42);
  vec3 color = dust;
  float alpha = 0.28;
  if (vKind > 1.5) {
    color = lamp;
    alpha = 0.9;
  } else if (vKind > 0.5) {
    color = evidence;
    alpha = uChapter > 2.7 ? 0.28 : 0.72;
  }
  fragColor = vec4(color, soft * alpha * vFade);
}
`;

function chapterOf(zone: string) {
  if (zone === "model") return 1;
  if (zone === "see") return 2;
  if (zone === "trust") return 3;
  if (zone === "doors" || zone === "get") return 4;
  return 0;
}

export function StoneField({ zone }: { zone: string }) {
  const host = useRef<HTMLDivElement>(null);
  const zoneRef = useRef(zone);
  zoneRef.current = zone;

  useEffect(() => {
    const el = host.current;
    if (!el) return;

    let renderer: THREE.WebGLRenderer;
    try {
      renderer = new THREE.WebGLRenderer({
        antialias: false,
        alpha: true,
        powerPreference: "low-power",
      });
    } catch {
      return;
    }
    if (!renderer.getContext()) {
      renderer.dispose();
      return;
    }

    const reduce = window.matchMedia("(prefers-reduced-motion: reduce)").matches;
    renderer.setPixelRatio(Math.min(window.devicePixelRatio || 1, 1.5));
    renderer.setClearColor(0x000000, 0);
    el.appendChild(renderer.domElement);

    const scene = new THREE.Scene();
    const camera = new THREE.PerspectiveCamera(42, 1, 0.1, 20);
    camera.position.set(0, 1.15, 3.4);

    const ringN = 88;
    const dustN = 220;
    const total = ringN + dustN;
    const positions = new Float32Array(total * 3);
    const scales = new Float32Array(total);
    const kinds = new Float32Array(total);

    for (let i = 0; i < ringN; i += 1) {
      const a = (i / ringN) * Math.PI * 2;
      const r = 1.62;
      positions[i * 3] = Math.cos(a) * r;
      positions[i * 3 + 1] = 0;
      positions[i * 3 + 2] = Math.sin(a) * r * 0.62;
      const mark = i % 22 === 0;
      scales[i] = mark ? 4.2 : 2.1;
      kinds[i] = mark ? 2 : 1;
    }
    for (let i = 0; i < dustN; i += 1) {
      const n = ringN + i;
      positions[n * 3] = (Math.random() - 0.5) * 6.2;
      positions[n * 3 + 1] = (Math.random() - 0.5) * 2.4;
      positions[n * 3 + 2] = (Math.random() - 0.5) * 3.2 - 0.4;
      scales[n] = 0.7 + Math.random() * 0.8;
      kinds[n] = 0;
    }

    const geometry = new THREE.BufferGeometry();
    geometry.setAttribute("position", new THREE.BufferAttribute(positions, 3));
    geometry.setAttribute("aScale", new THREE.BufferAttribute(scales, 1));
    geometry.setAttribute("aKind", new THREE.BufferAttribute(kinds, 1));
    const indices = new Float32Array(total);
    for (let i = 0; i < total; i += 1) indices[i] = i;
    geometry.setAttribute("aIndex", new THREE.BufferAttribute(indices, 1));
    const material = new THREE.ShaderMaterial({
      glslVersion: THREE.GLSL3,
      uniforms: { uTime: { value: 0 }, uChapter: { value: 0 } },
      vertexShader: VERT,
      fragmentShader: FRAG,
      transparent: true,
      depthWrite: false,
    });
    const points = new THREE.Points(geometry, material);
    points.position.y = 0.35;
    points.rotation.x = 0.72;
    scene.add(points);

    const pointer = { x: 0, y: 0 };
    const onMove = (event: PointerEvent) => {
      pointer.x = (event.clientX / Math.max(window.innerWidth, 1) - 0.5) * 2;
      pointer.y = (event.clientY / Math.max(window.innerHeight, 1) - 0.5) * 2;
    };
    window.addEventListener("pointermove", onMove);

    let frame = 0;
    let running = !reduce;
    let chapter = 0;
    const clock = new THREE.Clock();
    const draw = () => {
      const targetY = pointer.x * 0.18;
      const targetX = 0.72 + pointer.y * 0.05;
      points.rotation.y += (targetY - points.rotation.y) * 0.045;
      points.rotation.x += (targetX - points.rotation.x) * 0.045;
      const next = chapterOf(zoneRef.current);
      chapter += (next - chapter) * (reduce ? 1 : 0.045);
      material.uniforms.uChapter.value = chapter;
      material.uniforms.uTime.value = reduce ? 0.4 : clock.getElapsedTime();
      renderer.render(scene, camera);
    };
    const loop = () => {
      frame = window.requestAnimationFrame(loop);
      if (running) draw();
    };

    const fit = () => {
      const width = el.clientWidth;
      const height = el.clientHeight;
      if (width < 2 || height < 2) return;
      camera.aspect = width / height;
      camera.updateProjectionMatrix();
      renderer.setSize(width, height, false);
      draw();
    };
    const observer = new ResizeObserver(fit);
    observer.observe(el);
    fit();

    const onVis = () => {
      running = document.visibilityState === "visible" && !reduce;
    };
    document.addEventListener("visibilitychange", onVis);
    if (!reduce) loop();

    return () => {
      window.cancelAnimationFrame(frame);
      document.removeEventListener("visibilitychange", onVis);
      observer.disconnect();
      window.removeEventListener("pointermove", onMove);
      geometry.dispose();
      material.dispose();
      renderer.dispose();
      renderer.domElement.remove();
    };
  }, []);

  return <div className="page-field" ref={host} aria-hidden="true" />;
}
