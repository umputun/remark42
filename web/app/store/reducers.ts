import comments from './comments/reducers';
import postinfo from './post_info/reducers';
import theme from './theme/reducers';
import user from './user/reducers';
import userInfo from './user-info/reducers';
import thread from './thread/reducers';
import sort from './sort/reducers';

/** Merged store reducers */
export default {
  ...sort,
  ...comments,
  ...postinfo,
  ...theme,
  ...user,
  ...userInfo,
  ...thread,
};
