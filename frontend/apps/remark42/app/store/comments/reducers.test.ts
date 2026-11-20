import type { Comment, Node } from 'common/types';

import { allComments } from './reducers';
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
