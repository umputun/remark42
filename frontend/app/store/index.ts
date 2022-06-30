import { createStore, applyMiddleware, AnyAction, compose, combineReducers } from 'redux';
import { useDispatch, useSelector, TypedUseSelectorHook } from 'react-redux';
import thunk, { ThunkAction, ThunkDispatch } from 'redux-thunk';
import { rootProvider } from './reducers';
import { ACTIONS } from './actions';

const reducers = combineReducers(rootProvider);
const middleware = applyMiddleware(thunk);
export const store = createStore(reducers, compose(middleware));

export type StoreState = ReturnType<typeof store.getState>;
export type StoreDispatch = ThunkDispatch<StoreState, undefined, ACTIONS>;
export type StoreAction<ReturnType = void> = ThunkAction<ReturnType, StoreState, unknown, AnyAction>;

export const useAppDispatch: () => StoreDispatch = useDispatch;
export const useAppSelector: TypedUseSelectorHook<StoreState> = useSelector;

if (process.env.NODE_ENV === 'development') {
  window.ReduxStore = store;
}
