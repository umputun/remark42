import { collapsedThreads } from './thread.reducers';
import collapsedCommentsMiddleware from './collapsedCommentsMiddleware';

export const threadReducers = { collapsedThreads };

export const threadMiddlewares = [collapsedCommentsMiddleware];

export { default } from './thread';
