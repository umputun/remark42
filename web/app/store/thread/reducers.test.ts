import { Comment } from '@app/common/types';

import { setCollapse } from './actions';
import { THREAD_SET_COLLAPSE } from './types';

describe('collapsedThreads', () => {
  const comment = { id: 'some-id' } as Comment;

  it('should set collapsed to true', () => {
    const state = { collapsedThreads: {} };

    const dispatch = jest.fn();
    const getState = jest.fn(() => state) as any;
    setCollapse(comment.id)(dispatch, getState, undefined);
    expect(dispatch).toBeCalledWith({
      type: THREAD_SET_COLLAPSE,
      id: 'some-id',
      collapsed: true,
    });
  });

  it('should collapse toggled', () => {
    const dispatch = jest.fn();
    const getState = jest.fn();
    getState.mockReturnValue({ collapsedThreads: { 'some-id': true } });

    setCollapse(comment.id)(dispatch, getState, undefined);
    expect(dispatch).toBeCalledWith({
      type: THREAD_SET_COLLAPSE,
      id: 'some-id',
      collapsed: false,
    });
  });
});
