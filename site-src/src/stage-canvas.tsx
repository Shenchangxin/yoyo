import { useEffect, useRef } from "react";
import * as THREE from "three";

/** The picture inside a video shot. Light follows the episode, nothing else moves. */
export function ShotLight({ lit }: { lit: boolean }) {
  const ref = useRef<HTMLCanvasElement>(null);
  const litRef = useRef(lit);
  litRef.current = lit;

  useEffect(() => {
    const canvas = ref.current;
    if (!canvas) return;
    let renderer: THREE.WebGLRenderer;
    try {
      renderer = new THREE.WebGLRenderer({ canvas, antialias: true, alpha: false, powerPreference: "low-power" });
    } catch {
      return;
    }
    renderer.setClearColor(0x14110e, 1);
    const scene = new THREE.Scene();
    scene.background = new THREE.Color("#14110e");
    const camera = new THREE.PerspectiveCamera(32, 1, 0.1, 20);
    camera.position.set(0.15, 0.85, 2.35);
    camera.lookAt(0, 0.15, 0);

    const geometries: THREE.BufferGeometry[] = [];
    const materials: THREE.Material[] = [];
    const add = (geometry: THREE.BufferGeometry, material: THREE.Material, position: [number, number, number]) => {
      geometries.push(geometry);
      materials.push(material);
      const mesh = new THREE.Mesh(geometry, material);
      mesh.position.set(...position);
      scene.add(mesh);
      return mesh;
    };

    add(new THREE.BoxGeometry(2.4, 0.06, 1.15), new THREE.MeshStandardMaterial({ color: 0x3a2c22, roughness: 0.92 }), [0, 0, 0]);
    add(new THREE.CylinderGeometry(0.012, 0.014, 0.42, 12), new THREE.MeshStandardMaterial({ color: 0x5c4c3d, roughness: 0.8 }), [0.28, 0.24, 0.05]);
    const shade = add(
      new THREE.CylinderGeometry(0.09, 0.14, 0.1, 16, 1, true),
      new THREE.MeshStandardMaterial({ color: 0x8d7b64, roughness: 0.7, side: THREE.DoubleSide, emissive: new THREE.Color("#f3e2bc"), emissiveIntensity: 0 }),
      [0.28, 0.48, 0.05],
    ) as THREE.Mesh;
    const shadeMat = shade.material as THREE.MeshStandardMaterial;

    const ambient = new THREE.AmbientLight(0xf4f0e6, 0.18);
    const lamp = new THREE.PointLight(0xf3d7a4, 0, 3.2);
    lamp.position.set(0.28, 0.42, 0.05);
    scene.add(ambient, lamp);

    let frame = 0;
    let level = litRef.current ? 1 : 0;
    const reduce = window.matchMedia("(prefers-reduced-motion: reduce)").matches;
    const draw = () => {
      const w = canvas.clientWidth;
      const h = canvas.clientHeight;
      if (w < 2 || h < 2) return;
      const dpr = Math.min(window.devicePixelRatio || 1, 1.5);
      if (canvas.width !== Math.floor(w * dpr) || canvas.height !== Math.floor(h * dpr)) {
        renderer.setPixelRatio(dpr);
        renderer.setSize(w, h, false);
        camera.aspect = w / h;
        camera.updateProjectionMatrix();
      }
      const target = litRef.current ? 1 : 0;
      level += (target - level) * (reduce ? 1 : 0.08);
      lamp.intensity = 0.15 + level * 7.5;
      ambient.intensity = 0.16 + level * 0.22;
      shadeMat.emissiveIntensity = level * 0.55;
      shadeMat.color.set(level > 0.5 ? "#f4e2bc" : "#8d7b64");
      renderer.render(scene, camera);
    };
    const loop = () => {
      frame = window.requestAnimationFrame(loop);
      draw();
      if (reduce) window.cancelAnimationFrame(frame);
    };
    loop();
    return () => {
      window.cancelAnimationFrame(frame);
      geometries.forEach((geometry) => geometry.dispose());
      materials.forEach((material) => material.dispose());
      renderer.dispose();
    };
  }, []);

  return <canvas ref={ref} className="shot-light" />;
}
