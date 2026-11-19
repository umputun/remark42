import type { Sorting } from 'common/types';
import { LS_SORT_KEY } from 'common/constants';
import { defaultSort } from 'common/settings';
import { getItem } from 'common/local-storage';

export function getInitialSort(): Sorting {
  const sort = getItem(LS_SORT_KEY) as Sorting;

  if (!sort) {
    return defaultSort;
  }

  return sort;
}
