import type { Comment, Node } from 'common/types';

import { allComments, childComments } from './reducers';
import { COMMENTS_SET } from './types';

function node(patch: Partial<Comment>): Node {
  return {
    comment: {
      id: 'c1',
      pid: '',
      text: '<p>text</p>',
      user: { id: 'author', name: 'author', picture: '', admin: false, ip: '', block: false, verified: false },
      locator: { site: 'remark', url: 'https://example.com/post' },
      score: 0,
      vote: 0,
      controversy: 0,
      time: '2026-10-31T12:00:00Z',
      title: '',
      ...patch,
    } as Comment,
  };
}

describe('allComments', () => {
  it('takes the fetched vote and score of a comment nobody edited', () => {
    // what the widget holds for a reader before signing in, then the same comment fetched again
    // for the signed-in reader who had voted for it earlier
    const before = allComments({}, { type: COMMENTS_SET, comments: [node({ vote: 0, score: 0 })] });
    const after = allComments(before, { type: COMMENTS_SET, comments: [node({ vote: 1, score: 1 })] });

    expect(after.c1.vote).toBe(1);
    expect(after.c1.score).toBe(1);
  });

  it('keeps the stored object when the fetched comment is the same', () => {
    const before = allComments({}, { type: COMMENTS_SET, comments: [node({ vote: 1, score: 3 })] });
    const after = allComments(before, { type: COMMENTS_SET, comments: [node({ vote: 1, score: 3 })] });

    expect(after.c1).toBe(before.c1);
  });
});

/** a comment `c0` and `length` replies, each answering the one before it */
function chain(length: number): Node {
  let thread = node({ id: `c${length}`, pid: `c${length - 1}` });

  for (let i = length - 1; i >= 0; i--) {
    thread = { ...node({ id: `c${i}`, pid: i > 0 ? `c${i - 1}` : '' }), replies: [thread] };
  }

  return thread;
}

describe('a chain of replies longer than the stack is deep', () => {
  const DEEP = 20000;
  const comments = [chain(DEEP)];

  it('is stored whole', () => {
    const all = allComments({}, { type: COMMENTS_SET, comments });

    expect(Object.keys(all)).toHaveLength(DEEP + 1);
    expect(all[`c${DEEP}`].pid).toBe(`c${DEEP - 1}`);
  });

  it('has each reply listed under the comment it answers', () => {
    const childs = childComments({}, { type: COMMENTS_SET, comments });

    expect(Object.keys(childs)).toHaveLength(DEEP);
    expect(childs.c0).toEqual(['c1']);
    expect(childs[`c${DEEP - 1}`]).toEqual([`c${DEEP}`]);
  });
});

describe('childComments', () => {
  it('lists the replies to every comment in their order, and nothing for a comment that came without replies', () => {
    const reply = (id: string, pid: string, replies?: Node[]): Node => ({ ...node({ id, pid }), replies });
    const comments = [
      reply('a', '', [reply('a1', 'a', [reply('a1x', 'a1'), reply('a1y', 'a1')]), reply('a2', 'a', [])]),
      reply('b', ''),
    ];

    expect(childComments({}, { type: COMMENTS_SET, comments })).toEqual({
      a: ['a1', 'a2'],
      a1: ['a1x', 'a1y'],
      a2: [],
    });
  });
});
