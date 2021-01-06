import { Node } from 'common/types';

/**
 * Function to debug node tree.
 */
export function debugNode(n: Node): Node {
  const d = (n: Node, level: number): void => {
    // eslint-disable-next-line no-console
    console.log(
      `${'  '.repeat(level)}${n.comment.text.trim()} | id: ${n.comment.id} | delete: ${n.comment.delete} | pin: ${
        n.comment.pin
      }`
    );
    if (n.replies) {
      for (const node of n.replies) {
        d(node, level + 1);
      }
    }
  };
  d(n, 0);
  return n;
}
