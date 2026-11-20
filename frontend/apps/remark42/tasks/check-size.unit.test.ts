import assert from 'node:assert/strict';
import { mkdtempSync, writeFileSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import { describe, it } from 'node:test';
import { gzipSync } from 'node:zlib';

import { checkSizes, gzippedSize, report } from './check-size.ts';

const budgets = [
  { file: 'a.mjs', limit: 100 },
  { file: 'b.css', limit: 50 },
];

describe('checkSizes', () => {
  it('passes a file at or under its budget', () => {
    const results = checkSizes(budgets, (file) => (file === 'a.mjs' ? 100 : 10));

    assert.deepEqual(
      results.map(({ passed }) => passed),
      [true, true]
    );
  });

  it('fails a file over its budget by one byte', () => {
    const results = checkSizes(budgets, (file) => (file === 'a.mjs' ? 101 : 10));

    assert.deepEqual(
      results.map(({ passed }) => passed),
      [false, true]
    );
  });

  // a renamed or unbuilt file would otherwise weigh nothing and pass every budget
  it('fails a file that is missing', () => {
    const results = checkSizes(budgets, (file) => (file === 'a.mjs' ? null : 10));

    assert.equal(results[0].passed, false);
    assert.match(report(results), /^FAIL a\.mjs\s+missing of 0\.1 kB$/m);
  });
});

describe('gzippedSize', () => {
  // a body on which the levels give different sizes, as they do on the real bundles, so a measurement
  // at any level but the highest is told apart
  it('measures the gzipped bytes at the highest level', () => {
    const dir = mkdtempSync(join(tmpdir(), 'check-size-'));
    let body = '';
    for (let i = 0; i < 3000; i++) {
      const colour = (((i * 2654435761) >>> 8) & 0xffffff).toString(16).padStart(6, '0');
      body += `.c${(i * 37) % 211}{margin:${i % 13}px ${(i * 7) % 11}px;color:#${colour}}`;
    }
    writeFileSync(join(dir, 'f.css'), body);

    const best = gzipSync(body, { level: 9 }).length;
    assert.notEqual(best, gzipSync(body).length, 'the body does not tell the levels apart');
    assert.equal(gzippedSize(join(dir, 'f.css')), best);
  });

  it('reports a missing file as null', () => {
    assert.equal(gzippedSize(join(tmpdir(), 'no-such-file-for-check-size.mjs')), null);
  });
});
