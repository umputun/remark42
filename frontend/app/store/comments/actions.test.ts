import { mockStore } from '@app/testUtils/mockStore';
import { LS_SORT_KEY } from '@app/common/constants';

import { updateSorting } from './actions';
import { COMMENTS_SET_SORT } from './types';

describe('Store comments actions', () => {
  beforeAll(() => {
    require('jest-fetch-mock').enableMocks();
  });
  afterAll(() => {
    require('jest-fetch-mock').resetMocks();
  });

  it('handles changing a purchase status and fetches all purchases', async () => {
    const newSort = '+controversy';
    const store = mockStore({
      comments: {
        sort: '+active',
      },
      hiddenUsers: {},
    });

    await store.dispatch(updateSorting(newSort));

    const [setCommentAction] = store.getActions();

    expect(setCommentAction).toEqual({ type: COMMENTS_SET_SORT, payload: newSort });
    expect(localStorage.setItem).toHaveBeenCalledWith(LS_SORT_KEY, newSort);
  });
});
