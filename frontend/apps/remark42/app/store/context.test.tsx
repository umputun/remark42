import { h } from 'preact';
import { render, screen, act } from '@testing-library/preact';
import { createStore } from 'redux';

import { Provider, useSelector } from './context';

type State = { value: number };

function reducer(state: State = { value: 0 }, action: { type: string }): State {
  return action.type === 'bump' ? { value: state.value + 1 } : state;
}

function Display() {
  const value = useSelector((s: State) => s.value);
  return <span data-testid="value">{value}</span>;
}

describe('useSelector', () => {
  it('renders the value at mount', () => {
    const store = createStore(reducer);
    render(
      <Provider store={store}>
        <Display />
      </Provider>
    );
    expect(screen.getByTestId('value').textContent).toBe('0');
  });

  it('re-renders on a store change', () => {
    const store = createStore(reducer);
    render(
      <Provider store={store}>
        <Display />
      </Provider>
    );
    act(() => {
      store.dispatch({ type: 'bump' });
    });
    expect(screen.getByTestId('value').textContent).toBe('1');
  });

  // the subscribe-time check must not re-run a selector that builds a fresh object,
  // or every connected component renders twice at mount
  it('renders once at mount with an object-building selector', () => {
    const store = createStore(reducer);
    let renders = 0;

    function Counting() {
      const { value } = useSelector((s: State) => ({ value: s.value }));

      renders += 1;

      return <span data-testid="value">{value}</span>;
    }

    render(
      <Provider store={store}>
        <Counting />
      </Provider>
    );

    expect(renders).toBe(1);
    expect(screen.getByTestId('value').textContent).toBe('0');
  });

  it('does not miss a dispatch that lands between render and subscribe', () => {
    const store = createStore(reducer);

    // dispatch from inside the component body, which runs during render and so
    // before the subscribing effect. this is the window a real async resolution
    // can land in, and a subscription created afterwards never hears about it.
    let dispatched = false;
    function Racing() {
      const value = useSelector((s: State) => s.value);
      if (!dispatched) {
        dispatched = true;
        store.dispatch({ type: 'bump' });
      }
      return <span data-testid="value">{value}</span>;
    }

    render(
      <Provider store={store}>
        <Racing />
      </Provider>
    );

    expect(store.getState().value).toBe(1);
    expect(screen.getByTestId('value').textContent).toBe('1');
  });
});
