import type { Sorting } from 'common/types';
import { LS_SORT_KEY, DEFAULT_SORT } from 'common/constants';
import { getItem } from 'common/local-storage';

export function getInitialSort(): Sorting {
  const sort = getItem(LS_SORT_KEY) as Sorting;

  if (!sort) {
    return DEFAULT_SORT;
  }

  return sort;
}
