import { useSelector } from 'store/context';

import type { StoreState } from 'store';
import type { Theme } from 'common/types';

export function useTheme() {
  return useSelector<StoreState, Theme>(({ theme }) => theme);
}
