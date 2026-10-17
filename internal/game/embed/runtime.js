(function () {
  const spec = window.__YOYO_GAME__ || {};
  const canvas = document.getElementById("stage");
  const hud = document.getElementById("hud");
  const outcomeEl = document.getElementById("outcome");
  const restartBtn = document.getElementById("restart");
  const keys = Object.create(null);
  const state = {
    ready: false,
    inputSeen: false,
    loopTick: 0,
    outcome: "",
    score: 0,
    collected: 0,
    hp: 3,
    flags: {},
    inventory: [],
  };
  const player = {
    x: (spec.player && spec.player.position && spec.player.position[0]) || 0,
    y: (spec.player && spec.player.position && spec.player.position[1]) || 0,
    z: (spec.player && spec.player.position && spec.player.position[2]) || 0,
    yaw: (spec.player && spec.player.yaw) || 0,
    r: (spec.player && spec.player.radius) || 0.42,
    h: (spec.player && spec.player.height) || 1.7,
    speed: (spec.player && spec.player.speed) || 7,
  };
  const start = { x: player.x, y: player.y, z: player.z, yaw: player.yaw };
  const need = spec.need_collect || 0;
  const camMode = spec.camera || "tps";
  const elements = (spec.elements || []).map((el) => ({ ...el, gone: false }));
  const terrain = spec.scene || { width: 8, depth: 8, cell_size: 2, heights: [], surfaces: [] };
  let last = performance.now();
  let pointer = false;
  let mx = 0;

  function heightAt(x, z) {
    const w = terrain.width || 1;
    const d = terrain.depth || 1;
    const cs = terrain.cell_size || 2;
    const gx = x / cs + w / 2;
    const gz = z / cs + d / 2;
    const ix = Math.max(0, Math.min(w - 1, Math.floor(gx)));
    const iz = Math.max(0, Math.min(d - 1, Math.floor(gz)));
    const i = iz * w + ix;
    const h = (terrain.heights && terrain.heights[i]) || 0;
    const surf = (terrain.surfaces && terrain.surfaces[i]) || "ground";
    return { h: Math.max(0, h), surf };
  }

  function reset() {
    player.x = start.x;
    player.y = start.y;
    player.z = start.z;
    player.yaw = start.yaw;
    state.outcome = "";
    state.score = 0;
    state.collected = 0;
    state.hp = 3;
    state.flags = {};
    state.inventory = [];
    elements.forEach((el) => { el.gone = false; });
    if (outcomeEl) outcomeEl.textContent = "";
    paintHud();
  }

  function paintHud() {
    if (!hud) return;
    const obj = spec.objective || "Collect and reach the goal.";
    hud.innerHTML = `<div id="objective">${obj}</div><div id="score">score ${state.score} · relics ${state.collected}/${need} · hp ${state.hp}</div>`;
  }

  function applyDelta(d) {
    if (!d) return;
    if (d.score) state.score += d.score;
    if (d.hp) state.hp += d.hp;
    if (d.flag) state.flags[d.flag] = true;
    if (d.inventory) state.inventory.push(d.inventory);
    if (d.teleport && d.teleport_to) {
      player.x = d.teleport_to[0];
      player.y = d.teleport_to[1];
      player.z = d.teleport_to[2];
    }
    if (d.win) finish("win");
    if (d.fail || state.hp <= 0) finish("fail");
  }

  function finish(kind) {
    if (state.outcome) return;
    state.outcome = kind;
    if (outcomeEl) {
      outcomeEl.textContent = kind === "win" ? "WIN" : "FAIL";
      outcomeEl.dataset.state = kind;
    }
  }

  function fire(trigger, id) {
    (spec.interactions || []).forEach((it) => {
      if (it.trigger !== trigger) return;
      if (it.participants && it.participants.length && id && !it.participants.includes(id)) return;
      if (it.precondition === "all_collected" && state.collected < need) return;
      applyDelta(it.delta || {});
    });
  }

  function collideMove(nx, nz) {
    const here = heightAt(nx, nz);
    if (here.surf === "water") {
      fire("enter", "water");
      applyDelta({ fail: true });
      return;
    }
    for (const el of elements) {
      if (el.gone || !el.solid) continue;
      const p = el.position || [0, 0, 0];
      const s = el.size || [1, 1, 1];
      if (Math.abs(nx - p[0]) < (s[0] / 2 + player.r) && Math.abs(nz - p[2]) < (s[2] / 2 + player.r)) {
        return;
      }
    }
    player.x = nx;
    player.z = nz;
    player.y = here.h;
  }

  function step(dt) {
    if (state.outcome) return;
    let mxw = 0, mzw = 0;
    if (keys.KeyW || keys.ArrowUp) { mxw += Math.sin(player.yaw); mzw += Math.cos(player.yaw); }
    if (keys.KeyS || keys.ArrowDown) { mxw -= Math.sin(player.yaw); mzw -= Math.cos(player.yaw); }
    if (keys.KeyA || keys.ArrowLeft) { mxw -= Math.cos(player.yaw); mzw += Math.sin(player.yaw); }
    if (keys.KeyD || keys.ArrowRight) { mxw += Math.cos(player.yaw); mzw -= Math.sin(player.yaw); }
    const len = Math.hypot(mxw, mzw);
    if (len > 0) {
      collideMove(player.x + (mxw / len) * player.speed * dt, player.z + (mzw / len) * player.speed * dt);
    }
    for (const el of elements) {
      if (el.gone) continue;
      const p = el.position || [0, 0, 0];
      const reach = 1.15 + ((el.size && el.size[0]) || 1) * 0.25;
      if (Math.hypot(player.x - p[0], player.z - p[2]) > reach) continue;
      if (el.type === "collectible" || el.type === "item_pickup") {
        el.gone = true;
        state.collected += 1;
        fire("collect", el.id);
      } else if (el.type === "hazard_zone") {
        fire("enter", el.id);
        applyDelta({ hp: -3, fail: true });
      } else if (el.type === "goal_area") {
        fire("enter", el.id);
        if (state.collected >= need) finish("win");
      }
    }
  }

  function project(x, y, z, cam) {
    const dx = x - cam.x, dy = y - cam.y, dz = z - cam.z;
    const cy = Math.cos(-cam.yaw), sy = Math.sin(-cam.yaw);
    const rx = dx * cy - dz * sy;
    const rz = dx * sy + dz * cy;
    const cp = Math.cos(-cam.pitch), sp = Math.sin(-cam.pitch);
    const ry = dy * cp - rz * sp;
    const rz2 = dy * sp + rz * cp;
    const depth = Math.max(0.35, rz2);
    const f = (canvas.height * 0.72) / depth;
    return { x: canvas.width / 2 + rx * f, y: canvas.height * 0.62 - ry * f, d: depth };
  }

  function camera() {
    if (camMode === "fps") {
      return { x: player.x, y: player.y + 1.5, z: player.z, yaw: player.yaw, pitch: -0.08 };
    }
    if (camMode === "top") {
      return { x: player.x, y: player.y + 22, z: player.z - 0.01, yaw: 0, pitch: -1.2 };
    }
    return {
      x: player.x - Math.sin(player.yaw) * 8,
      y: player.y + 4.5,
      z: player.z - Math.cos(player.yaw) * 8,
      yaw: player.yaw,
      pitch: -0.28,
    };
  }

  function drawBox(ctx, cam, pos, size, color, alpha) {
    const hx = size[0] / 2, hy = size[1], hz = size[2] / 2;
    const corners = [
      [-hx, 0, -hz], [hx, 0, -hz], [hx, 0, hz], [-hx, 0, hz],
      [-hx, hy, -hz], [hx, hy, -hz], [hx, hy, hz], [-hx, hy, hz],
    ].map(([x, y, z]) => project(pos[0] + x, pos[1] + y, pos[2] + z, cam));
    if (corners.every((c) => c.d < 0.4)) return;
    ctx.globalAlpha = alpha == null ? 1 : alpha;
    ctx.fillStyle = color;
    ctx.beginPath();
    ctx.moveTo(corners[4].x, corners[4].y);
    ctx.lineTo(corners[5].x, corners[5].y);
    ctx.lineTo(corners[6].x, corners[6].y);
    ctx.lineTo(corners[7].x, corners[7].y);
    ctx.closePath();
    ctx.fill();
    ctx.strokeStyle = "rgba(0,0,0,0.35)";
    ctx.stroke();
    ctx.globalAlpha = 1;
  }

  function draw() {
    const ctx = canvas.getContext("2d");
    const w = canvas.width, h = canvas.height;
    ctx.fillStyle = "#0f1110";
    ctx.fillRect(0, 0, w, h);
    const cam = camera();
    const cs = terrain.cell_size || 2;
    for (let z = 0; z < (terrain.depth || 0); z += 2) {
      for (let x = 0; x < (terrain.width || 0); x += 2) {
        const i = z * terrain.width + x;
        const wx = (x - terrain.width / 2) * cs;
        const wz = (z - terrain.depth / 2) * cs;
        const wy = (terrain.heights && terrain.heights[i]) || 0;
        const surf = (terrain.surfaces && terrain.surfaces[i]) || "ground";
        const color = surf === "water" ? "#1b4b6b" : surf === "rock" ? "#5a5348" : surf === "obstacle" ? "#3f3a34" : "#2a3328";
        drawBox(ctx, cam, [wx, wy, wz], [cs * 1.9, 0.08, cs * 1.9], color, 0.95);
      }
    }
    const sprites = [];
    for (const el of elements) {
      if (el.gone) continue;
      sprites.push({ d: Math.hypot(el.position[0] - cam.x, el.position[2] - cam.z), el });
    }
    sprites.sort((a, b) => b.d - a.d);
    for (const s of sprites) {
      const el = s.el;
      const size = el.size || [1, 1, 1];
      drawBox(ctx, cam, el.position, size, el.color || "#ccc", 1);
    }
    drawBox(ctx, cam, [player.x, player.y, player.z], [0.7, player.h, 0.7], "#d7d2c4", 1);
    state.ready = true;
    state.loopTick += 1;
  }

  function frame(t) {
    const dt = Math.min(0.05, (t - last) / 1000);
    last = t;
    step(dt);
    draw();
    paintHud();
    requestAnimationFrame(frame);
  }

  function resize() {
    const r = canvas.getBoundingClientRect();
    canvas.width = Math.max(320, Math.floor(r.width * devicePixelRatio));
    canvas.height = Math.max(240, Math.floor(r.height * devicePixelRatio));
  }

  window.addEventListener("keydown", (e) => {
    keys[e.code] = true;
    state.inputSeen = true;
    if (camMode === "fps" && (e.code === "ArrowLeft")) player.yaw -= 0.08;
    if (camMode === "fps" && (e.code === "ArrowRight")) player.yaw += 0.08;
    if (e.code === "KeyR") reset();
  });
  window.addEventListener("keyup", (e) => { keys[e.code] = false; });
  canvas.addEventListener("click", () => {
    canvas.focus();
    state.inputSeen = true;
    if (camMode === "fps" && canvas.requestPointerLock) canvas.requestPointerLock();
  });
  window.addEventListener("mousemove", (e) => {
    if (document.pointerLockElement !== canvas) return;
    player.yaw += e.movementX * 0.0025;
    mx += e.movementX;
  });
  if (restartBtn) restartBtn.addEventListener("click", reset);
  window.addEventListener("resize", resize);
  resize();
  paintHud();
  requestAnimationFrame(frame);

  window.__yoyoGame = {
    get ready() { return state.ready; },
    get inputSeen() { return state.inputSeen; },
    get loopTick() { return state.loopTick; },
    get outcome() { return state.outcome; },
    restart: reset,
    state,
  };
})();
