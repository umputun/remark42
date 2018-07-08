import { createStore, applyMiddleware } from 'redux';
import { combineReducers } from 'redux';

import { threadReducers, threadMiddlewares } from './components/thread';

const reducers = combineReducers({
  ...threadReducers,
});

const middlewares = applyMiddleware(...threadMiddlewares);

export default createStore(reducers, middlewares);
