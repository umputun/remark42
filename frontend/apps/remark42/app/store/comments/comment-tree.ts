import type { Comment, Node } from 'common/types';

/**
 * Tree operations over a thread, with nothing browser-shaped in them.
 *
 * Kept apart from utils.ts so they can be tested without a bundler: the rest of that module reads
 * localStorage at import time, which is enough to make the whole file unimportable outside a
 * browser however pure these three are.
 */

/**
 * Filters tree node
 */
export function filterTree(tree: Node[], fn: (node: Node) => boolean): Node[] {
  let filtered = false;
  const newTree = tree.reduce<Node[]>((tree, node) => {
    if (!fn(node)) {
      filtered = true;
      return tree;
    }
    const newNode: Node = !node.replies ? node : { ...node, replies: filterTree(node.replies, fn) };
    if (newNode !== node) {
      filtered = true;
    }
    tree.push(newNode);
    return tree;
  }, []);
  if (!filtered) return tree;
  return newTree;
}

export function findPinnedComments(thread: Node): Comment[] {
  let result: Comment[] = [];

  if (thread.comment.pin) {
    result = result.concat(thread.comment);
  }

  if (thread.replies) {
    result = result.concat(
      thread.replies.reduce((acc: Comment[], thread: Node) => acc.concat(findPinnedComments(thread)), [])
    );
  }

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
  return tree.reduce(
    (count, node) => count + (fn(node.comment) ? 1 : 0) + (node.replies ? countComments(node.replies, fn) : 0),
    0
  );
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
