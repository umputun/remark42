/**
 * A store for tests that keeps the state it was given and records what is dispatched to it. A
 * function is called with dispatch and getState, as the widget's store calls it, so an action
 * creator can be run against a fixed state and its actions read back from getActions.
 */
// eslint-disable-next-line @typescript-eslint/no-explicit-any
export function mockStore<S = any>(state: S) {
  const actions: unknown[] = [];
  let listeners: Array<() => void> = [];

  const store = {
    getState: () => state,
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    dispatch(action: any): any {
      if (typeof action === 'function') {
        return action(store.dispatch, store.getState, undefined);
      }
      actions.push(action);
      listeners.forEach((listener) => listener());
      return action;
    },
    subscribe(listener: () => void) {
      listeners = [...listeners, listener];
      return () => {
        listeners = listeners.filter((l) => l !== listener);
      };
    },
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    getActions: (): any[] => actions,
  };

  return store;
}
