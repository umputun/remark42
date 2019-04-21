/** type that makes certain properties of type optional */
export type Derequire<T, K extends keyof T> = Pick<T, Exclude<keyof T, K>> & Partial<Pick<T, K>>;
