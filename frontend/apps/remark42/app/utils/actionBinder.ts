import { StoreAction } from 'store';
import { Action } from 'redux';

/** Helper type which is used to convert actionCreator to redux `connect` bound prop */
export type BoundActionCreator<A> = A extends (...args: infer U) => StoreAction<infer R>
  ? (...args: U) => R
  : A extends (...args: infer U) => Action<infer R>
  ? (...args: U) => Action<R>
  : A extends (...args: infer U) => Promise<infer R>
  ? (...args: U) => Promise<R>
  : never;

/** Helper type which is used to convert actionCreators map to redux `connect` bound props */
export type BoundActionCreators<T extends object> = { [K in keyof T]: BoundActionCreator<T[K]> };

/**
 * no-op function that is used for type conversion for action creators connected
 * through mapDispatchToProps
 */
export function bindActions<A extends object>(obj: A): BoundActionCreators<A> {
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  return obj as any;
}
