import { getInitialSort } from './utils';
import { DEFAULT_SORT, LS_SORT_KEY } from 'common/constants';

describe('store comments utils', () => {
  beforeEach(() => {
    localStorage.clear();
  });

  describe('getInitialSort', () => {
    it('should return default sorting', () => {
      expect(getInitialSort()).toBe(DEFAULT_SORT);
    });

    it('should return value from local storage', () => {
      const currentSort = '+active';

      localStorage.setItem(LS_SORT_KEY, currentSort);
      expect(getInitialSort()).toBe(currentSort);
      expect(localStorage.getItem).toHaveBeenCalledWith(LS_SORT_KEY);
    });
  });
});
