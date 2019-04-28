/** return shallow clone of object without given props */
export function exclude<T extends object, K extends keyof T>(o: T, ...exclude: K[]): Pick<T, Exclude<keyof T, K>> {
  const clone = { ...o };
  for (const item of exclude) {
    delete clone[item];
  }
  return clone;
}
