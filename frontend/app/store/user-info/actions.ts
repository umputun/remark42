import { getUserComments } from 'common/api';
import { Comment } from 'common/types';

import { StoreAction } from '../index';
import { USER_INFO_SET } from './types';

export const fetchInfo =
  (id: string): StoreAction<Promise<Comment[] | null>> =>
  async (dispatch) => {
    // TODO: limit
    const info = await getUserComments(id, 10);

    dispatch({
      type: USER_INFO_SET,
      id,
      comments: info.comments,
    });
    return info.comments;
  };
