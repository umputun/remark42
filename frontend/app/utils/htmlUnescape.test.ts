import { htmlUnescape } from './htmlUnescape';

describe('htmlUnescape', () => {
  const data: ([string, string])[] = [
    ['blah &amp; &amp; 123', 'blah & & 123'],
    ['blah & & 123 &mdash; &mdash;', 'blah & & 123 — —'],
    ['name &lt;&gt; & \' ` "', 'name <> & \' ` "'],
  ];

  for (const example of data) {
    it(`unescapes ${example[0]} to ${example[1]}`, () => {
      expect(htmlUnescape(example[0])).toStrictEqual(example[1]);
    });
  }
});
