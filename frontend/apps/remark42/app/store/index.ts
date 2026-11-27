import { useDispatch, useSelector, type TypedUseSelectorHook } from './context';
import { createStore, combineReducers } from './create-store';
import { rootProvider } from './reducers';
import type { ACTIONS } from './actions';

const reducers = combineReducers(rootProvider);
export const store = createStore(reducers);

export type StoreState = ReturnType<typeof store.getState>;

/** an action creator that needs the store: dispatch calls it and hands back what it returns */
export type StoreAction<ReturnType = void> = (
  dispatch: StoreDispatch,
  getState: () => StoreState,
  extraArgument: undefined
) => ReturnType;

/** dispatch an action, or run a StoreAction and get its result */
export interface StoreDispatch {
  <R>(action: StoreAction<R>): R;
  <A extends ACTIONS>(action: A): A;
}

export const useAppDispatch: () => StoreDispatch = useDispatch;
export const useAppSelector: TypedUseSelectorHook<StoreState> = useSelector;

if (process.env.NODE_ENV === 'development') {
  window.ReduxStore = store;
}
