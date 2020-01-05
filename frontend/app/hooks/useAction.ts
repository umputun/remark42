import { useCallback, useMemo } from 'preact/compat';
import { useDispatch } from 'react-redux';
import { BoundActionCreator, BoundActionCreators } from '@app/utils/actionBinder';

/* eslint-disable @typescript-eslint/no-explicit-any */

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
    [dispatch, ...Object.values(actions)]
  ) as any;
};

export const useAction = <Action extends Function>(action: Action): BoundActionCreator<Action> => {
  const dispatch = useDispatch();

  return useCallback(((...args: any[]) => dispatch(action(...args))) as any, [dispatch, action]);
};

/* eslint-enable @typescript-eslint/no-explicit-any */
