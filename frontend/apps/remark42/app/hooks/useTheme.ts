import { useSelector } from 'react-redux';

import { StoreState } from 'store';
import { Theme } from 'common/types';

export function useTheme() {
  const theme = useSelector<StoreState, Theme>(({ theme }) => theme);

  return theme;
}
