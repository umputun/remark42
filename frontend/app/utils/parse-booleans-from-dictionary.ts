export function parseBooleansFromDictionary(input: Record<string, unknown>, ...args: string[]) {
  const result: Record<string, boolean> = {};
  for (let key of args) {
    if (input[key] === undefined) {
      continue;
    }
    if (input[key] === 'true') {
      result[key] = true;
    }
    if (input[key] === 'false') {
      result[key] = false;
    }
  }
  return result;
}
