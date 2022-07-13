import * as comments from './comments/reducers';
import * as postInfo from './post-info/reducers';
import * as theme from './theme/reducers';
import * as user from './user/reducers';
import * as thread from './thread/reducers';

/** Merged store reducers */
export const rootProvider = {
  ...comments,
  ...theme,
  ...postInfo,
  ...thread,
  ...user,
};
