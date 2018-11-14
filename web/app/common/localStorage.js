import { isAvailable } from 'common/constants';

const failMessage = 'remark42: localStorage access denied, check browser preferences';

export const setItem = isAvailable
  ? localStorage.setItem.bind(localStorage)
  : () => {
      console.error(failMessage); // eslint-disable-line no-console
    };

export const getItem = isAvailable
  ? localStorage.getItem.bind(localStorage)
  : () => {
      console.error(failMessage); // eslint-disable-line no-console
      return null;
    };

export const removeItem = isAvailable
  ? localStorage.removeItem.bind(localStorage)
  : () => {
      console.error(failMessage); // eslint-disable-line no-console
    };
