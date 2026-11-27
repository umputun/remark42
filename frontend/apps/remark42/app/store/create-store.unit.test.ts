import assert from 'node:assert/strict';
import { describe, it } from 'node:test';

import { bindActionCreators, combineReducers, createStore } from './create-store.js';

type Counter = { type: 'add'; by: number } | { type: 'other' };

const count = (state = 0, action: Counter) => (action.type === 'add' ? state + action.by : state);
const label = (state = 'x', action: Counter) => (action.type === 'other' ? `${state}!` : state);

describe('createStore', () => {
  it('starts from what the reducer returns for no state', () => {
    assert.equal(createStore(count).getState(), 0);
  });

  it('runs an action through the reducer and hands the action back', () => {
    const store = createStore(count);
    const action = { type: 'add', by: 2 } as const;

    assert.equal(store.dispatch(action), action);
    assert.equal(store.getState(), 2);
  });

  it('calls a function with dispatch and getState, and hands back what it returns', async () => {
    const store = createStore(count);

    const result = await store.dispatch(async (dispatch: (a: Counter) => void, getState: () => number) => {
      dispatch({ type: 'add', by: 3 });
      return getState() * 10;
    });

    assert.equal(result, 30);
    assert.equal(store.getState(), 3);
  });

  it('tells every listener after each action, and a removed one no more', () => {
    const store = createStore(count);
    const calls: string[] = [];
    const removeA = store.subscribe(() => calls.push(`a${store.getState()}`));
    store.subscribe(() => calls.push(`b${store.getState()}`));

    store.dispatch({ type: 'add', by: 1 });
    removeA();
    removeA();
    store.dispatch({ type: 'add', by: 1 });

    assert.deepEqual(calls, ['a1', 'b1', 'b2']);
  });

  it('applies a subscription made during an action from the next action on', () => {
    const store = createStore(count);
    const calls: string[] = [];
    store.subscribe(() => {
      calls.push('outer');
      store.subscribe(() => calls.push('inner'));
    });

    store.dispatch({ type: 'add', by: 1 });
    assert.deepEqual(calls, ['outer']);
    store.dispatch({ type: 'add', by: 1 });
    assert.deepEqual(calls, ['outer', 'outer', 'inner']);
  });

  it('still tells the others about the action during which a listener removed itself', () => {
    const store = createStore(count);
    const calls: string[] = [];
    const removeA = store.subscribe(() => {
      calls.push('a');
      removeA();
    });
    store.subscribe(() => calls.push('b'));
    store.subscribe(() => calls.push('c'));

    store.dispatch({ type: 'add', by: 1 });
    store.dispatch({ type: 'add', by: 1 });

    assert.deepEqual(calls, ['a', 'b', 'c', 'b', 'c']);
  });

  it('runs an action dispatched by a listener before the rest hear the first one', () => {
    const store = createStore(count);
    const calls: string[] = [];
    let first = true;
    store.subscribe(() => {
      calls.push(`a${store.getState()}`);
      if (first) {
        first = false;
        store.subscribe(() => calls.push(`c${store.getState()}`));
        store.dispatch({ type: 'add', by: 1 });
      }
    });
    store.subscribe(() => calls.push(`b${store.getState()}`));

    store.dispatch({ type: 'add', by: 2 });

    assert.deepEqual(calls, ['a2', 'a3', 'b3', 'c3', 'b3']);
  });

  it('removes only the subscription it was given, once, when one listener is subscribed twice', () => {
    const store = createStore(count);
    let calls = 0;
    const listener = () => calls++;
    const first = store.subscribe(listener);
    store.subscribe(listener);

    first();
    first();
    store.dispatch({ type: 'add', by: 1 });

    assert.equal(calls, 1);
  });
});

describe('combineReducers', () => {
  const both = combineReducers({ count, label });

  it('builds the initial state from each reducer', () => {
    assert.deepEqual(both(undefined, { type: 'other' } as Counter), { count: 0, label: 'x!' });
  });

  it('returns the same object when no part changed', () => {
    const state = { count: 1, label: 'y' };

    assert.equal(both(state, { type: 'add', by: 0 }), state);
  });

  it('returns a new object when a part changed, leaving the rest as they were', () => {
    const state = { count: 1, label: 'y' };
    const next = both(state, { type: 'add', by: 1 });

    assert.notEqual(next, state);
    assert.deepEqual(next, { count: 2, label: 'y' });
  });
});

describe('bindActionCreators', () => {
  it('dispatches what each creator returns, with the arguments it was called with', () => {
    const store = createStore(count);
    const bound = bindActionCreators({ add: (by: number) => ({ type: 'add', by }) as const }, store.dispatch);

    bound.add(5);

    assert.equal(store.getState(), 5);
  });
});
