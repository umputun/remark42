import { getInitialSort } from './utils';
import { LS_SORT_KEY } from 'common/constants';

jest.mock('common/settings', () => ({ defaultSort: '-time' }));

describe('store comments utils', () => {
  beforeEach(() => {
    localStorage.clear();
  });

  describe('getInitialSort', () => {
    it('should return the configured default sorting', () => {
      expect(getInitialSort()).toBe('-time');
    });

    it('should return value from local storage', () => {
      const currentSort = '+active';

      localStorage.setItem(LS_SORT_KEY, currentSort);
      expect(getInitialSort()).toBe(currentSort);
      expect(localStorage.getItem).toHaveBeenCalledWith(LS_SORT_KEY);
    });
  });
});
