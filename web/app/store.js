import { createStore, applyMiddleware } from 'redux';
import { combineReducers } from 'redux';

import { threadReducers, threadMiddlewares } from './components/thread';
import { userInfoReducers } from './components/user-info';

const reducers = combineReducers({
  ...threadReducers,
  ...userInfoReducers,
});

const middlewares = applyMiddleware(...threadMiddlewares);

export default createStore(reducers, middlewares);
