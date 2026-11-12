/**
 * 拥有：把导师对话里一次 learn_quiz 工具调用画成题卡，并把学习者在卡片上说的话交回那段对话。
 * 不拥有：题卡本身的作答和判定，那在 LearnQuestionCard。工具结果的解析在 lib/learn-quiz。
 */

import type { Item } from "../../../lib/protocol";
import { emitLearnSay } from "../../../lib/learn-events";
import { quizPostedOf } from "../../../lib/learn-quiz";
import { LearnQuestionCard } from "./LearnQuestionCard";

export { isLearnQuizItem } from "../../../lib/learn-quiz";

/** LearnQuizCard 在工具结果到了之后画出题卡。结果还没到，或读不出题目时什么都不画。判完后把学习者的作答发回这段对话，导师接着往下走。 */
export function LearnQuizCard(props: { call?: Item; result?: Item }) {
  const posted = quizPostedOf(props.result);
  if (!posted) return null;
  const sessionId = props.result?.sessionId || props.call?.sessionId || "";
  return (
    <LearnQuestionCard
      pathId={posted.pathId}
      questionId={posted.question.id}
      initial={posted.question}
      sessionId={sessionId}
      onGraded={(said) => emitLearnSay({ sessionId, text: said })}
    />
  );
}
