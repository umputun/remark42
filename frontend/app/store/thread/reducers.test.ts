import { Comment } from 'common/types';

import { setCollapse } from './actions';
import { THREAD_SET_COLLAPSE } from './types';

describe('collapsedThreads', () => {
  it('should set collapsed to true', () => {
    const comment = { id: 'some-id' } as Comment;
    const node = { comment, replies: [] };
    const state = { collapsedThreads: {}, comments: [node] };

    const dispatch = jest.fn();
    const getState = jest.fn(() => state) as any;
    setCollapse(comment.id, true)(dispatch, getState, undefined);
    expect(dispatch).toBeCalledWith({
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
    expect(dispatch).toBeCalledWith({
      type: THREAD_SET_COLLAPSE,
      id: 'some-id',
      collapsed: false,
    });
  });
});
