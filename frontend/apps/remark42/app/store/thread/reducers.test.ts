import type { Comment } from 'common/types';
import type { StoreState } from 'store';

import { setCollapse } from './actions';
import { THREAD_SET_COLLAPSE } from './types';

describe('collapsedThreads', () => {
  it('should set collapsed to true', () => {
    const comment = { id: 'some-id' } as Comment;
    const state = { collapsedThreads: {} } as StoreState;

    const dispatch = jest.fn();
    const getState = jest.fn(() => state);

    setCollapse(comment.id, true)(dispatch, getState, undefined);
    expect(dispatch).toHaveBeenCalledWith({
      type: THREAD_SET_COLLAPSE,
      id: 'some-id',
      collapsed: true,
    });
  });

  it('should collapse toggled', () => {
    const comment = { id: 'some-id' } as Comment;
    const node = { comment, replies: [] };
    const dispatch = jest.fn();
    const getState = jest.fn();
    getState.mockReturnValue({ collapsedThreads: { 'some-id': true }, comments: [node] });

    setCollapse(comment.id, false)(dispatch, getState, undefined);
    expect(dispatch).toHaveBeenCalledWith({
      type: THREAD_SET_COLLAPSE,
      id: 'some-id',
      collapsed: false,
    });
  });
});
