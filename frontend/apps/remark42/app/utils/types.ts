/* This interface defines an object that has keys and values of unknown type. */
export interface UnknownDict {
  [index: string]: unknown;
}

/* Checks if the provided argument is an UnknownDict. */
export const isUnknownDict = (candidate: unknown): candidate is UnknownDict =>
  typeof candidate === 'object' && candidate !== null;

/* Checks if a value is of type string */
export const isStr = (val: unknown): val is string => typeof val === 'string';

/* Checks if a value is of type string or undefined */
export const isStrOrUndef = (val: unknown): val is string | undefined => isStr(val) || isUndef(val);

/* Checks if a value is an array of strings */
export const isStrArr = (val: unknown): val is string[] =>
  isArr(val) && val.reduce<boolean>((memo, itm) => memo && isStr(itm), true);

/* Checks if a value is of type number */
export const isNum = (val: unknown): val is number => typeof val === 'number';

/* Checks if a value is of type number or undefined */
export const isNumOrUndef = (val: unknown): val is number => typeof val === 'number' || isUndef(val);

/* Checks if a value is an array of numbers */
export const isNumArr = (val: unknown): val is number[] =>
  isArr(val) && val.reduce<boolean>((memo, itm) => isNum(itm) && memo, true);

/* Checks if a value is an array of numbers or undefined */
export const isNumArrOrUndef = (val: unknown): val is number[] | undefined => isNumArr(val) || isUndef(val);

/* Checks if a value is of type boolean */
export const isBool = (val: unknown): val is boolean => typeof val === 'boolean';

/* Checks if a value is of type boolean or undefined */
export const isBoolOrUndef = (val: unknown): val is boolean | undefined => isBool(val) || isUndef(val);

/* Checks if a value is null */
export const isNull = (val: unknown): val is null => val === null;

/* Checks if a value is undefined */
export const isUndef = (val: unknown): val is undefined => typeof val === 'undefined';

/* Checks if a value is an array */
export const isArr = (val: unknown): val is unknown[] => Array.isArray(val);

/* Checks if a value is a function */
export const isFunc = (val: unknown): val is () => void => !!val && {}.toString.call(val) === '[object Function]';

/* Checks if a value is a Date object */
export const isDate = (val: unknown): val is Date => val instanceof Date;

/* Checks if a value is an Error object */
export const isErr = (val: unknown): val is Error => val instanceof Error;
