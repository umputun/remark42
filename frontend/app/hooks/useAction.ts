/* eslint-disable @typescript-eslint/no-explicit-any */
import { useMemo } from 'preact/compat';
import { useDispatch } from 'react-redux';
import { BoundActionCreator, BoundActionCreators } from 'utils/actionBinder';

/** binds actions to dispatch */
export const useActions = <Actions extends { [key: string]: Function }>(
  actions: Actions
): BoundActionCreators<Actions> => {
  const dispatch = useDispatch();

  return useMemo(
    () =>
      Object.entries(actions).reduce<BoundActionCreator<Actions>>((result, [key, fn]) => {
        (result as any)[key] = (...args: any[]) => dispatch(fn(...args));
        return result;
      }, {} as any),
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [dispatch, ...Object.values(actions)]
  ) as any;
};
