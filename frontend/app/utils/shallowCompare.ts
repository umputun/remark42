export function shallowCompare<T extends object>(a: T, b: T): boolean {
  const entriesA = Object.entries(a);
  const keysB = Object.keys(b);
  if (entriesA.length !== keysB.length) return false;
  for (const [key, value] of entriesA) {
    // @ts-ignore
    if (value !== b[key]) return false;
  }
  return true;
}
