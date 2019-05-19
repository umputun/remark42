import comments from './comments/reducers';
import postinfo from './post_info/reducers';
import sort from './sort/reducers';
import theme from './theme/reducers';
import user from './user/reducers';
import userInfo from './user-info/reducers';
import thread from './thread/reducers';

/** Merged store reducers */
export default {
  ...comments,
  ...postinfo,
  ...sort,
  ...theme,
  ...user,
  ...userInfo,
  ...thread,
};
