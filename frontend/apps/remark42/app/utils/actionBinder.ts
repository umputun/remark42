import type { StoreAction } from 'store';
import type { Action } from 'store/context';

/** Helper type which is used to convert an action creator to the bound prop of a connected component */
export type BoundActionCreator<A> = A extends (...args: infer U) => StoreAction<infer R>
  ? (...args: U) => R
  : A extends (...args: infer U) => Action
    ? (...args: U) => ReturnType<A>
    : A extends (...args: infer U) => Promise<infer R>
      ? (...args: U) => Promise<R>
      : never;

/** Helper type which is used to convert a map of action creators to the bound props of a connected component */
export type BoundActionCreators<T extends object> = { [K in keyof T]: BoundActionCreator<T[K]> };

/**
 * no-op function that is used for type conversion for action creators connected
 * through mapDispatchToProps
 */
export function bindActions<A extends object>(obj: A): BoundActionCreators<A> {
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  return obj as any;
}
