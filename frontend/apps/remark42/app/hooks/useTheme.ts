import { useSelector } from 'react-redux';

import { StoreState } from 'store';
import { Theme } from 'common/types';

export function useTheme() {
  return useSelector<StoreState, Theme>(({ theme }) => theme);
}
