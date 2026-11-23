import type { Comment, Node } from 'common/types';

/**
 * Tree operations over a thread, with nothing browser-shaped in them.
 *
 * Kept apart from utils.ts so they can be tested without a bundler: the rest of that module reads
 * localStorage at import time, which is enough to make the whole file unimportable outside a
 * browser however pure they are.
 */

/**
 * Calls `fn` for every node of the tree, a comment before the replies to it and the replies in
 * their order.
 *
 * The nodes still to visit are kept in a list. A function calling itself for the replies would need
 * a call stack as deep as the longest chain of replies, and a chain of a few thousand replies, each
 * answering the one before, is deeper than the stack.
 */
export function walkTree(tree: Node[], fn: (node: Node) => void): void {
  // the next node to visit is the last one
  const pending = tree.slice().reverse();

  while (pending.length > 0) {
    const node = pending.pop() as Node;

    fn(node);
    if (!node.replies) continue;
    for (let i = node.replies.length - 1; i >= 0; i--) {
      pending.push(node.replies[i]);
    }
  }
}

/** a list of replies being filtered: what it held, how far it was read and what is kept so far */
interface Filtering {
  nodes: Node[];
  next: number;
  kept: Node[];
  changed: boolean;
}

/**
 * Filters tree node
 *
 * Like `walkTree` it keeps its place in a list of its own instead of calling itself: the lists
 * being filtered, the tree's own first and the one being read last.
 */
export function filterTree(tree: Node[], fn: (node: Node) => boolean): Node[] {
  const lists: Filtering[] = [{ nodes: tree, next: 0, kept: [], changed: false }];

  for (;;) {
    const list = lists[lists.length - 1];

    if (list.next < list.nodes.length) {
      const node = list.nodes[list.next++];

      if (!fn(node)) {
        list.changed = true;
      } else if (!node.replies) {
        list.kept.push(node);
      } else {
        lists.push({ nodes: node.replies, next: 0, kept: [], changed: false });
      }
      continue;
    }

    // the list is read to its end. It stays the same array when nothing in it was dropped or rebuilt
    const filtered = list.changed ? list.kept : list.nodes;

    lists.pop();
    if (lists.length === 0) return filtered;

    // these were the replies to the node its parent list read last, which is rebuilt around them
    const parent = lists[lists.length - 1];

    parent.kept.push({ ...parent.nodes[parent.next - 1], replies: filtered });
    parent.changed = true;
  }
}

export function findPinnedComments(thread: Node): Comment[] {
  const result: Comment[] = [];

  walkTree([thread], ({ comment }) => {
    if (comment.pin) {
      result.push(comment);
    }
  });

  return result;
}

export function getPinnedComments(threads: Node[]): Comment[] {
  return threads.reduce((acc: Comment[], thread: Node) => acc.concat(findPinnedComments(thread)), []);
}

/**
 * Counts the comments of a tree, replies included, the way the server counts them for a page, or
 * only those `fn` accepts
 */
export function countComments(tree: Node[], fn: (comment: Comment) => boolean = () => true): number {
  let count = 0;

  walkTree(tree, ({ comment }) => {
    if (fn(comment)) count++;
  });

  return count;
}

/**
 * Drops the top-level threads already loaded. Pages are cut from an order that can shift between
 * requests, under score or activity, and a reader's own new comment is on screen before its page
 * comes, so a page can repeat a thread.
 */
export function dropLoadedThreads(tree: Node[], loaded: readonly Comment['id'][]): Node[] {
  const known = new Set(loaded);

  return tree.filter((node) => !known.has(node.comment.id));
}
