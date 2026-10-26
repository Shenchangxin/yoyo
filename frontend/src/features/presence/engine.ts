/** 拥有情绪球实例的创建，并补上庆祝、挥手和签名。不拥有导演规则，也不另开模型循环。不改服务端。 */
import "../../vendor/emotion-ball/rings.js";
import "../../vendor/emotion-ball/ball.js";
import "../../vendor/emotion-ball/emotions.js";
import "../../vendor/emotion-ball/engine.js";
import type { EmotionBallAPI, MoodMate, MoodMateCreateOpts } from "./types";

/** emotionBall 取出页面上的情绪球，没挂上就抛错。 */
export function emotionBall(): EmotionBallAPI {
  const api = window.EmotionBall;
  if (!api) throw new Error("Emotion Ball engine failed to load");
  return api;
}

/** createMate 在元素上建一只情绪球，并补上庆祝、挥手、签名和可卸载的监听。el 传给 create。opts 展开传递。 */
export function createMate(el: HTMLElement, opts: MoodMateCreateOpts = {}): MoodMate {
  const ball = emotionBall().create(el, {
    emotion: "02",
    fallbackId: "02",
    shape: "blob",
    ...opts,
  });
  const origOn = ball.on.bind(ball);
  const origOff = typeof ball.off === "function" ? ball.off.bind(ball) : undefined;
  /** 先跳一下，力度不是零时再转并散开。strength 与 0 比较是否不等。 */
  ball.celebrate = function (strength) {
    this.bounce();
    if (strength !== 0) {
      this.spin(1);
      this.burst(16);
    }
    return this;
  };
  /** 先轻推再转一下。 */
  ball.flourish = function () {
    this.nudge();
    this.spin(1);
    return this;
  };
  /** 转一下并交回成功。 */
  ball.signature = function () {
    this.spin(1);
    return true;
  };
  /** 接上原来的监听，并交回能卸掉的函数。ev 传给 origOn，传给 origOff。cb 传给 origOn，传给 origOff。 */
  ball.on = function (ev, cb) {
    origOn(ev, cb);
    return () => {
      origOff?.(ev, cb);
    };
  };
  return ball;
}
