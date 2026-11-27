// The widget's store, in place of redux and redux-thunk: the three functions the widget calls, with
// the behaviour it relies on and nothing else.

/**
 * @typedef {{ type: string }} Action
 */

/**
 * @template S
 * @typedef {(state: S | undefined, action: any) => S} Reducer
 */

/**
 * A store holding the state reducer computes. dispatch takes an action, or a function, which it
 * calls with dispatch and getState and whose return value it hands back, as redux-thunk does.
 * Listeners are called after every action, in the order they subscribed; one subscribed or
 * removed while they run takes effect from the next action.
 *
 * @template S
 * @param {Reducer<S>} reducer
 */
export function createStore(reducer) {
  let state = reducer(undefined, { type: '@@remark42/INIT' });
  // the listeners an action is announced to, and the list subscriptions change. The second is
  // copied only on the first change after an action, so a change made while listeners run leaves
  // that action's list alone, and a thread of thousands of comments, each subscribing as it
  // mounts, does not copy the list for every one of them
  /** @type {Array<() => void> | null} */
  let current = [];
  let next = /** @type {Array<() => void>} */ (current);

  function changeable() {
    if (next === current) {
      next = current.slice();
    }
    return next;
  }

  /** @returns {S} */
  function getState() {
    return state;
  }

  /**
   * @param {any} action
   * @returns {any}
   */
  function dispatch(action) {
    if (typeof action === 'function') {
      return action(dispatch, getState, undefined);
    }
    state = reducer(state, action);
    current = next;
    for (const listener of current) {
      listener();
    }
    return action;
  }

  /**
   * @param {() => void} listener
   * @returns {() => void} removes the listener; calling it again does nothing
   */
  function subscribe(listener) {
    changeable().push(listener);
    let subscribed = true;

    return () => {
      if (!subscribed) {
        return;
      }
      subscribed = false;
      const listeners = changeable();
      listeners.splice(listeners.indexOf(listener), 1);
      // the last action's list may still hold it; dropped, so a removed listener and what it
      // holds can be collected before the next action. An action whose listeners are running
      // keeps its own reference to the list
      current = null;
    };
  }

  return { getState, dispatch, subscribe };
}

/**
 * One reducer over an object of reducers, each owning the key it is stored under. When no reducer
 * changes its part, the state object itself is returned unchanged: the store's selectors take an
 * unchanged object to mean that nothing has to render.
 *
 * @template {Record<string, Reducer<any>>} R
 * @param {R} reducers
 * @returns {Reducer<{ [K in keyof R]: ReturnType<R[K]> }>}
 */
export function combineReducers(reducers) {
  const keys = Object.keys(reducers);

  return (state, action) => {
    const previous = /** @type {Record<string, unknown>} */ (state ?? {});
    /** @type {Record<string, unknown>} */
    const next = {};
    let changed = state === undefined;

    for (const key of keys) {
      next[key] = reducers[key](previous[key], action);
      changed = changed || next[key] !== previous[key];
    }

    return /** @type {any} */ (changed ? next : previous);
  };
}

/**
 * Each action creator wrapped so that calling it dispatches what it returns.
 *
 * @template {Record<string, (...args: any[]) => any>} C
 * @param {C} creators
 * @param {(action: any) => any} dispatch
 * @returns {{ [K in keyof C]: (...args: Parameters<C[K]>) => any }}
 */
export function bindActionCreators(creators, dispatch) {
  /** @type {Record<string, (...args: any[]) => any>} */
  const bound = {};

  for (const [name, create] of Object.entries(creators)) {
    bound[name] = (...args) => dispatch(create(...args));
  }

  return /** @type {any} */ (bound);
}
