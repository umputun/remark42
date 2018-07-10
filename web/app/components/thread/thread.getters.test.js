import store from 'common/store';
import { getThreadIsCollapsed } from './thread.getters';

describe('collapsedThreads', () => {
  const notScoredComment = { id: 1 };
  const goodComment = { id: 1, score: 3 };
  const badComment = { id: 1, score: -1 };

  it('takes value from the state', () => {
    const state = { collapsedThreads: { [notScoredComment.id]: true } };
    const collapsed = getThreadIsCollapsed(state, notScoredComment);
    expect(collapsed).toEqual(true);
  });

  beforeAll(() => {
    const config = { critical_score: 2 };
    store.set('config', config);
  });

  it('returns true when score is less then critical_score', () => {
    const state = { collapsedThreads: {} };
    const collapsed = getThreadIsCollapsed(state, badComment);
    expect(collapsed).toEqual(true);
  });

  it('returns true when score is less then critical_score', () => {
    const state = { collapsedThreads: {} };
    const collapsed = getThreadIsCollapsed(state, goodComment);
    expect(collapsed).toEqual(false);
  });
});
