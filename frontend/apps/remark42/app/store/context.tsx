import { createContext, h, type ComponentChildren } from 'preact';
import { useContext, useEffect, useRef, useState } from 'preact/hooks';
import type { Store, Dispatch, Action } from 'redux';

/**
 * Minimal store binding over preact context, replacing react-redux.
 *
 * Only what the widget uses: a provider, dispatch, and a selector with an optional
 * equality function.
 */

const StoreContext = createContext<Store | null>(null);

export function Provider<S, A extends Action>({
  store,
  children,
}: {
  store: Store<S, A>;
  children?: ComponentChildren;
}) {
  return <StoreContext.Provider value={store as unknown as Store}>{children}</StoreContext.Provider>;
}

function useStore<S, A extends Action>(): Store<S, A> {
  const store = useContext(StoreContext);
  if (!store) {
    throw new Error('store accessed outside of a Provider');
  }
  return store as unknown as Store<S, A>;
}

export function useDispatch<D extends Dispatch = Dispatch>(): D {
  return useStore().dispatch as D;
}

export function useSelector<S, R>(selector: (state: S) => R, equalityFn?: (a: R, b: R) => boolean): R {
  const store = useStore<S, Action>();
  const [, setTick] = useState(0);

  const selected = selector(store.getState());

  // refs keep the subscription stable while always comparing against the latest
  // render's selector and value, so a changed selector cannot resurrect a stale result
  const selectorRef = useRef(selector);
  const equalityRef = useRef(equalityFn);
  const selectedRef = useRef(selected);
  selectorRef.current = selector;
  equalityRef.current = equalityFn;
  selectedRef.current = selected;

  useEffect(
    () =>
      store.subscribe(() => {
        const next = selectorRef.current(store.getState());
        const equal = equalityRef.current
          ? equalityRef.current(selectedRef.current, next)
          : Object.is(selectedRef.current, next);

        if (!equal) {
          selectedRef.current = next;
          setTick((n) => n + 1);
        }
      }),
    [store]
  );

  return selected;
}

export function shallowEqual(a: unknown, b: unknown): boolean {
  if (Object.is(a, b)) {
    return true;
  }
  if (typeof a !== 'object' || a === null || typeof b !== 'object' || b === null) {
    return false;
  }

  const aKeys = Object.keys(a);
  const bKeys = Object.keys(b);

  return (
    aKeys.length === bKeys.length &&
    aKeys.every(
      (key) =>
        Object.prototype.hasOwnProperty.call(b, key) &&
        Object.is((a as Record<string, unknown>)[key], (b as Record<string, unknown>)[key])
    )
  );
}

export type TypedUseSelectorHook<S> = <R>(selector: (state: S) => R, equalityFn?: (a: R, b: R) => boolean) => R;
