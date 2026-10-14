/* ============================================================
 * emotions.js —— 数据层（纯数据，不含任何 DOM / 逻辑代码）
 *
 * ID 分段规则（十位 = 分组前缀，组间空号为扩展预留位，编号即对外契约不可重排）：
 *   00-09 生命周期 · 10-29 情绪反应 · 30-49 代理工作状态 · 50+ 自定义
 *
 * 50 套表情的完整定义，配置字段：
 *   pool      眼环索引池（25 组轮廓眼环，逐点弹簧插值切换）
 *   poolMs    池内轮换间隔 [min, max] ms
 *   poolSpeed 形变弹簧频率（检索 / 兴奋类 10，默认 6）
 *   blinkMs   眨眼间隔 [min, max] ms（null = 不眨眼）
 *   mouth     嘴形槽位（smile / grin / o / flat / frown / wavy / pout / open / dot）
 *   openness  常驻开合度（如睡眠 0.08、疲惫 0.55）
 *   antics    待机随机小动作（自旋 / 弹跳，9~18s 一次）
 *   ribbons / confetti  进入表情时的一次性事件：自旋甩彩带 / 撒花
 *   body.zzz   睡眠字母粒子（右上角循环漂浮）
 *   body.orbit 常驻水平环带（头顶彩带持续环绕）
 *   sequence  关键帧序列，settle: 'base' 回落基础姿态 / 'hold' 定格末帧
 *             / { next } 播完切换到下一个表情
 *   anims     sine / pulse / jitter / scan / glance / blink / ease / nod / sway / hold
 *   en        英文文案 { name, desc }（界面双语用，引擎不消费）
 *
 * 眼环池速查：
 *   0/8 平静 · 2/11/17/19 笑眼 · 3/21 圆睁 · 13/22/4 闭合困倦
 *   14/5/23 斜眼 · 7/16 怒目 · 15/9/20/12/18 扫读 · 10/1 聆听 · 24 羞怯
 * ============================================================ */

window.EMOTION_GROUPS = [
  { key: 'life',    name: '生命周期',     en: 'Lifecycle' },
  { key: 'emotion', name: '情绪反应',     en: 'Emotions' },
  { key: 'agent',   name: '代理工作状态', en: 'Agent States' },
  { key: 'custom',  name: '自定义',       en: 'Custom' }
];

window.EMOTION_SEED = [

  /* ==================== 1）生命周期（8 个） ==================== */

  {
    id: '00', name: '睡眠', group: 'life',
    desc: '闭眼成细线，右上角 zzz 缓缓飘起，头微垂，只剩缓慢呼吸',
    en: { name: 'Sleeping', desc: 'Eyes closed to thin lines, zzz drifting up at the top right, only a slow breath remains' },
    transition: 900,
    mouth: 'flat',
    gaze: false,
    pool: [13, 22, 4], poolMs: [6000, 10000], blinkMs: null, openness: 0.08,
    body: { y: 4, rotate: -2, breathe: 0.018, color: '#EEEBE4', zzz: 1 },
    eyes: { both: { y: 4, lookY: 2 } },
    anims: [
      { target: 'eyes', prop: 'y', type: 'sine', amp: 1.2, period: 3600 }
    ]
  },
  {
    id: '01', name: '唤醒', group: 'life',
    desc: '从闭合眼环缓缓睁开，先揉眼似的眨两下，随后进入待机',
    en: { name: 'Waking', desc: 'Eyes slowly crack open with a couple of groggy blinks, then settles into idle' },
    transition: 320,
    mouth: 'o',
    pool: [13], poolMs: [800, 800], blinkMs: null,
    sequence: {
      settle: { next: '02' },
      frames: [
        { at: 0,    eyes: { both: { open: 0.1, y: 4 } } },
        { at: 420,  eyes: { left: { open: 0.55, y: 2 }, right: { open: 0.12, y: 4 } } },
        { at: 820,  eyes: { both: { open: 0.3, y: 3 } } },
        { at: 1400, eyes: { both: { open: 1, scaleX: 1.12, scaleY: 1.12, y: -2 } } },
        { at: 2100, eyes: { both: { open: 1, y: 0 } } }
      ]
    }
  },
  {
    id: '02', name: '待机放空', group: 'life',
    desc: '左看看、右看看，目光在两侧各停留片刻，偶尔自旋甩彩带 / 弹跳',
    en: { name: 'Idle', desc: 'Glances left, glances right, lingering on each side; an occasional ribbon spin or bounce' },
    transition: 700,
    mouth: 'smile',
    pool: [0, 8], poolMs: [9000, 16000], blinkMs: [6000, 14000], antics: true,
    body: { breathe: 0.012 },
    anims: [
      { target: 'eyes', prop: 'lookX', type: 'glance', amp: 10, period: 4800 },
      { target: 'eyes', prop: 'lookY', type: 'sine', amp: 2, period: 4100, phase: 1.1 },
      { target: 'body', prop: 'x', type: 'sway', amp: 1.1, period: 5400 }
    ]
  },
  {
    id: '03', name: '好奇', group: 'life',
    desc: '圆睁 / 平静 / 扫视眼环快速轮换，头微倾，目光轻轻打量',
    en: { name: 'Curious', desc: 'Wide, calm and scanning eye rings rotate quickly, head tilted, sizing things up' },
    transition: 420,
    mouth: 'o',
    pool: [3, 21, 0, 15], poolMs: [1800, 3200], blinkMs: [2500, 5500],
    body: { rotate: 4, breathe: 0.01 },
    eyes: { both: { lookY: -1 } },
    anims: [
      { target: 'eyes', prop: 'lookX', type: 'sine', amp: 2.4, period: 2800 }
    ]
  },
  {
    id: '04', name: '发呆', group: 'life',
    desc: '半闭眼环慢轮换，双眼各望各的，偶尔弹跳提神',
    en: { name: 'Spacing Out', desc: 'Half-closed rings rotate slowly, each eye wandering its own way; a bounce now and then' },
    transition: 800,
    mouth: 'dot',
    pool: [4, 22, 0], poolMs: [3500, 6000], blinkMs: [4000, 8000], antics: true,
    body: { rotate: -3, breathe: 0.008 },
    eyes: {
      left:  { lookX: -4, lookY: 2 },
      right: { lookX: 5,  lookY: -1 }
    },
    anims: [
      { target: 'eyes', prop: 'lookX', type: 'sine', amp: 3, period: 6800 }
    ]
  },
  {
    id: '05', name: '加载苏醒', group: 'life',
    desc: '双眼缓慢交替亮起，像系统正在逐项初始化',
    en: { name: 'Booting', desc: 'Eyes light up in slow alternation, like a system initializing step by step' },
    transition: 480,
    mouth: 'flat',
    pool: [0, 8], poolMs: [6000, 10000], blinkMs: null,
    anims: [
      { target: 'left',  prop: 'open', type: 'blink', interval: 1600, dur: 700 },
      { target: 'right', prop: 'open', type: 'blink', interval: 1600, dur: 700, phaseMs: 800 },
      { target: 'eyes',  prop: 'scale', type: 'pulse', amp: 0.04, period: 1600 },
      { target: 'body', prop: 'scale', type: 'hold', amp: 0.018, period: 2400 }
    ]
  },
  {
    id: '06', name: '休眠', group: 'life',
    desc: '困倦眼环 + 半开合，几乎静止，只剩极弱的呼吸起伏',
    en: { name: 'Dormant', desc: 'Drowsy rings at half openness, nearly still, only the faintest breathing' },
    transition: 1200,
    mouth: 'flat',
    gaze: false,
    pool: [4, 22, 13], poolMs: [4000, 8000], blinkMs: null, openness: 0.4,
    body: { y: 6, scale: 0.98, rotate: -1, breathe: 0.005, color: '#EBE8E1' },
    eyes: { both: { y: 5 } }
  },
  {
    id: '07', name: '抖动唤醒', group: 'life',
    desc: '整球轻颤，闭合眼环交错睁开，随后进入待机',
    en: { name: 'Shake Awake', desc: 'The whole body trembles as the eyes stagger open, then settles into idle' },
    transition: 220,
    mouth: 'o',
    pool: [13], poolMs: [800, 800], blinkMs: null,
    body: { breathe: 0.004 },
    anims: [
      { target: 'body', prop: 'x', type: 'jitter', amp: 4.5, speed: 10, decay: 1600 },
      { target: 'body', prop: 'rotate', type: 'jitter', amp: 3, speed: 8, decay: 1600 }
    ],
    sequence: {
      settle: { next: '02' },
      frames: [
        { at: 0,    eyes: { both: { open: 0.1 } } },
        { at: 380,  eyes: { left: { open: 0.4 }, right: { open: 0.12 } } },
        { at: 900,  eyes: { both: { open: 0.7, y: 1 } } },
        { at: 1600, eyes: { both: { open: 1, scaleX: 1.08, scaleY: 1.08, y: -1 } } }
      ]
    }
  },

  /* ==================== 2）情绪反应（12 个） ==================== */

  {
    id: '10', name: '开心', group: 'emotion',
    desc: '笑眼轮换，目光下看看、上看看，身体轻快起伏，偶尔自旋甩彩带',
    en: { name: 'Happy', desc: 'Smiling rings rotate, gaze bobs down and up, body bounces lightly; occasional ribbon spin' },
    transition: 380,
    mouth: 'grin',
    pool: [2, 11, 17, 19], poolMs: [2500, 4500], blinkMs: [2500, 5000], antics: true,
    body: { y: -3, breathe: 0.014, color: '#F6EFE4' },
    eyes: { both: { y: -3 } },
    anims: [
      { target: 'eyes', prop: 'lookY', type: 'glance', amp: 6, period: 3000 },
      { target: 'body', prop: 'y', type: 'sine', amp: 2.2, period: 1400 }
    ]
  },
  {
    id: '11', name: '疑惑', group: 'emotion',
    desc: '斜眼环轮换，头微倾，一眼放大一眼收小',
    en: { name: 'Puzzled', desc: 'Skeptical rings, head atilt, one eye enlarged and the other shrunk' },
    transition: 420,
    mouth: 'wavy',
    pool: [14, 5, 8], poolMs: [2200, 3800], blinkMs: [2800, 5500],
    body: { rotate: -8, breathe: 0.008 },
    eyes: {
      left:  { y: -4, scaleX: 1.1, scaleY: 1.1 },
      right: { y: 3,  scaleX: 0.9, scaleY: 0.9, lookX: 3 }
    },
    anims: [
      { target: 'body', prop: 'rotate', type: 'sine', amp: 1.4, period: 3200 }
    ]
  },
  {
    id: '12', name: '失落', group: 'emotion',
    desc: '困倦 / 闭合眼环慢轮换，眼睛下沉，目光低垂',
    en: { name: 'Down', desc: 'Drowsy rings rotate slowly, eyes sinking with a downcast gaze' },
    transition: 820,
    mouth: 'frown',
    pool: [4, 13, 22], poolMs: [4000, 7000], blinkMs: [4000, 8000],
    body: { y: 5, rotate: -4, breathe: 0.007, color: '#EDEAE3' },
    eyes: { both: { y: 8, scaleX: 0.88, scaleY: 0.88, lookY: 4 } },
    anims: [
      { target: 'eyes', prop: 'y', type: 'sine', amp: 1.6, period: 3600 }
    ]
  },
  {
    id: '13', name: '惊讶', group: 'emotion',
    desc: '双眼瞬间放大到 1.45 倍再回稳，身体轻抬定住',
    en: { name: 'Surprised', desc: 'Eyes pop to 1.45x in an instant then steady, the body lifting and freezing' },
    transition: 180,
    mouth: 'open',
    pool: [3, 21], poolMs: [2500, 4000], blinkMs: [1800, 3500],
    body: { y: -4, scale: 1.03, breathe: 0.006 },
    eyes: { both: { scaleX: 1.14, scaleY: 1.14, y: -2, lookY: -2 } },
    sequence: {
      settle: 'base',
      frames: [
        { at: 0,   eyes: { both: { scaleX: 0.92, scaleY: 0.92 } } },
        { at: 150, eyes: { both: { scaleX: 1.45, scaleY: 1.45, y: -4 } }, body: { y: -7, scale: 1.05 } },
        { at: 420, eyes: { both: { scaleX: 1.22, scaleY: 1.22, y: -3 } } }
      ]
    }
  },
  {
    id: '14', name: '害羞', group: 'emotion',
    desc: '目光躲向一侧，脸颊在一秒多里渐渐泛起粉色',
    en: { name: 'Shy', desc: 'Gaze slips off to one side as the face slowly blushes pink over a second' },
    transition: 560,
    mouth: 'pout',
    pool: [0, 24, 13], poolMs: [3000, 5500], blinkMs: [3000, 6000],
    /* base 色 = 序列终态粉，保证静态缩略图与最终观感一致 */
    body: { rotate: 6, breathe: 0.012, color: '#F4D3D0' },
    eyes: { both: { y: 4, lookX: 8, lookY: 3 } },
    anims: [
      { target: 'eyes', prop: 'lookX', type: 'sine', amp: 2, period: 2600 }
    ],
    sequence: {
      settle: 'hold',
      frames: [
        { at: 0,    body: { color: '#F3F0EA' } },
        { at: 1500, body: { color: '#F4D3D0' } }
      ]
    }
  },
  {
    id: '15', name: '疲惫', group: 'emotion',
    desc: '眼皮沉重半睁（低开合度 + 困倦眼环），目光下沉',
    en: { name: 'Tired', desc: 'Heavy eyelids at half openness with drowsy rings, gaze sinking low' },
    transition: 900,
    mouth: 'flat',
    pool: [4, 22, 13], poolMs: [4000, 8000], blinkMs: null, openness: 0.55,
    body: { y: 4, rotate: -3, breathe: 0.016, color: '#EFECE5' },
    eyes: { both: { y: 5, lookY: 3 } },
    anims: [
      { target: 'eyes', prop: 'open', type: 'sine', amp: 0.06, period: 3400 }
    ]
  },
  {
    id: '16', name: '专注', group: 'emotion',
    desc: '专注眼环族轮换，双眼微微内聚，身体几乎不动',
    en: { name: 'Focused', desc: 'Focused ring family rotates, eyes converge slightly, body almost motionless' },
    transition: 320,
    mouth: 'flat',
    pool: [7, 16, 11, 10], poolMs: [1800, 3200], blinkMs: [2800, 5500],
    body: { breathe: 0.004 },
    eyes: {
      left:  { x: 4 },
      right: { x: -4 }
    }
  },
  {
    id: '17', name: '慌张', group: 'emotion',
    desc: '圆睁眼环高频轮换，目光乱晃，整球细颤',
    en: { name: 'Panicked', desc: 'Wide rings rotate at high frequency, gaze darting about, the whole body quivering' },
    transition: 200,
    mouth: 'wavy',
    pool: [3, 21], poolMs: [900, 1800], blinkMs: [1200, 3000],
    body: { breathe: 0.006 },
    anims: [
      { target: 'eyes', prop: 'lookX', type: 'jitter', amp: 6, speed: 11 },
      { target: 'eyes', prop: 'lookY', type: 'jitter', amp: 4, speed: 9 },
      { target: 'body', prop: 'x', type: 'jitter', amp: 1.6, speed: 8 }
    ]
  },
  {
    id: '18', name: '无奈', group: 'emotion',
    desc: '斜眼环轮换，头一歪，双眼翻向斜上方',
    en: { name: 'Resigned', desc: 'Sidelong rings, head cocked, both eyes rolling up and away' },
    transition: 560,
    mouth: 'flat',
    pool: [14, 5, 23], poolMs: [2600, 4500], blinkMs: [4500, 8000],
    body: { rotate: 10, y: 2, breathe: 0.01 },
    eyes: { both: { lookX: 7, lookY: -8 } },
    anims: [
      { target: 'eyes', prop: 'lookY', type: 'sine', amp: 1.6, period: 3000 },
      { target: 'body', prop: 'y', type: 'sine', amp: 1.2, period: 3000 }
    ]
  },
  {
    id: '19', name: '满意', group: 'emotion',
    desc: '目光正视前方，双眼有节奏地上下点动，如同点头认可',
    en: { name: 'Satisfied', desc: 'Gazing straight ahead while the eyes nod up and down in steady approval' },
    transition: 580,
    mouth: 'smile',
    pool: [15, 8, 2], poolMs: [3500, 6000], blinkMs: [3500, 7000], antics: true,
    body: { breathe: 0.012, color: '#F5EFE6' },
    anims: [
      { target: 'eyes', prop: 'y', type: 'sine', amp: 5, period: 1050 },
      { target: 'body', prop: 'y', type: 'sine', amp: 1.6, period: 1050, phase: 0.6 }
    ]
  },
  {
    id: '20', name: '困惑', group: 'emotion',
    desc: '两眼大小不一，斜眼环轮换，注视方向对不齐',
    en: { name: 'Confused', desc: 'Mismatched eye sizes and sidelong rings, the two gazes never quite aligning' },
    transition: 480,
    mouth: 'wavy',
    pool: [14, 5, 8], poolMs: [2200, 3800], blinkMs: [2800, 5500],
    body: { rotate: -5, breathe: 0.008 },
    eyes: {
      left:  { scaleX: 1.16, scaleY: 1.16, y: -3, lookX: -3 },
      right: { scaleX: 0.8,  scaleY: 0.8,  y: 4,  lookX: 5 }
    },
    anims: [
      { target: 'left',  prop: 'lookX', type: 'sine', amp: 2.5, period: 3200 },
      { target: 'right', prop: 'lookX', type: 'sine', amp: 2.5, period: 3200, phase: 1.6 }
    ]
  },
  {
    id: '21', name: '生气', group: 'emotion',
    desc: '怒目圆睁，脸色在 0.25s 内迅速涨红并保持，身体细微发抖',
    en: { name: 'Angry', desc: 'Glaring eyes; the face flushes red within a quarter second and stays, body trembling' },
    transition: 260,
    mouth: 'frown',
    pool: [7, 16], poolMs: [2200, 3800], blinkMs: [3500, 7000],
    /* base 色 = 序列终态红，保证静态缩略图与最终观感一致 */
    body: { y: 1, breathe: 0.004, color: '#E4574A' },
    anims: [
      { target: 'body', prop: 'x', type: 'jitter', amp: 1.1, speed: 7 }
    ],
    sequence: {
      settle: 'hold',
      frames: [
        { at: 0,   body: { color: '#F3F0EA' } },
        { at: 250, body: { color: '#E4574A' } }
      ]
    }
  },

  /* ==================== 3）代理工作状态（12 个） ==================== */

  {
    id: '30', name: '思考中', group: 'agent',
    desc: '思考眼环轮换，目光在上方巡回，一条彩带在头顶水平环绕',
    en: { name: 'Thinking', desc: 'Thinking rings rotate, gaze patrolling upward, a ribbon orbiting the head' },
    transition: 480,
    mouth: 'dot',
    pool: [8, 16, 14, 17, 5], poolMs: [2000, 3600], blinkMs: [3500, 7000],
    body: { rotate: -3, breathe: 0.01, orbit: 1 },
    eyes: { both: { lookY: -6, y: -2 } },
    anims: [
      { target: 'eyes', prop: 'lookX', type: 'ease', amp: 9, period: 2800 },
      { target: 'body', prop: 'rotate', type: 'sway', amp: 1.2, period: 3600 }
    ]
  },
  {
    id: '31', name: '接收任务', group: 'agent',
    desc: '轻轻眨一下并放大，像点头确认收到',
    en: { name: 'Receiving', desc: 'A quick blink and slight enlargement, like a nod of acknowledgement' },
    transition: 220,
    mouth: 'smile',
    pool: [19, 0, 8], poolMs: [4000, 8000], blinkMs: null,
    body: { breathe: 0.008 },
    sequence: {
      settle: 'base',
      frames: [
        { at: 0 },
        { at: 100, eyes: { both: { open: 0.1 } } },
        { at: 280, eyes: { both: { open: 1, scaleX: 1.12, scaleY: 1.12, y: -2 } }, body: { y: -3 } },
        { at: 700, eyes: { both: { open: 1 } } }
      ]
    }
  },
  {
    id: '32', name: '处理中忙碌', group: 'agent',
    desc: '专注眼环轮换，目光小幅循环往复',
    en: { name: 'Busy', desc: 'Focused rings rotate while the gaze loops in tight little circuits' },
    transition: 360,
    mouth: 'flat',
    pool: [7, 16, 11, 10], poolMs: [1800, 3200], blinkMs: [2800, 5500],
    body: { breathe: 0.008 },
    anims: [
      { target: 'eyes', prop: 'lookX', type: 'sine', amp: 6, period: 1200 },
      { target: 'eyes', prop: 'lookY', type: 'sine', amp: 4, period: 900, phase: 0.8 }
    ]
  },
  {
    id: '33', name: '任务完成', group: 'agent',
    desc: '笑眼轮换 + 自旋甩彩带 + 撒花庆祝',
    en: { name: 'Done', desc: 'Smiling rings, a celebratory ribbon spin and a burst of confetti' },
    transition: 240,
    mouth: 'grin',
    pool: [2, 8, 17], poolMs: [1400, 2600], blinkMs: [2200, 4500],
    body: { ribbons: 1, confetti: 0.95 },
    eyes: { both: { y: -3 } },
    sequence: {
      settle: 'base',
      frames: [
        { at: 0,   body: { y: 0 } },
        { at: 300, eyes: { both: { scaleX: 1.1, scaleY: 1.1, y: -5 } }, body: { y: -6 } },
        { at: 700, eyes: { both: { scaleX: 1.05, scaleY: 1.05, y: -4 } }, body: { y: -2 } },
        { at: 1100, body: { y: 0 } }
      ]
    }
  },
  {
    id: '34', name: '出错', group: 'agent',
    desc: '圆睁眼环，脸色红白两色急促闪动，最后定格在警示红',
    en: { name: 'Error', desc: 'Wide rings as the face flashes red and white, settling on alarm red' },
    transition: 220,
    mouth: 'wavy',
    pool: [3, 21], poolMs: [2000, 3600], blinkMs: null,
    body: { rotate: -6, color: '#E25B5B' },
    eyes: {
      left:  { y: -3, rotate: -6 },
      right: { y: 4,  rotate: 8 }
    },
    anims: [
      { target: 'eyes', prop: 'lookX', type: 'jitter', amp: 1.4, speed: 8, decay: 800 }
    ],
    sequence: {
      settle: 'hold',
      frames: [
        { at: 0,   body: { color: '#E25B5B', rotate: -6 } },
        { at: 170, body: { color: '#F6F3EC', rotate: -4 } },
        { at: 340, body: { color: '#E25B5B', rotate: -7 } },
        { at: 510, body: { color: '#F6F3EC', rotate: -5 } },
        { at: 700, body: { color: '#DE5555', rotate: -6 } }
      ]
    }
  },
  {
    id: '35', name: '等待输入', group: 'agent',
    desc: '聆听眼环轮换，目光轻轻上下扫读',
    en: { name: 'Listening', desc: 'Listening rings rotate while the gaze sweeps gently up and down' },
    transition: 480,
    mouth: 'smile',
    pool: [10, 1, 19], poolMs: [2800, 5000], blinkMs: [3000, 7000],
    body: { breathe: 0.01 },
    anims: [
      { target: 'eyes', prop: 'lookY', type: 'sine', amp: 6, period: 2200 }
    ]
  },
  {
    id: '36', name: '联网加载', group: 'agent',
    desc: '左右眼轮流眨，像信号在两端来回跳',
    en: { name: 'Loading', desc: 'Eyes blink in alternation, like a signal hopping between two endpoints' },
    transition: 380,
    mouth: 'dot',
    pool: [0, 8], poolMs: [6000, 10000], blinkMs: null,
    anims: [
      { target: 'left',  prop: 'open', type: 'blink', interval: 1200, dur: 380 },
      { target: 'right', prop: 'open', type: 'blink', interval: 1200, dur: 380, phaseMs: 600 },
      { target: 'eyes',  prop: 'lookX', type: 'sine', amp: 2, period: 2400 },
      { target: 'body', prop: 'x', type: 'sway', amp: 1.4, period: 2600 }
    ]
  },
  {
    id: '37', name: '复述回忆', group: 'agent',
    desc: '聆听眼环慢轮换，目光飘向上方翻检记忆',
    en: { name: 'Recalling', desc: 'Listening rings rotate slowly as the gaze drifts upward, leafing through memory' },
    transition: 780,
    mouth: 'dot',
    pool: [10, 1, 19], poolMs: [4000, 8000], blinkMs: null,
    body: { rotate: -2, breathe: 0.009 },
    eyes: { both: { lookY: -9, lookX: 3, y: -3 } },
    anims: [
      { target: 'eyes', prop: 'lookY', type: 'sine', amp: 2.4, period: 4000 }
    ]
  },
  {
    id: '38', name: '拒绝/受限', group: 'agent',
    desc: '斜眼下压，进入时连续摇头，明确表示不行',
    en: { name: 'Refusing', desc: 'A lowered sidelong gaze with a firm head-shake on entry: the answer is no' },
    transition: 380,
    mouth: 'frown',
    pool: [14, 5, 23], poolMs: [2600, 4500], blinkMs: [4500, 8000], openness: 0.6,
    body: { y: 2, rotate: -2, color: '#EFE8E4' },
    eyes: { both: { lookY: 3, y: 2 } },
    sequence: {
      settle: 'base',
      frames: [
        { at: 0,   body: { x: 0 } },
        { at: 130, body: { x: -9, rotate: -6 } },
        { at: 300, body: { x: 8,  rotate: 2 } },
        { at: 470, body: { x: -6, rotate: -5 } },
        { at: 630, body: { x: 4,  rotate: 0 } },
        { at: 800, body: { x: 0,  rotate: -2 } }
      ]
    }
  },
  {
    id: '39', name: '输出回复', group: 'agent',
    desc: '扫读眼环轮换，随输出节奏轻微缩放',
    en: { name: 'Replying', desc: 'Reading rings rotate with a gentle pulse that keeps time with the output' },
    transition: 360,
    mouth: 'open',
    pool: [15, 9], poolMs: [4000, 8000], blinkMs: null,
    body: { breathe: 0.008 },
    eyes: { both: { y: -2 } },
    anims: [
      { target: 'eyes', prop: 'scale', type: 'pulse', amp: 0.1, period: 680 },
      { target: 'body', prop: 'y', type: 'sine', amp: 1.2, period: 680 }
    ]
  },
  {
    id: '40', name: '检索资料', group: 'agent',
    desc: '6 组扫读眼环高速轮换（弹簧加速），目光左右快扫',
    en: { name: 'Searching', desc: 'Six reading rings rotate at high speed while the gaze sweeps rapidly side to side' },
    transition: 320,
    mouth: 'flat',
    pool: [15, 9, 3, 20, 12, 18], poolMs: [1000, 1800], poolSpeed: 10, blinkMs: [1600, 4000],
    body: { breathe: 0.006 },
    anims: [
      { target: 'eyes', prop: 'lookX', type: 'scan', amp: 11, period: 700 }
    ]
  },
  {
    id: '41', name: '停止终止', group: 'agent',
    desc: '闭合眼环，慢慢收小半闭后定格',
    en: { name: 'Powering Off', desc: 'Closing rings shrink to half-closed and quietly freeze' },
    transition: 280,
    mouth: 'flat',
    gaze: false,
    pool: [13, 22], poolMs: [6000, 9000], blinkMs: null,
    body: { y: 3, breathe: 0.004, color: '#EBE8E2' },
    sequence: {
      settle: 'hold',
      frames: [
        { at: 0 },
        { at: 1500, eyes: { both: { scaleX: 0.6, scaleY: 0.6, open: 0.35, y: 3 } }, body: { y: 4, scale: 0.97 } }
      ]
    }
  },

  /* ==================== 4）生命周期补完（08-09） ==================== */

  {
    id: '08', name: '伸懒腰', group: 'life',
    desc: '闭眼向上舒展，再轻轻落下，像刚醒过来活动一下',
    en: { name: 'Stretching', desc: 'Eyes close as the body lifts and lengthens, then settles back down' },
    transition: 520,
    mouth: 'o',
    pool: [13, 22, 2], poolMs: [4000, 7000], blinkMs: null,
    body: { breathe: 0.008 },
    sequence: {
      settle: 'base',
      frames: [
        { at: 0,    eyes: { both: { open: 0.35, y: 2 } }, body: { y: 2, scale: 0.98 } },
        { at: 420,  eyes: { both: { open: 0.12, y: -2 } }, body: { y: -8, scale: 1.07, rotate: -3 } },
        { at: 980,  eyes: { both: { open: 0.2, y: -1 } }, body: { y: -5, scale: 1.04, rotate: 2 } },
        { at: 1500, eyes: { both: { open: 1, y: 0 } }, body: { y: 0, scale: 1, rotate: 0 } }
      ]
    },
    anims: [
      { target: 'body', prop: 'y', type: 'sine', amp: 1.1, period: 2400 }
    ]
  },
  {
    id: '09', name: '偷瞄', group: 'life',
    desc: '身体微侧，一眼睁大一眼收细，目光悄悄瞟过去',
    en: { name: 'Peeking', desc: 'A slight lean, one eye wide and one narrowed, glancing aside' },
    transition: 380,
    mouth: 'dot',
    pool: [14, 5, 3], poolMs: [2200, 4000], blinkMs: [2800, 5600],
    body: { rotate: 9, breathe: 0.01 },
    eyes: {
      left:  { scaleX: 1.12, scaleY: 1.12, lookX: 8, lookY: -1 },
      right: { scaleX: 0.82, scaleY: 0.82, open: 0.72, lookX: 7, y: 2 }
    },
    anims: [
      { target: 'eyes', prop: 'lookX', type: 'ease', amp: 1.8, period: 2600 },
      { target: 'body', prop: 'rotate', type: 'sine', amp: 1.2, period: 3000 }
    ]
  },

  /* ==================== 5）情绪反应补完（22-29） ==================== */

  {
    id: '22', name: '眨眼', group: 'emotion',
    desc: '右眼定时轻轻一眨，带一点笑意，像偷偷打招呼',
    en: { name: 'Winking', desc: 'The right eye winks on a slow beat, with a small smile' },
    transition: 320,
    mouth: 'smile',
    pool: [2, 11, 19], poolMs: [3200, 5600], blinkMs: null,
    body: { rotate: 4, breathe: 0.012, color: '#F5EFE6' },
    eyes: { both: { y: -1 } },
    anims: [
      { target: 'right', prop: 'open', type: 'blink', interval: 2600, dur: 220 },
      { target: 'body', prop: 'rotate', type: 'sine', amp: 1.1, period: 2800 }
    ]
  },
  {
    id: '23', name: '心动', group: 'emotion',
    desc: '笑眼轻轻放大，面颊慢慢染上一层暖粉',
    en: { name: 'Fond', desc: 'Smiling eyes soften as a warm blush settles on the face' },
    transition: 640,
    mouth: 'pout',
    pool: [2, 11, 17], poolMs: [2800, 4800], blinkMs: [3200, 6400],
    body: { y: -1, breathe: 0.014, color: '#F4D8D4' },
    eyes: { both: { y: -2, scaleX: 1.06, scaleY: 1.06 } },
    anims: [
      { target: 'body', prop: 'scale', type: 'pulse', amp: 0.016, period: 1800 },
      { target: 'eyes', prop: 'scale', type: 'pulse', amp: 0.03, period: 1800, phase: 0.4 }
    ],
    sequence: {
      settle: 'hold',
      frames: [
        { at: 0,    body: { color: '#F3F0EA' } },
        { at: 900,  body: { color: '#F4D8D4' } }
      ]
    }
  },
  {
    id: '24', name: '得意', group: 'emotion',
    desc: '头微扬，一眼略眯，像把事情做成了还不愿明说',
    en: { name: 'Smug', desc: 'Chin up, one eye slightly narrowed, pleased but understated' },
    transition: 440,
    mouth: 'smile',
    pool: [11, 2, 14], poolMs: [2800, 5000], blinkMs: [3600, 7000], antics: true,
    body: { rotate: -6, y: -2, breathe: 0.01, color: '#F6F1E8' },
    eyes: {
      left:  { y: -3, lookY: -2 },
      right: { y: -1, scaleY: 0.88, lookY: -1 }
    },
    anims: [
      { target: 'body', prop: 'y', type: 'sine', amp: 1.2, period: 2200 }
    ]
  },
  {
    id: '25', name: '轻笑', group: 'emotion',
    desc: '笑眼轮换，身体用很轻的节奏上下颤一下',
    en: { name: 'Giggling', desc: 'Smiling rings with a tiny rhythmic bob, like a quiet laugh' },
    transition: 300,
    mouth: 'grin',
    pool: [2, 17, 19, 11], poolMs: [1600, 2800], blinkMs: [2200, 4200], antics: true,
    body: { breathe: 0.008, color: '#F6EFE4' },
    eyes: { both: { y: -2 } },
    anims: [
      { target: 'body', prop: 'y', type: 'sine', amp: 1.6, period: 640 },
      { target: 'eyes', prop: 'y', type: 'sine', amp: 0.8, period: 640, phase: 0.5 }
    ]
  },
  {
    id: '26', name: '坚定', group: 'emotion',
    desc: '双眼微微内聚，身体略前倾，几乎不晃',
    en: { name: 'Determined', desc: 'Eyes converge and the body leans in, almost still' },
    transition: 280,
    mouth: 'flat',
    pool: [7, 16, 10], poolMs: [2000, 3600], blinkMs: [3000, 5800],
    body: { y: -2, scale: 1.02, breathe: 0.005 },
    eyes: {
      left:  { x: 3, y: -1 },
      right: { x: -3, y: -1 }
    },
    anims: [
      { target: 'body', prop: 'y', type: 'hold', amp: -1.2, period: 2200, rise: 0.2, fall: 0.25 }
    ]
  },
  {
    id: '27', name: '惊叹', group: 'emotion',
    desc: '眼睛慢慢睁圆，比惊讶更慢、更轻，像看见了好东西',
    en: { name: 'Awe', desc: 'Eyes widen slowly, softer than surprise, as if something lovely appeared' },
    transition: 520,
    mouth: 'o',
    pool: [3, 21, 19], poolMs: [2400, 4200], blinkMs: [2800, 5200],
    body: { y: -2, scale: 1.02, breathe: 0.008, color: '#F7F3EB' },
    eyes: { both: { y: -2, lookY: -3 } },
    sequence: {
      settle: 'base',
      frames: [
        { at: 0,    eyes: { both: { scaleX: 0.96, scaleY: 0.96 } } },
        { at: 420,  eyes: { both: { scaleX: 1.22, scaleY: 1.22, y: -3 } }, body: { y: -4, scale: 1.03 } },
        { at: 900,  eyes: { both: { scaleX: 1.1, scaleY: 1.1, y: -2 } } }
      ]
    }
  },
  {
    id: '28', name: '喷嚏', group: 'emotion',
    desc: '先吸气闭眼，再轻轻一震，随后若无其事',
    en: { name: 'Sneeze', desc: 'A small inhale, a tiny shake, then composure returns' },
    transition: 220,
    mouth: 'o',
    pool: [13, 3], poolMs: [4000, 7000], blinkMs: null,
    body: { breathe: 0.004 },
    sequence: {
      settle: 'base',
      frames: [
        { at: 0,    eyes: { both: { open: 0.85 } }, body: { scale: 1 } },
        { at: 280,  eyes: { both: { open: 0.08, y: 2 } }, body: { scale: 1.06, y: -2 } },
        { at: 460,  eyes: { both: { open: 0.1 } }, body: { scale: 0.97, y: 3, x: 3 } },
        { at: 620,  eyes: { both: { open: 0.2 } }, body: { scale: 1.01, y: 0, x: -2 } },
        { at: 900,  eyes: { both: { open: 1, y: 0 } }, body: { scale: 1, x: 0 } }
      ]
    }
  },
  {
    id: '29', name: '俏皮', group: 'emotion',
    desc: '歪头、目光左右跳，偶尔自己弹一下',
    en: { name: 'Playful', desc: 'Head tilted, gaze hopping side to side, with an occasional bounce' },
    transition: 340,
    mouth: 'grin',
    pool: [3, 2, 19, 11], poolMs: [1800, 3200], blinkMs: [2200, 4800], antics: true,
    body: { rotate: 7, breathe: 0.012, color: '#F6F0E6' },
    eyes: { both: { y: -1 } },
    anims: [
      { target: 'eyes', prop: 'lookX', type: 'glance', amp: 8, period: 2200 },
      { target: 'body', prop: 'rotate', type: 'ease', amp: 2.2, period: 2600 }
    ]
  },

  /* ==================== 6）代理工作状态补完（42-49） ==================== */

  {
    id: '42', name: '耐心等待', group: 'agent',
    desc: '慢眨眼，嘴角微弯，像在安静地等结果',
    en: { name: 'Waiting', desc: 'Slow blinks and a small smile, patiently holding still' },
    transition: 560,
    mouth: 'smile',
    pool: [0, 8, 10], poolMs: [4000, 7200], blinkMs: [3800, 7200],
    body: { breathe: 0.014 },
    eyes: { both: { y: 1 } },
    anims: [
      { target: 'eyes', prop: 'lookY', type: 'ease', amp: 2.2, period: 3600 },
      { target: 'body', prop: 'scale', type: 'pulse', amp: 0.012, period: 2800 }
    ]
  },
  {
    id: '43', name: '构建中', group: 'agent',
    desc: '专注眼环配有节奏的轻轻起伏，像一层层叠上去',
    en: { name: 'Building', desc: 'Focused rings with a measured rise and fall, like work stacking up' },
    transition: 360,
    mouth: 'flat',
    pool: [7, 16, 11], poolMs: [1600, 2800], blinkMs: [2600, 5000],
    body: { breathe: 0.006 },
    eyes: { both: { y: -1 } },
    anims: [
      { target: 'body', prop: 'y', type: 'hold', amp: -2.2, period: 1600, rise: 0.22, fall: 0.2 },
      { target: 'eyes', prop: 'scale', type: 'pulse', amp: 0.04, period: 1600 }
    ]
  },
  {
    id: '44', name: '同步中', group: 'agent',
    desc: '目光平稳左右扫，身体跟着画一个很慢的圆',
    en: { name: 'Syncing', desc: 'Gaze sweeps evenly while the body traces a slow circle' },
    transition: 420,
    mouth: 'dot',
    pool: [15, 9, 12, 18], poolMs: [1800, 3000], blinkMs: [2800, 5600],
    body: { breathe: 0.008 },
    anims: [
      { target: 'eyes', prop: 'lookX', type: 'scan', amp: 7, period: 1400 },
      { target: 'body', prop: 'x', type: 'sine', amp: 1.6, period: 1800 },
      { target: 'body', prop: 'y', type: 'sine', amp: 1.1, period: 1800, phase: 1.57 }
    ]
  },
  {
    id: '45', name: '接通中', group: 'agent',
    desc: '头顶细环缓缓转，双眼交替亮起，比加载更轻',
    en: { name: 'Connecting', desc: 'A thin orbit and alternating eyes, lighter than the loading hop' },
    transition: 480,
    mouth: 'dot',
    pool: [0, 10, 8], poolMs: [5000, 8000], blinkMs: null,
    body: { breathe: 0.01, orbit: 1, color: '#F1EFE8' },
    anims: [
      { target: 'left',  prop: 'open', type: 'blink', interval: 1800, dur: 520 },
      { target: 'right', prop: 'open', type: 'blink', interval: 1800, dur: 520, phaseMs: 900 },
      { target: 'body', prop: 'rotate', type: 'sway', amp: 1.4, period: 3200 }
    ]
  },
  {
    id: '46', name: '低语', group: 'agent',
    desc: '身体略侧，嘴巴小小开合，像贴着耳朵说话',
    en: { name: 'Whispering', desc: 'Leaning in with a tiny mouth pulse, as if speaking close' },
    transition: 400,
    mouth: 'o',
    pool: [10, 1, 15], poolMs: [3000, 5400], blinkMs: [3200, 6400],
    body: { rotate: 8, scale: 0.98, breathe: 0.01 },
    eyes: { both: { lookX: 4, y: 1 } },
    face: { mouthSX: 0.78, mouthSY: 0.78 },
    anims: [
      { target: 'body', prop: 'y', type: 'sine', amp: 0.9, period: 720 },
      { target: 'eyes', prop: 'lookX', type: 'sine', amp: 1.4, period: 2400 }
    ]
  },
  {
    id: '47', name: '礼成', group: 'agent',
    desc: '轻轻一抬，甩一条彩带，不撒花，把完成收得很干净',
    en: { name: 'Flourish', desc: 'A small lift and a single ribbon, celebration without confetti' },
    transition: 280,
    mouth: 'smile',
    pool: [2, 17, 8], poolMs: [1800, 3200], blinkMs: [2400, 4800],
    body: { ribbons: 1, color: '#F6F0E6' },
    eyes: { both: { y: -2 } },
    sequence: {
      settle: 'base',
      frames: [
        { at: 0,    body: { y: 0, scale: 1 } },
        { at: 260,  eyes: { both: { scaleX: 1.08, scaleY: 1.08, y: -4 } }, body: { y: -5, scale: 1.03 } },
        { at: 640,  eyes: { both: { y: -2 } }, body: { y: -1, scale: 1 } },
        { at: 980,  body: { y: 0 } }
      ]
    }
  },
  {
    id: '48', name: '附和', group: 'agent',
    desc: '两拍点头，目光稳定，像在说“嗯，是的”',
    en: { name: 'Agreeing', desc: 'A two-beat nod with a steady gaze: a quiet yes' },
    transition: 360,
    mouth: 'smile',
    pool: [8, 2, 15], poolMs: [3200, 5600], blinkMs: [3400, 6800],
    body: { breathe: 0.01 },
    anims: [
      { target: 'body', prop: 'y', type: 'nod', amp: 5.5, period: 1680 },
      { target: 'eyes', prop: 'y', type: 'nod', amp: 3.2, period: 1680 }
    ]
  },
  {
    id: '49', name: '回神', group: 'agent',
    desc: '从半闭到睁清，肩膀一正，把走神收回来',
    en: { name: 'Snapping To', desc: 'Half-closed eyes clear and the posture squares, coming back' },
    transition: 300,
    mouth: 'o',
    pool: [4, 0, 8], poolMs: [4000, 7000], blinkMs: null,
    body: { breathe: 0.006 },
    sequence: {
      settle: 'base',
      frames: [
        { at: 0,    eyes: { both: { open: 0.4, y: 4 } }, body: { rotate: -5, y: 3 } },
        { at: 360,  eyes: { left: { open: 0.85, y: 1 }, right: { open: 0.35, y: 3 } } },
        { at: 720,  eyes: { both: { open: 1, scaleX: 1.1, scaleY: 1.1, y: -2 } }, body: { rotate: 0, y: -2 } },
        { at: 1100, eyes: { both: { open: 1, y: 0 } }, body: { y: 0 } }
      ]
    }
  }
];

export {};
