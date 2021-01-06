export function bench<T>(fn: () => T, label = 'bench'): T {
  const d = performance.now();
  const r = fn();
  const dd = performance.now();
  // eslint-disable-next-line no-console
  console.info(label, dd - d);
  return r;
}
