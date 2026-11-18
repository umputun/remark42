import type { Sorting } from './types';

/** every order the sort picker offers, also the values `default_sort` accepts */
export const SORTINGS: ReadonlyArray<Sorting> = [
  '-score',
  '+score',
  '-time',
  '+time',
  '-active',
  '+active',
  '-controversy',
  '+controversy',
];

/** returns the value if it is one of the sort orders, undefined for anything else */
export function parseSorting(value: unknown): Sorting | undefined {
  return SORTINGS.find((s) => s === value);
}
