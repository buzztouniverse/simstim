// The signal field: the landing page's three.js piece (decision 0021).
//
// A plane of braille cells seen in perspective. The three strands of the
// launcher's welcome mark (internal/tui/mark.go) run across it as raised,
// brighter bands that weave and pinch at both ends, in the brand gradient, and
// a pulse runs along them. Each point is drawn as a braille glyph whose dots
// fill up with the signal's strength. The pointer lifts the field under it,
// and a click sends a ripple. All the motion is in the vertex shader: each
// frame sets a few numbers and draws once.
//
// landing.js loads this module only when motion is on and WebGL works. It
// pauses when the field is off screen or the tab is hidden, and draws one
// still frame when motion turns off.

import {
  WebGLRenderer,
  Scene,
  PerspectiveCamera,
  BufferGeometry,
  BufferAttribute,
  Points,
  ShaderMaterial,
  CanvasTexture,
  Color,
  Vector2,
  Vector3,
  Raycaster,
  Plane,
  NormalBlending,
  AdditiveBlending,
  LinearFilter,
} from "../vendor/three/three.module.js";
import { everyFrame, motionOn } from "./lib/signal.js";

const VERTEX = /* glsl */ `
  uniform float uTime;
  uniform float uWidth;
  uniform float uDepth;
  uniform vec3 uPointer;   // x, z on the plane, and its strength
  uniform vec4 uRipple;    // x, z, start time, strength
  uniform float uScale;
  attribute vec2 aGrid;    // x and z on the plane
  attribute float aSeed;
  varying float vLevel;
  varying float vAlpha;
  varying float vU;

  // The mark's three strands: amplitude, waves across, speed, phase.
  const vec4 S0 = vec4(1.0, 1.0, 1.2, 0.0);
  const vec4 S1 = vec4(0.8, 1.6, -0.9, 2.2);
  const vec4 S2 = vec4(0.55, 2.4, 1.7, 4.4);

  float strand(vec4 s, float u, float z, float env) {
    float c = -uDepth * 0.42 + s.x * env * uDepth * 0.34 * sin(6.2831853 * u * s.y + s.z * uTime * 0.5 + s.w);
    float d = (z - c) / 0.34;
    return exp(-d * d);
  }

  void main() {
    float u = aGrid.x / uWidth + 0.5;
    float z = aGrid.y;
    float env = pow(sin(3.14159265 * clamp(u, 0.0, 1.0)), 1.2);
    float glow = max(max(strand(S0, u, z, env), strand(S1, u, z, env)), strand(S2, u, z, env)) * env;
    // A pulse runs along the strands, as on the welcome mark.
    float p = (u - fract(uTime / 5.2) * 1.3 + 0.15) / 0.07;
    glow *= 0.62 + 0.38 * exp(-p * p);

    float swell = 0.07 * sin(aGrid.x * 0.55 + uTime * 0.45) * cos(z * 0.7 - uTime * 0.3);
    vec2 dp = vec2(aGrid.x - uPointer.x, z - uPointer.y);
    float near = exp(-dot(dp, dp) / 1.6) * uPointer.z;
    float rd = length(vec2(aGrid.x - uRipple.x, z - uRipple.y));
    float rt = uTime - uRipple.z;
    float ripple = uRipple.w * sin(rd * 5.0 - rt * 7.0) * exp(-rt * 1.4) * exp(-rd * 0.35) * step(0.0, rt) * step(rd, rt * 1.6 + 0.5);

    float h = glow * 0.55 + swell + near * 0.5 + ripple * 0.22;
    vec4 mv = modelViewMatrix * vec4(aGrid.x, h, z, 1.0);
    gl_Position = projectionMatrix * mv;
    gl_PointSize = uScale / -mv.z;

    float level = clamp(glow * 1.05 + near * 0.55 + abs(ripple) * 0.35 + aSeed * 0.08, 0.0, 1.0);
    vLevel = level;
    vU = u;
    float fog = (1.0 - smoothstep(uDepth * 0.45, uDepth * 1.02, -z)) * smoothstep(0.0, 0.1, u) * (1.0 - smoothstep(0.9, 1.0, u));
    vAlpha = fog * (0.3 + 0.7 * level);
  }
`;

const FRAGMENT = /* glsl */ `
  uniform sampler2D uAtlas;
  uniform vec3 uA;
  uniform vec3 uB;
  uniform float uGlyphs;
  varying float vLevel;
  varying float vAlpha;
  varying float vU;
  void main() {
    float i = floor(vLevel * (uGlyphs - 0.001));
    vec2 uv = vec2((i + gl_PointCoord.x) / uGlyphs, 1.0 - gl_PointCoord.y);
    float a = texture2D(uAtlas, uv).a * vAlpha;
    if (a < 0.02) discard;
    vec3 c = mix(uA, uB, clamp(vU * 1.1 - 0.05, 0.0, 1.0));
    gl_FragColor = vec4(c * (0.7 + 0.9 * vLevel), a);
  }
`;

// The glyphs, from one dot to eight: a braille cell filling up.
const FILL = [0x02, 0x06, 0x16, 0x36, 0x76, 0xf6, 0xf7, 0xff];
const DOT = [[0, 0], [0, 1], [0, 2], [1, 0], [1, 1], [1, 2], [0, 3], [1, 3]];

function atlas() {
  const cell = 64;
  const c = document.createElement("canvas");
  c.width = cell * FILL.length;
  c.height = cell;
  const g = c.getContext("2d");
  g.fillStyle = "#fff";
  FILL.forEach((bits, i) => {
    for (let b = 0; b < 8; b++) {
      if (!(bits & (1 << b))) continue;
      const [col, row] = DOT[b];
      g.beginPath();
      g.arc(i * cell + 20 + col * 24, 8 + row * 16, 5.2, 0, Math.PI * 2);
      g.fill();
    }
  });
  const t = new CanvasTexture(c);
  t.minFilter = LinearFilter;
  t.magFilter = LinearFilter;
  t.generateMipmaps = false;
  return t;
}

function themeColours(dark) {
  return dark
    ? { a: new Color("#0e9c9a"), b: new Color("#00e5ff"), blend: AdditiveBlending }
    : { a: new Color("#045a5a"), b: new Color("#0088a6"), blend: NormalBlending };
}

function isDark() {
  return document.documentElement.classList.contains("dark");
}

export function start(host) {
  const canvas = document.createElement("canvas");
  canvas.setAttribute("aria-hidden", "true");
  let renderer;
  try {
    // No software GL: a slow machine keeps the still mark instead.
    renderer = new WebGLRenderer({ canvas, alpha: true, antialias: false, powerPreference: "low-power", failIfMajorPerformanceCaveat: true });
  } catch (e) {
    return null; // no WebGL: the still mark stays
  }
  host.prepend(canvas);
  const phone = window.matchMedia("(max-width: 719px)").matches;
  const nx = phone ? 84 : 150;
  const nz = phone ? 26 : 40;
  const width = phone ? 15 : 26;
  const depth = phone ? 8 : 11;

  const grid = new Float32Array(nx * nz * 2);
  const seed = new Float32Array(nx * nz);
  const pos = new Float32Array(nx * nz * 3);
  let k = 0;
  for (let j = 0; j < nz; j++) {
    for (let i = 0; i < nx; i++) {
      const x = (i / (nx - 1) - 0.5) * width;
      const z = -(j / (nz - 1)) * depth;
      grid[k * 2] = x;
      grid[k * 2 + 1] = z;
      pos[k * 3] = x;
      pos[k * 3 + 2] = z;
      seed[k] = ((Math.sin(i * 12.9898 + j * 78.233) * 43758.5453) % 1 + 1) % 1;
      k++;
    }
  }
  const geo = new BufferGeometry();
  geo.setAttribute("position", new BufferAttribute(pos, 3));
  geo.setAttribute("aGrid", new BufferAttribute(grid, 2));
  geo.setAttribute("aSeed", new BufferAttribute(seed, 1));

  let colours = themeColours(isDark());
  const uniforms = {
    uTime: { value: 0 },
    uWidth: { value: width },
    uDepth: { value: depth },
    uPointer: { value: new Vector3(0, -depth * 0.4, 0) },
    uRipple: { value: [0, 0, -99, 0] },
    uScale: { value: 1 },
    uAtlas: { value: atlas() },
    uA: { value: colours.a },
    uB: { value: colours.b },
    uGlyphs: { value: FILL.length },
  };
  const material = new ShaderMaterial({
    uniforms,
    vertexShader: VERTEX,
    fragmentShader: FRAGMENT,
    transparent: true,
    depthWrite: false,
    blending: colours.blend,
  });
  const points = new Points(geo, material);
  points.frustumCulled = false;
  const scene = new Scene();
  scene.add(points);
  const camera = new PerspectiveCamera(32, 2, 0.1, 60);
  const look = new Vector3(0, 0, -depth * 0.2);

  const size = () => {
    const w = host.clientWidth;
    const h = host.clientHeight;
    if (!w || !h) return;
    renderer.setPixelRatio(Math.min(window.devicePixelRatio || 1, 1.75));
    renderer.setSize(w, h, false);
    camera.aspect = w / h;
    camera.position.set(0, phone ? 2.4 : 2.2, phone ? 3.6 : 3.2);
    camera.lookAt(look);
    camera.updateProjectionMatrix();
    // Glyphs about 16 px tall at the front of the field, smaller with distance.
    uniforms.uScale.value = renderer.getPixelRatio() * (phone ? 80 : 112);
  };
  size();
  const ro = new ResizeObserver(size);
  ro.observe(host);

  // The pointer: where it meets the plane, eased in and out.
  const ray = new Raycaster();
  const plane = new Plane(new Vector3(0, 1, 0), 0);
  const hit = new Vector3();
  const target = new Vector3(0, -depth * 0.4, 0);
  const toPlane = (ev) => {
    const r = canvas.getBoundingClientRect();
    if (ev.clientY < r.top || ev.clientY > r.bottom) return null;
    const v = new Vector2(((ev.clientX - r.left) / r.width) * 2 - 1, -((ev.clientY - r.top) / r.height) * 2 + 1);
    ray.setFromCamera(v, camera);
    return ray.ray.intersectPlane(plane, hit) ? hit : null;
  };
  const move = (ev) => {
    const p = toPlane(ev);
    if (p) target.set(p.x, p.z, 1);
    else target.z = 0;
  };
  const leave = () => {
    target.z = 0;
  };
  const click = (ev) => {
    const p = toPlane(ev);
    if (p) uniforms.uRipple.value = [p.x, p.z, uniforms.uTime.value, 1];
  };
  // The field sits behind the page, so it listens on the page itself.
  const stage = host.parentElement;
  stage.addEventListener("pointermove", move, { passive: true });
  stage.addEventListener("pointerleave", leave, { passive: true });
  stage.addEventListener("pointerdown", click, { passive: true });

  const theme = new MutationObserver(() => {
    colours = themeColours(isDark());
    uniforms.uA.value = colours.a;
    uniforms.uB.value = colours.b;
    material.blending = colours.blend;
    material.needsUpdate = true;
    if (!motionOn()) draw(still);
  });
  theme.observe(document.documentElement, { attributes: true, attributeFilter: ["class"] });

  let last = 0;
  const still = 7.3; // a moment where the strands cross well
  const draw = (t) => {
    const dt = Math.min(t - last, 0.1);
    last = t;
    const p = uniforms.uPointer.value;
    const ease = 1 - Math.exp(-dt * 6);
    p.x += (target.x - p.x) * ease;
    p.y += (target.y - p.y) * ease;
    p.z += (target.z - p.z) * (1 - Math.exp(-dt * 3));
    uniforms.uTime.value = t;
    renderer.render(scene, camera);
  };
  const t0 = performance.now() / 1000 - still;
  everyFrame(30, (t) => draw(t - t0), () => motionOn(), () => {
    uniforms.uPointer.value.z = 0;
    draw(still);
  }, host);
  requestAnimationFrame(() => host.classList.add("live"));
  return {
    stop() {
      ro.disconnect();
      theme.disconnect();
      stage.removeEventListener("pointermove", move);
      stage.removeEventListener("pointerleave", leave);
      stage.removeEventListener("pointerdown", click);
      geo.dispose();
      material.dispose();
      uniforms.uAtlas.value.dispose();
      renderer.dispose();
      canvas.remove();
    },
  };
}
