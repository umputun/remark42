import { htmlUnescape, htmlPartialUnescape } from './htmlUnescape';

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

describe('htmlPartialUnescape', () => {
  const data: ([string, string])[] = [
    ['blah &amp; &amp; &#34; &#39; 123', 'blah & & " \' 123'],
    ['&amp;&amp;&amp;&amp;', '&&&&'],
    ['blah & & 123 &mdash; &mdash;', 'blah & & 123 &mdash; &mdash;'],
    ['name &lt;&gt; & \' ` "', 'name &lt;&gt; & \' ` "'],
  ];

  for (const example of data) {
    it(`unescapes ${example[0]} to ${example[1]}`, () => {
      expect(htmlPartialUnescape(example[0])).toStrictEqual(example[1]);
    });
  }
});
