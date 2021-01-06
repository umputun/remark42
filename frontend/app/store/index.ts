import { createStore, applyMiddleware, AnyAction, compose, combineReducers } from 'redux';
import thunk, { ThunkAction, ThunkDispatch } from 'redux-thunk';
import storeReducers from './reducers';
import { ACTIONS } from './actions';

const reducers = combineReducers(storeReducers);

export type StoreState = ReturnType<typeof reducers>;

const middleware = applyMiddleware(thunk);

/**
 * Thunk Action shortcut
 */
export type StoreAction<R, A extends AnyAction = ACTIONS> = ThunkAction<R, StoreState, undefined, A>;

/**
 * Thunk Dispatch shortcut
 */
export type StoreDispatch = ThunkDispatch<StoreState, undefined, ACTIONS>;

// eslint-disable-next-line @typescript-eslint/no-explicit-any
const composeEnhancers = (window as any).__REDUX_DEVTOOLS_EXTENSION_COMPOSE__
  ? // eslint-disable-next-line @typescript-eslint/no-explicit-any
    (window as any).__REDUX_DEVTOOLS_EXTENSION_COMPOSE__
  : compose;
const store = createStore(reducers, composeEnhancers(middleware));

if (process.env.NODE_ENV === 'development') {
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  (window as any).ReduxStore = store;
}

export default store;
