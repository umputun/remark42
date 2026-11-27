import en from 'locales/en.json';
import { createIntl } from 'common/intl-message';
import { combineReducers, createStore } from 'store/create-store';
import { patchComment } from 'store/comments/actions';
import { allComments } from 'store/comments/reducers';
import type { Comment } from 'common/types';

import { mountVotes } from './votes.js';

// loading preact from anything this file imports fails the run, so the case below proves the
// vote buttons work with no preact at all
jest.mock('preact', () => {
  throw new Error('preact was loaded');
});
jest.mock('preact/hooks', () => {
  throw new Error('preact/hooks was loaded');
});

it('votes on the store alone, with no preact loaded', async () => {
  const store = createStore(combineReducers({ allComments }));
  store.dispatch(patchComment({ id: 'c1', score: 3, vote: 0 } as Comment));
  const select = () => {
    const comment = store.getState().allComments.c1;
    return { id: 'c1', vote: comment.vote, votes: comment.score, controversy: undefined };
  };
  const root = document.createElement('span');
  const votes = mountVotes(root, select(), {
    intl: createIntl('en', en),
    putVote: () => Promise.resolve({ id: 'c1', score: 4 }),
    patch: (patch) => store.dispatch(patchComment(patch)),
    positiveScore: () => false,
  });
  store.subscribe(() => votes.update(select()));

  (root.querySelector('[title="Vote up"]') as HTMLButtonElement).click();
  await Promise.resolve();
  await Promise.resolve();

  expect(store.getState().allComments.c1).toMatchObject({ score: 4, vote: 1 });
  expect(root.querySelector('[title="Score"]')!.textContent).toBe('4');
  expect(root.querySelector('[title="Vote up"]')!.className).toContain('upVoteButtonActive');
});
