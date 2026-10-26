import { useEffect, useRef } from "react";
import * as THREE from "three";
import { RoundedBoxGeometry } from "three/examples/jsm/geometries/RoundedBoxGeometry.js";
import { RoomEnvironment } from "three/examples/jsm/environments/RoomEnvironment.js";

type Theme = "light" | "dark";

/** SpaceStill 画一张安静的桌面：一块台面，前面一张纸，后面一盏灯。theme 只改颜色。 */
export function SpaceStill({ theme }: { theme: Theme }) {
  const rootRef = useRef<HTMLDivElement>(null);
  const canvasRef = useRef<HTMLCanvasElement>(null);
  const themeRef = useRef(theme);
  const drawRef = useRef<() => void>(() => {});
  themeRef.current = theme;

  useEffect(() => {
    const root = rootRef.current;
    const canvas = canvasRef.current;
    if (!root || !canvas) return;

    let renderer: THREE.WebGLRenderer;
    try {
      renderer = new THREE.WebGLRenderer({
        canvas,
        antialias: true,
        alpha: true,
        powerPreference: "high-performance",
        premultipliedAlpha: false,
      });
    } catch {
      return;
    }

    renderer.setClearColor(0x000000, 0);
    renderer.outputColorSpace = THREE.SRGBColorSpace;
    renderer.toneMapping = THREE.ACESFilmicToneMapping;
    renderer.shadowMap.enabled = true;
    renderer.shadowMap.type = THREE.PCFSoftShadowMap;

    const scene = new THREE.Scene();
    const camera = new THREE.PerspectiveCamera(30, 1, 0.1, 40);
    const world = new THREE.Group();
    scene.add(world);

    const geometries: THREE.BufferGeometry[] = [];
    const materials: THREE.Material[] = [];
    const textures: THREE.Texture[] = [];

    /** take 调用 push。geometry 放进 geometries。 */
    const take = <T extends THREE.BufferGeometry>(geometry: T) => {
      geometries.push(geometry);
      return geometry;
    };
    /** shade 调用 push。material 放进 materials。 */
    const shade = <T extends THREE.Material>(material: T) => {
      materials.push(material);
      return material;
    };

    const faceTex = document.createElement("canvas");
    faceTex.width = 512;
    faceTex.height = 680;
    const faceCtx = faceTex.getContext("2d");
    if (faceCtx) {
      const wash = faceCtx.createLinearGradient(0, 0, 0, 680);
      wash.addColorStop(0, "#3f3c38");
      wash.addColorStop(0.45, "#1a1917");
      wash.addColorStop(1, "#0e0d0c");
      faceCtx.fillStyle = wash;
      faceCtx.fillRect(0, 0, 512, 680);
      const sheen = faceCtx.createLinearGradient(0, 0, 280, 160);
      sheen.addColorStop(0, "rgba(255,244,224,0.28)");
      sheen.addColorStop(0.35, "rgba(255,244,224,0)");
      faceCtx.fillStyle = sheen;
      faceCtx.fillRect(0, 0, 512, 680);
    }
    const faceMap = new THREE.CanvasTexture(faceTex);
    faceMap.colorSpace = THREE.SRGBColorSpace;
    textures.push(faceMap);

    const blobTex = document.createElement("canvas");
    blobTex.width = 256;
    blobTex.height = 256;
    const blobCtx = blobTex.getContext("2d");
    if (blobCtx) {
      const rad = blobCtx.createRadialGradient(128, 128, 16, 128, 128, 122);
      rad.addColorStop(0, "rgba(70,48,28,0.55)");
      rad.addColorStop(0.55, "rgba(70,48,28,0.16)");
      rad.addColorStop(1, "rgba(70,48,28,0)");
      blobCtx.fillStyle = rad;
      blobCtx.fillRect(0, 0, 256, 256);
    }
    const blobMap = new THREE.CanvasTexture(blobTex);
    textures.push(blobMap);

    const plinthMat = shade(new THREE.MeshStandardMaterial({ color: "#d9cbb6", roughness: 0.86, metalness: 0.02 }));
    const paperMat = shade(new THREE.MeshStandardMaterial({ color: "#fffaf3", roughness: 0.94, metalness: 0 }));
    const sphereMat = shade(
      new THREE.MeshPhysicalMaterial({ color: "#d7cec2", roughness: 0.42, metalness: 0.02, clearcoat: 0.28, clearcoatRoughness: 0.4 }),
    );
    const screenMat = shade(
      new THREE.MeshPhysicalMaterial({ color: "#2a2826", roughness: 0.46, metalness: 0.18, clearcoat: 0.22, clearcoatRoughness: 0.4 }),
    );
    const faceMat = shade(new THREE.MeshStandardMaterial({ map: faceMap, roughness: 0.46, metalness: 0.08 }));
    const stemMat = shade(new THREE.MeshStandardMaterial({ color: "#a68455", roughness: 0.45, metalness: 0.35 }));
    const shadeMat = shade(
      new THREE.MeshStandardMaterial({
        color: "#f0d3a8",
        roughness: 0.42,
        metalness: 0.05,
        emissive: new THREE.Color("#ffd7a4"),
        emissiveIntensity: 0.55,
        side: THREE.DoubleSide,
      }),
    );
    const blobMat = shade(new THREE.MeshBasicMaterial({ map: blobMap, transparent: true, depthWrite: false, opacity: 0.85 }));
    const shadowMat = shade(new THREE.ShadowMaterial({ color: "#6a5038", opacity: 0.28 }));

    /** mesh 调用 Mesh。geometry 传给 THREE.Mesh。material 传给 THREE.Mesh。parent 调用 add。 */
    const mesh = (
      geometry: THREE.BufferGeometry,
      material: THREE.Material,
      parent: THREE.Object3D = world,
    ) => {
      const item = new THREE.Mesh(geometry, material);
      item.castShadow = true;
      item.receiveShadow = true;
      parent.add(item);
      return item;
    };

    const blob = mesh(take(new THREE.PlaneGeometry(1.45, 0.85)), blobMat);
    blob.rotation.x = -Math.PI / 2;
    blob.position.set(0.1, 0.004, 0);
    blob.castShadow = false;
    blob.receiveShadow = false;

    const catcher = mesh(take(new THREE.PlaneGeometry(2.4, 1.5)), shadowMat);
    catcher.rotation.x = -Math.PI / 2;
    catcher.position.y = 0.001;
    catcher.castShadow = false;
    catcher.receiveShadow = true;

    const plinth = mesh(take(new RoundedBoxGeometry(1.48, 0.08, 0.92, 4, 0.04)), plinthMat);
    plinth.position.set(0.1, 0.04, 0);

    const screen = new THREE.Group();
    screen.position.set(0.12, 0.52, -0.04);
    screen.rotation.set(0.04, -0.32, 0);
    world.add(screen);
    mesh(take(new RoundedBoxGeometry(0.68, 0.9, 0.04, 3, 0.018)), screenMat, screen);
    const face = mesh(take(new RoundedBoxGeometry(0.58, 0.78, 0.008, 2, 0.006)), faceMat, screen);
    face.position.z = 0.018;
    face.castShadow = false;

    const paper = mesh(take(new RoundedBoxGeometry(0.42, 0.3, 0.012, 2, 0.006)), paperMat);
    paper.position.set(-0.12, 0.26, 0.16);
    paper.rotation.set(0.1, 0.42, -0.03);

    const sphere = mesh(take(new THREE.SphereGeometry(0.07, 48, 32)), sphereMat);
    sphere.position.set(0.34, 0.11, 0.16);

    const stem = mesh(take(new THREE.CylinderGeometry(0.01, 0.012, 0.28, 16)), stemMat);
    stem.position.set(0.46, 0.22, -0.16);
    const lampShade = mesh(take(new THREE.CylinderGeometry(0.07, 0.11, 0.09, 28, 1, true)), shadeMat);
    lampShade.position.set(0.46, 0.4, -0.16);

    const hemi = new THREE.HemisphereLight("#fff8ee", "#d9cbb6", 0.7);
    const key = new THREE.DirectionalLight("#fff4e6", 2.7);
    key.position.set(-2.6, 4.1, 2.6);
    key.castShadow = true;
    key.shadow.mapSize.set(1024, 1024);
    key.shadow.radius = 5;
    key.shadow.bias = -0.0004;
    key.shadow.normalBias = 0.025;
    const shadowCam = key.shadow.camera;
    shadowCam.near = 0.4;
    shadowCam.far = 12;
    shadowCam.left = -2.4;
    shadowCam.right = 2.4;
    shadowCam.top = 2.4;
    shadowCam.bottom = -2.4;
    const fill = new THREE.DirectionalLight("#d5e3f2", 0.55);
    fill.position.set(2.8, 1.6, 1.4);
    const rim = new THREE.DirectionalLight("#fffaf4", 0.7);
    rim.position.set(-0.6, 2.2, -2.8);
    const lampLight = new THREE.PointLight("#ffd2a2", 2.4, 1.8, 2);
    lampLight.position.set(0.46, 0.36, -0.16);
    scene.add(hemi, key, fill, rim, lampLight);

    let envTex: THREE.Texture | null = null;
    try {
      const pmrem = new THREE.PMREMGenerator(renderer);
      const room = new RoomEnvironment();
      envTex = pmrem.fromScene(room, 0.04).texture;
      scene.environment = envTex;
      room.traverse((obj) => {
        const item = obj as THREE.Mesh;
        item.geometry?.dispose();
        const mat = item.material;
        if (Array.isArray(mat)) mat.forEach((entry) => entry.dispose());
        else mat?.dispose();
      });
      pmrem.dispose();
    } catch {
      envTex = null;
    }

    let painted = "";
    /** applyTheme 调用 set。next 与 painted、dark 比较是否相等，赋给 painted。 */
    const applyTheme = (next: Theme) => {
      if (painted === next) return;
      painted = next;
      const dark = next === "dark";
      plinthMat.color.set(dark ? "#1c1b19" : "#d9cbb6");
      paperMat.color.set(dark ? "#3a3631" : "#fffdf8");
      sphereMat.color.set(dark ? "#c4bbae" : "#f4efe6");
      screenMat.color.set(dark ? "#141312" : "#2c2a27");
      stemMat.color.set(dark ? "#8d734c" : "#a68455");
      shadeMat.color.set(dark ? "#e7c79a" : "#f0d3a8");
      shadeMat.emissiveIntensity = dark ? 0.9 : 0.5;
      blobMat.opacity = dark ? 0.7 : 0.85;
      shadowMat.opacity = dark ? 0.48 : 0.26;
      shadowMat.color.set(dark ? "#000000" : "#6a5038");
      key.intensity = dark ? 1.15 : 1.7;
      fill.intensity = dark ? 0.22 : 0.48;
      hemi.intensity = dark ? 0.22 : 0.62;
      hemi.color.set(dark ? "#3a3834" : "#fff8ee");
      hemi.groundColor.set(dark ? "#141311" : "#d9cbb6");
      lampLight.intensity = dark ? 4.5 : 2.8;
      scene.environmentIntensity = dark ? 0.4 : 0.72;
      renderer.toneMappingExposure = dark ? 0.86 : 1.02;
    };

    const reduce = window.matchMedia("(prefers-reduced-motion: reduce)").matches;
    let aimY = 0;
    let aimX = 0;
    let looping = false;
    let visible = true;
    let lastW = 0;
    let lastH = 0;
    let lastDpr = 0;

    const resize = () => {
      const w = canvas.clientWidth;
      const h = canvas.clientHeight;
      if (w < 2 || h < 2) return false;
      const dpr = Math.min(window.devicePixelRatio || 1, w < 700 ? 1.5 : 1.75);
      if (w === lastW && h === lastH && dpr === lastDpr) return true;
      lastW = w;
      lastH = h;
      lastDpr = dpr;
      renderer.setPixelRatio(dpr);
      renderer.setSize(w, h, false);
      camera.aspect = w / h;
      camera.fov = camera.aspect < 1 ? 30 : 24;
      const dist = camera.aspect < 0.9 ? 4.15 : camera.aspect < 1.2 ? 3.55 : 3.15;
      camera.position.set(0.42, 0.72, dist);
      camera.lookAt(0.14, 0.34, 0);
      camera.updateProjectionMatrix();
      return true;
    };

    const draw = () => {
      applyTheme(themeRef.current);
      if (!resize()) return;
      if (!reduce && visible) {
        const t = performance.now() * 0.001;
        const y = aimY + Math.sin(t * 0.32) * 0.045;
        const x = aimX + Math.sin(t * 0.24) * 0.012;
        world.rotation.y += (y - world.rotation.y) * 0.055;
        world.rotation.x += (x - world.rotation.x) * 0.055;
        world.position.y = Math.sin(t * 0.42) * 0.012;
      }
      renderer.render(scene, camera);
      root.classList.add("is-live");
    };
    drawRef.current = draw;

    const loop = () => {
      if (!looping) return;
      draw();
      window.requestAnimationFrame(loop);
    };
    const start = () => {
      if (looping || reduce) return;
      looping = true;
      window.requestAnimationFrame(loop);
    };
    const stop = () => {
      looping = false;
    };

    /** onPointer 调用 getBoundingClientRect。event 读取 clientX、clientY。 */
    const onPointer = (event: PointerEvent) => {
      const rect = root.getBoundingClientRect();
      if (rect.width < 2) return;
      const nx = (event.clientX - rect.left) / rect.width - 0.5;
      const ny = (event.clientY - rect.top) / rect.height - 0.5;
      const near = Math.abs(nx) < 0.85 && ny > -0.9 && ny < 1.15;
      aimY = near ? nx * 0.32 : 0;
      aimX = near ? ny * -0.08 : 0;
    };

    const io = new IntersectionObserver(([entry]) => {
      visible = !!entry?.isIntersecting;
      if (visible) start();
      else stop();
    });
    io.observe(root);

    const onVis = () => {
      if (document.hidden) stop();
      else if (visible) start();
    };

    resize();
    draw();
    if (!reduce) {
      window.addEventListener("pointermove", onPointer, { passive: true });
      document.addEventListener("visibilitychange", onVis);
      start();
    }

    return () => {
      stop();
      io.disconnect();
      window.removeEventListener("pointermove", onPointer);
      document.removeEventListener("visibilitychange", onVis);
      drawRef.current = () => {};
      geometries.forEach((geometry) => geometry.dispose());
      materials.forEach((material) => material.dispose());
      textures.forEach((texture) => texture.dispose());
      envTex?.dispose();
      renderer.dispose();
      root.classList.remove("is-live");
    };
  }, []);

  useEffect(() => {
    drawRef.current();
  }, [theme]);

  return (
    <div className="space-still" ref={rootRef}>
      <div className="space-fallback" aria-hidden="true">
        <span className="fb-lamp" />
        <span className="fb-screen" />
        <span className="fb-paper" />
        <span className="fb-sphere" />
        <span className="fb-plinth" />
      </div>
      <canvas ref={canvasRef} className="space-canvas" />
    </div>
  );
}
