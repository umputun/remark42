import { setCollapse } from './thread.actions';
import { collapsedThreads } from './thread.reducers';
import { getThreadIsCollapsed } from './thread.getters';

describe('collapsedThreads', () => {
  const comment = { id: 1 };

  it('should set collapsed to true', () => {
    const collapsed = true;
    const action = setCollapse(comment, collapsed);

    const newState = {
      collapsedThreads: collapsedThreads({}, action),
    };

    expect(getThreadIsCollapsed(newState, comment)).toEqual(collapsed);
  });

  it('should set collapsed to false', () => {
    const collapsed = false;
    const action = setCollapse(comment, collapsed);

    const newState = {
      collapsedThreads: collapsedThreads({}, action),
    };

    expect(getThreadIsCollapsed(newState, comment)).toEqual(collapsed);
  });
});
