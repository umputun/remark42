import { IS_STORAGE_AVAILABLE } from 'common/constants';

const failMessage = 'remark42: localStorage access denied, check browser preferences';

export const setItem = IS_STORAGE_AVAILABLE
  ? localStorage.setItem.bind(localStorage)
  : () => {
      console.error(failMessage); // eslint-disable-line no-console
    };

export const getItem = IS_STORAGE_AVAILABLE
  ? localStorage.getItem.bind(localStorage)
  : () => {
      console.error(failMessage); // eslint-disable-line no-console
      return null;
    };

export const removeItem = IS_STORAGE_AVAILABLE
  ? localStorage.removeItem.bind(localStorage)
  : () => {
      console.error(failMessage); // eslint-disable-line no-console
    };
