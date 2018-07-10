import { setCollapse } from './thread.actions';
import { collapsedThreads } from './thread.reducers';

describe('collapsedThreads', () => {
  const comment = { id: 1 };

  it('should set collapsed to true', () => {
    const action = setCollapse(comment, true);
    const newState = collapsedThreads({}, action);
    expect(newState).toEqual({ [comment.id]: true });
  });

  it('should set collapsed to false', () => {
    const action = setCollapse(comment, false);
    const newState = collapsedThreads({}, action);
    expect(newState).toEqual({ [comment.id]: false });
  });
});
