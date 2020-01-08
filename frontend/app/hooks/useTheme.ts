import { useSelector } from 'react-redux';

import { StoreState } from '@app/store';
import { Theme } from '@app/common/types';

export default function useTheme() {
  const theme = useSelector<StoreState, Theme>(({ theme }) => theme);

  return theme;
}
