export default function shallowCompare<T extends Record<string, unknown>>(a: T, b: T): boolean {
  const entriesA = Object.entries(a);
  const keysB = Object.keys(b);
  if (entriesA.length !== keysB.length) return false;
  for (const [key, value] of entriesA) {
    if (value !== b[key]) return false;
  }
  return true;
}
