import assert from 'node:assert/strict';
import { describe, it } from 'node:test';

import type { Comment, Node } from '../../common/types.ts';

import { countComments, dropLoadedThreads, filterTree, findPinnedComments, getPinnedComments } from './comment-tree.ts';

function node(id: string, replies?: Node[], pin?: boolean): Node {
  return { comment: { id, pin } as Comment, replies };
}

function ids(nodes: Node[]): string[] {
  return nodes.map((n) => n.comment.id);
}

/** a comment `c0` and `length` replies, each answering the one before it; the last one can be pinned */
function chain(length: number, pinLast = false): Node {
  let thread = node(`c${length}`, undefined, pinLast);

  for (let i = length - 1; i >= 0; i--) {
    thread = node(`c${i}`, [thread]);
  }

  return thread;
}

/** far more replies, one under another, than a function calling itself for each has stack for */
const DEEP = 20000;

/**
 * filterTree is what hides a blocked user's comments, and it hides their replies' parents' place
 * in the thread along with them, so it has to walk the whole tree instead of the roots.
 *
 * It returns the very same array when nothing was dropped, but only for a tree of roots without
 * replies: a node carrying replies is rebuilt whichever way the predicate goes, which sets the
 * filtered flag and produces a new array. Its one caller filters a tree that is new on every fetch
 * and nothing compares the result against a previous one, so the identity decides no re-render; it
 * is pinned because it is what the function does.
 */
describe('filterTree', () => {
  it('returns the very same array when nothing is filtered and no node has replies', () => {
    const tree = [node('a'), node('b')];

    assert.equal(
      filterTree(tree, () => true),
      tree
    );
  });

  // a parent is rebuilt whether or not the predicate drops anything under it, so the identity above
  // does not survive one. stated because the shape of the claim matters more than the claim
  it('returns a new array when a node has replies, even with nothing filtered', () => {
    const tree = [node('a', [node('a1')])];

    assert.notEqual(
      filterTree(tree, () => true),
      tree
    );
  });

  it('drops a root the predicate rejects', () => {
    const tree = [node('a'), node('b'), node('c')];

    assert.deepEqual(ids(filterTree(tree, (n) => n.comment.id !== 'b')), ['a', 'c']);
  });

  it('drops a reply and keeps its parent', () => {
    const tree = [node('a', [node('a1'), node('a2')])];
    const filtered = filterTree(tree, (n) => n.comment.id !== 'a1');

    assert.deepEqual(ids(filtered), ['a']);
    assert.deepEqual(ids(filtered[0].replies ?? []), ['a2']);
  });

  it('recurses past the first level of replies', () => {
    const tree = [node('a', [node('a1', [node('a1x'), node('a1y')])])];
    const filtered = filterTree(tree, (n) => n.comment.id !== 'a1x');

    assert.deepEqual(ids(filtered[0].replies?.[0].replies ?? []), ['a1y']);
  });

  it('rebuilds the parent instead of mutating it when a reply goes', () => {
    const tree = [node('a', [node('a1')])];
    const filtered = filterTree(tree, (n) => n.comment.id !== 'a1');

    assert.notEqual(filtered[0], tree[0]);
    assert.deepEqual(ids(tree[0].replies ?? []), ['a1']);
  });

  it('leaves a node without replies alone', () => {
    const tree = [node('a')];

    assert.equal(filterTree(tree, () => true)[0].replies, undefined);
  });

  it('drops a parent even when its replies survive the predicate', () => {
    const tree = [node('a', [node('a1')])];

    assert.deepEqual(ids(filterTree(tree, (n) => n.comment.id !== 'a')), []);
  });

  it('returns an empty tree unchanged', () => {
    const tree: Node[] = [];

    assert.equal(
      filterTree(tree, () => false),
      tree
    );
  });

  it('filters a chain of replies longer than the stack is deep', () => {
    const tree = [chain(DEEP)];
    const filtered = filterTree(tree, (n) => n.comment.id !== `c${DEEP}`);

    let last = filtered[0];
    let length = 0;
    while (last.replies?.length) {
      last = last.replies[0];
      length++;
    }

    // the last reply is gone, and the one it answered is left with an empty list of replies
    assert.equal(length, DEEP - 1);
    assert.equal(last.comment.id, `c${DEEP - 1}`);
    assert.deepEqual(last.replies, []);
  });
});

/**
 * Pinned comments are lifted out of the thread and shown above it, so they are collected from
 * every depth and not only from the roots.
 */
describe('findPinnedComments', () => {
  it('finds nothing in a thread with no pins', () => {
    assert.deepEqual(findPinnedComments(node('a', [node('a1')])), []);
  });

  it('finds a pinned root', () => {
    assert.deepEqual(
      findPinnedComments(node('a', undefined, true)).map((c) => c.id),
      ['a']
    );
  });

  it('finds a pin nested two levels down', () => {
    const thread = node('a', [node('a1', [node('a1x', undefined, true)])]);

    assert.deepEqual(
      findPinnedComments(thread).map((c) => c.id),
      ['a1x']
    );
  });

  it('returns the parent before its replies', () => {
    const thread = node('a', [node('a1', undefined, true)], true);

    assert.deepEqual(
      findPinnedComments(thread).map((c) => c.id),
      ['a', 'a1']
    );
  });

  it('returns the replies to one comment before the replies to the next', () => {
    const thread = node('a', [node('a1', [node('a1x', undefined, true)], true), node('a2', undefined, true)]);

    assert.deepEqual(
      findPinnedComments(thread).map((c) => c.id),
      ['a1', 'a1x', 'a2']
    );
  });

  it('finds a pin at the end of a chain of replies longer than the stack is deep', () => {
    assert.deepEqual(
      findPinnedComments(chain(DEEP, true)).map((c) => c.id),
      [`c${DEEP}`]
    );
  });
});

describe('getPinnedComments', () => {
  it('collects across every thread, in order', () => {
    const threads = [node('a', [node('a1', undefined, true)]), node('b', undefined, true), node('c')];

    assert.deepEqual(
      getPinnedComments(threads).map((c) => c.id),
      ['a1', 'b']
    );
  });

  it('returns nothing for no threads', () => {
    assert.deepEqual(getPinnedComments([]), []);
  });
});

/** the number a page of the thread is measured in on the server: every comment, replies included */
describe('countComments', () => {
  it('counts the replies at every depth', () => {
    assert.equal(countComments([node('a', [node('b', [node('c')]), node('d')]), node('e')]), 5);
  });

  it('counts an empty tree as none', () => {
    assert.equal(countComments([]), 0);
  });

  it('counts only the comments the predicate accepts, at every depth', () => {
    const tree = [node('a', [node('b', [node('c', undefined, true)]), node('d')]), node('e', undefined, true)];

    assert.equal(
      countComments(tree, (comment) => !comment.pin),
      3
    );
  });

  it('counts a chain of replies longer than the stack is deep', () => {
    assert.equal(countComments([chain(DEEP)]), DEEP + 1);
  });
});

describe('dropLoadedThreads', () => {
  it('drops the top-level threads already loaded and keeps the order of the rest', () => {
    assert.deepEqual(ids(dropLoadedThreads([node('a'), node('b'), node('c')], ['b'])), ['a', 'c']);
  });

  it('compares top-level ids only, so a loaded id among the replies keeps its thread', () => {
    const tree = [node('a', [node('b')])];
    assert.deepEqual(ids(dropLoadedThreads(tree, ['b'])), ['a']);
  });
});
