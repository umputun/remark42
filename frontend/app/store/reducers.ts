import comments from './comments/reducers';
import * as postInfo from './post-info/reducers';
import * as theme from './theme/reducers';
import * as user from './user/reducers';
import * as userInfo from './user-info/reducers';
import * as thread from './thread/reducers';
import * as provider from './provider/reducers';

/** Merged store reducers */
const rootProvider = {
  comments,
  ...theme,
  ...postInfo,
  ...provider,
  ...userInfo,
  ...thread,
  ...user,
};

export default rootProvider;
