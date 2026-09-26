import "../../vendor/emotion-ball/rings.js";
import "../../vendor/emotion-ball/ball.js";
import "../../vendor/emotion-ball/emotions.js";
import "../../vendor/emotion-ball/engine.js";
import type { EmotionBallAPI, MoodMate, MoodMateCreateOpts } from "./types";

export function emotionBall(): EmotionBallAPI {
  const api = window.EmotionBall;
  if (!api) throw new Error("Emotion Ball engine failed to load");
  return api;
}

export function createMate(el: HTMLElement, opts: MoodMateCreateOpts = {}): MoodMate {
  const ball = emotionBall().create(el, {
    emotion: "02",
    fallbackId: "02",
    shape: "blob",
    ...opts,
  });
  const origOn = ball.on.bind(ball);
  const origOff = typeof ball.off === "function" ? ball.off.bind(ball) : undefined;
  ball.celebrate = function (strength) {
    this.bounce();
    if (strength !== 0) this.spin(1);
    return this;
  };
  ball.signature = function () {
    this.spin(1);
    return true;
  };
  ball.on = function (ev, cb) {
    origOn(ev, cb);
    return () => {
      origOff?.(ev, cb);
    };
  };
  return ball;
}
