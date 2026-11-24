import { readFileSync, realpathSync } from 'node:fs';
import { join } from 'node:path';
import { pathToFileURL } from 'node:url';
import { gzipSync } from 'node:zlib';

export type Budget = { file: string; limit: number };
export type Result = Budget & { size: number | null; passed: boolean };

/** The most each file of the production build may weigh gzipped, in bytes. */
export const BUDGETS: Budget[] = [
  { file: 'embed.mjs', limit: 2_500 },
  { file: 'remark.mjs', limit: 60_000 },
  { file: 'remark.css', limit: 11_000 },
  { file: 'last-comments.mjs', limit: 2_500 },
  { file: 'last-comments.css', limit: 3_500 },
  { file: 'deleteme.mjs', limit: 9_500 },
];

/** Gzipped size at the highest level, what a server sends; null for a file that does not exist. */
export function gzippedSize(path: string): number | null {
  try {
    return gzipSync(readFileSync(path), { level: 9 }).length;
  } catch (error) {
    if ((error as NodeJS.ErrnoException).code === 'ENOENT') {
      return null;
    }
    throw error;
  }
}

/** Measures each budgeted file; a file that is missing fails, since nothing was measured. */
export function checkSizes(budgets: Budget[], sizeOf: (file: string) => number | null): Result[] {
  return budgets.map((budget) => {
    const size = sizeOf(budget.file);

    return { ...budget, size, passed: size !== null && size <= budget.limit };
  });
}

export function report(results: Result[]): string {
  const width = Math.max(...results.map(({ file }) => file.length));

  return results
    .map(({ file, size, limit, passed }) => {
      const measured = size === null ? 'missing' : `${(size / 1000).toFixed(2)} kB`;
      const share = size === null ? '' : ` (${Math.round((size / limit) * 100)}%)`;

      return `${passed ? 'ok  ' : 'FAIL'} ${file.padEnd(width)}  ${measured} of ${(limit / 1000).toFixed(1)} kB${share}`;
    })
    .join('\n');
}

function main(): void {
  const output = join(import.meta.dirname, '..', 'public');
  const results = checkSizes(BUDGETS, (file) => gzippedSize(join(output, file)));

  console.log(report(results));
  if (results.some(({ passed }) => !passed)) {
    console.error(
      'A file is over its budget or missing. Raise the budget in tasks/check-size.ts on purpose, or bring the file back under it.'
    );
    process.exit(1);
  }
}

// run as a script, not when a test imports it. import.meta.main would say so, but only from node 24.2
if (process.argv[1] && import.meta.url === pathToFileURL(realpathSync(process.argv[1])).href) {
  main();
}
