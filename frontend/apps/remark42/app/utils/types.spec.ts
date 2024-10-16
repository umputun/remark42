import {
  isArr,
  isBool,
  isBoolOrUndef,
  isDate,
  isErr,
  isFunc,
  isNull,
  isNum,
  isNumArr,
  isNumArrOrUndef,
  isNumOrUndef,
  isStr,
  isStrArr,
  isStrOrUndef,
  isUndef,
  isUnknownDict,
  UnknownDict,
} from './types';

describe('isUnknownDict', () => {
  test('returns true for an UnknownDict', () => {
    const obj: UnknownDict = {
      key1: 'value1',
      key2: 2,
      key3: true,
      key4: null,
    };
    expect(isUnknownDict(obj)).toBe(true);
  });

  test('returns false for a non-object', () => {
    expect(isUnknownDict('not an object')).toBe(false);
  });

  test('returns false for null', () => {
    expect(isUnknownDict(null)).toBe(false);
  });

  test('returns true for an empty object', () => {
    const obj = {};
    expect(isUnknownDict(obj)).toBe(true);
  });
});

describe('isStr', () => {
  test('should return true when passed a string', () => {
    expect(isStr('hello')).toBe(true);
  });

  test('should return false when passed a non-string value', () => {
    expect(isStr(123)).toBe(false);
    expect(isStr(undefined)).toBe(false);
    expect(isStr({})).toBe(false);
    expect(isStr(null)).toBe(false);
    expect(isStr([])).toBe(false);
  });
});

describe('isStrOrUndef', () => {
  test('should return true when passed a string', () => {
    expect(isStrOrUndef('hello')).toBe(true);
  });

  test('should return true when passed undefined', () => {
    expect(isStrOrUndef(undefined)).toBe(true);
  });

  test('should return false when passed a non-string value other than undefined', () => {
    expect(isStrOrUndef(123)).toBe(false);
    expect(isStrOrUndef({})).toBe(false);
    expect(isStrOrUndef(null)).toBe(false);
    expect(isStrOrUndef([])).toBe(false);
  });
});

describe('isStrArr', () => {
  test('should return true when passed an array of strings', () => {
    expect(isStrArr(['hello', 'world'])).toBe(true);
  });

  test('should return false when passed an array that contains a non-string value', () => {
    expect(isStrArr(['hello', 123])).toBe(false);
    expect(isStrArr(['hello', undefined])).toBe(false);
    expect(isStrArr(['hello', {}])).toBe(false);
  });

  test('should return false when passed a non-array value', () => {
    expect(isStrArr('hello')).toBe(false);
    expect(isStrArr(123)).toBe(false);
    expect(isStrArr(undefined)).toBe(false);
    expect(isStrArr({})).toBe(false);
    expect(isStrArr(null)).toBe(false);
  });
});

describe('isNum', () => {
  test('returns true for numbers', () => {
    expect(isNum(5)).toBe(true);
    expect(isNum(0)).toBe(true);
    expect(isNum(-10)).toBe(true);
    expect(isNum(3.14)).toBe(true);
  });

  test('returns false for non-numbers', () => {
    expect(isNum('5')).toBe(false);
    expect(isNum(undefined)).toBe(false);
    expect(isNum(null)).toBe(false);
    expect(isNum({ key: 'value' })).toBe(false);
  });
});

describe('isNumOrUndef', () => {
  test('returns true for numbers or undefined', () => {
    expect(isNumOrUndef(5)).toBe(true);
    expect(isNumOrUndef(undefined)).toBe(true);
    expect(isNumOrUndef(null)).toBe(false);
    expect(isNumOrUndef('5')).toBe(false);
  });

  test('returns false for other types', () => {
    expect(isNumOrUndef({ key: 'value' })).toBe(false);
    expect(isNumOrUndef([1, 2, 3])).toBe(false);
  });
});

describe('isNumArr', () => {
  test('returns true for arrays of numbers', () => {
    expect(isNumArr([1, 2, 3])).toBe(true);
    expect(isNumArr([0])).toBe(true);
    expect(isNumArr([])).toBe(true);
  });

  test('returns false for non-arrays or arrays of non-numbers', () => {
    expect(isNumArr(undefined)).toBe(false);
    expect(isNumArr(null)).toBe(false);
    expect(isNumArr('1,2,3')).toBe(false);
    expect(isNumArr([1, '2', 3])).toBe(false);
    expect(isNumArr([1, null, 3])).toBe(false);
  });
});

describe('isNumArrOrUndef', () => {
  test('returns true for arrays of numbers or undefined', () => {
    expect(isNumArrOrUndef(undefined)).toBe(true);
    expect(isNumArrOrUndef([1, 2, 3])).toBe(true);
  });

  test('returns false for other types', () => {
    expect(isNumArrOrUndef(null)).toBe(false);
    expect(isNumArrOrUndef('1,2,3')).toBe(false);
    expect(isNumArrOrUndef([1, '2', 3])).toBe(false);
  });
});

describe('isBool', () => {
  test('returns true for a boolean value', () => {
    expect(isBool(true)).toBe(true);
    expect(isBool(false)).toBe(true);
  });

  test('returns false for non-boolean values', () => {
    expect(isBool(1)).toBe(false);
    expect(isBool('true')).toBe(false);
    expect(isBool(null)).toBe(false);
    expect(isBool(undefined)).toBe(false);
    expect(isBool({})).toBe(false);
    expect(isBool([])).toBe(false);
    expect(isBool(() => {})).toBe(false);
    expect(isBool(new Date())).toBe(false);
  });
});

describe('isBoolOrUndef', () => {
  test('returns true for a boolean value', () => {
    expect(isBoolOrUndef(true)).toBe(true);
    expect(isBoolOrUndef(false)).toBe(true);
  });

  test('returns true for undefined value', () => {
    expect(isBoolOrUndef(undefined)).toBe(true);
  });

  test('returns false for non-boolean and non-undefined values', () => {
    expect(isBoolOrUndef(1)).toBe(false);
    expect(isBoolOrUndef('true')).toBe(false);
    expect(isBoolOrUndef(null)).toBe(false);
    expect(isBoolOrUndef({})).toBe(false);
    expect(isBoolOrUndef([])).toBe(false);
    expect(isBoolOrUndef(() => {})).toBe(false);
    expect(isBoolOrUndef(new Date())).toBe(false);
  });
});

describe('isNull', () => {
  test('returns true for null value', () => {
    expect(isNull(null)).toBe(true);
  });

  test('returns false for non-null values', () => {
    expect(isNull(1)).toBe(false);
    expect(isNull('null')).toBe(false);
    expect(isNull(undefined)).toBe(false);
    expect(isNull({})).toBe(false);
    expect(isNull([])).toBe(false);
    expect(isNull(() => {})).toBe(false);
    expect(isNull(new Date())).toBe(false);
  });
});

describe('isUndef', () => {
  test('returns true for undefined value', () => {
    expect(isUndef(undefined)).toBe(true);
  });

  test('returns false for non-undefined values', () => {
    expect(isUndef(1)).toBe(false);
    expect(isUndef('undefined')).toBe(false);
    expect(isUndef(null)).toBe(false);
    expect(isUndef({})).toBe(false);
    expect(isUndef([])).toBe(false);
    expect(isUndef(() => {})).toBe(false);
    expect(isUndef(new Date())).toBe(false);
  });
});

describe('isArr', () => {
  test('returns true for an array', () => {
    expect(isArr([])).toBe(true);
    expect(isArr([1, 2, 3])).toBe(true);
    expect(isArr(['a', 'b', 'c'])).toBe(true);
  });

  test('returns false for non-array values', () => {
    expect(isArr(1)).toBe(false);
    expect(isArr('array')).toBe(false);
    expect(isArr(null)).toBe(false);
    expect(isArr(undefined)).toBe(false);
    expect(isArr({})).toBe(false);
    expect(isArr(() => {})).toBe(false);
    expect(isArr(new Date())).toBe(false);
  });
});

describe('isFunc', () => {
  test('returns true for a function', () => {
    expect(isFunc(() => {})).toBe(true);
    // eslint-disable-next-line prefer-arrow-callback
    expect(isFunc(function () {})).toBe(true);
  });

  test('returns false for non-function values', () => {
    expect(isFunc(1)).toBe(false);
    expect(isFunc('function')).toBe(false);
    expect(isFunc(null)).toBe(false);
    expect(isFunc(undefined)).toBe(false);
    expect(isFunc({})).toBe(false);
    expect(isFunc([])).toBe(false);
    expect(isFunc(new Date())).toBe(false);
    expect(isFunc(new Error())).toBe(false);
  });
});

describe('isDate', () => {
  test('returns true for a Date object', () => {
    expect(isDate(new Date())).toBe(true);
  });

  test('returns false for non-Date values', () => {
    expect(isDate(1)).toBe(false);
    expect(isDate('date')).toBe(false);
    expect(isDate(null)).toBe(false);
    expect(isDate(undefined)).toBe(false);
    expect(isDate({})).toBe(false);
    expect(isDate([])).toBe(false);
    expect(isDate(() => {})).toBe(false);
    expect(isDate(new Error())).toBe(false);
  });
});

describe('isErr', () => {
  test('returns true for an Error object', () => {
    expect(isErr(new Error())).toBe(true);
  });

  test('returns false for non-Error values', () => {
    expect(isErr(1)).toBe(false);
    expect(isErr('error')).toBe(false);
    expect(isErr(null)).toBe(false);
    expect(isErr(undefined)).toBe(false);
    expect(isErr({})).toBe(false);
    expect(isErr([])).toBe(false);
    expect(isErr(() => {})).toBe(false);
    expect(isErr(new Date())).toBe(false);
  });
});
