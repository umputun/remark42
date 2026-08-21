import { readdirSync, readFileSync } from 'fs';
import { join } from 'path';

import { render, screen } from '@testing-library/preact';

import { FormattedMessage, IntlProvider, createIntl, defineMessages, useIntl } from './intl';

const descriptor = { id: 'greeting', defaultMessage: 'Hello {name}' };

describe('createIntl', () => {
  it('prefers the catalogue over the default message', () => {
    const intl = createIntl('ru', { greeting: 'Привет {name}' });

    expect(intl.formatMessage(descriptor, { name: 'Sam' })).toBe('Привет Sam');
  });

  it('falls back to the default message when the key is absent', () => {
    const intl = createIntl('en', {});

    expect(intl.formatMessage(descriptor, { name: 'Sam' })).toBe('Hello Sam');
  });

  it('leaves a placeholder without a value untouched', () => {
    const intl = createIntl('en', {});

    expect(intl.formatMessage(descriptor)).toBe('Hello {name}');
  });

  it('accepts whitespace inside the braces, as ICU does', () => {
    const intl = createIntl('en', { greeting: 'Hello { name }' });

    expect(intl.formatMessage(descriptor, { name: 'Sam' })).toBe('Hello Sam');
  });

  it('treats an empty catalogue entry as missing', () => {
    const intl = createIntl('ru', { greeting: '' });

    expect(intl.formatMessage(descriptor, { name: 'Sam' })).toBe('Hello Sam');
  });

  it('degrades to the raw value when a date cannot be formatted', () => {
    const intl = createIntl('en', {});
    const invalid = new Date('nonsense');

    expect(intl.formatDate(invalid)).toBe(String(invalid));
    expect(intl.formatTime(invalid)).toBe(String(invalid));
  });

  it('formats time with hour and minute by default', () => {
    const intl = createIntl('en', {});
    const at = new Date(2026, 0, 2, 15, 4);

    expect(intl.formatTime(at)).toBe(new Intl.DateTimeFormat('en', { hour: 'numeric', minute: 'numeric' }).format(at));
    expect(intl.formatDate(at)).toBe(new Intl.DateTimeFormat('en').format(at));
  });

  // formatters are cached, so the cache key has to separate locales and option sets
  it('does not confuse cached formatters across locales and options', () => {
    const at = new Date(2026, 0, 2, 15, 4);

    expect(createIntl('ru', {}).formatDate(at)).not.toBe(createIntl('en', {}).formatDate(at));
    expect(createIntl('en', {}).formatDate(at, { dateStyle: 'full' })).not.toBe(createIntl('en', {}).formatDate(at));
    expect(createIntl('en', {}).formatDate(at)).toBe(createIntl('en', {}).formatDate(at));
  });

  it('honours explicit time options instead of adding defaults', () => {
    const intl = createIntl('en', {});
    const at = new Date(2026, 0, 2, 15, 4);

    expect(intl.formatTime(at, { hour: '2-digit' })).toBe(
      new Intl.DateTimeFormat('en', { hour: '2-digit' }).format(at)
    );
  });
});

describe('defineMessages', () => {
  it('returns its argument unchanged', () => {
    const messages = defineMessages({ greeting: descriptor });

    expect(messages.greeting).toBe(descriptor);
  });
});

describe('<FormattedMessage/>', () => {
  const link = { id: 'powered-by', defaultMessage: 'Powered by <a>Remark42</a>' };

  function renderWith(locale: string, messages: Record<string, string>) {
    return render(
      <IntlProvider locale={locale} messages={messages}>
        <FormattedMessage {...link} values={{ a: (chunk: string) => <a href="/x">{chunk}</a> }} />
      </IntlProvider>
    );
  }

  it('wraps the tagged chunk with the handler', () => {
    const { container } = renderWith('en', {});

    expect(container.textContent).toBe('Powered by Remark42');
    expect(screen.getByRole('link').textContent).toBe('Remark42');
  });

  it('uses the translated text around the tag', () => {
    const { container } = renderWith('ru', { 'powered-by': 'Работает на базе <a>Remark42</a>' });

    expect(container.textContent).toBe('Работает на базе Remark42');
  });

  // both shapes fall back to the source rather than reaching the page
  it('falls back to the default message when a tag carries attributes', () => {
    const { container } = renderWith('mk', {
      'powered-by': "Овозможено од <a href='diosfera.codeberg.page'>Диосфера</a>",
    });

    expect(container.textContent).toBe('Powered by Remark42');
  });

  // ro drops the link from both rich-text strings, which is valid
  it('renders a translation that does not use the tag at all', () => {
    const { container } = renderWith('ro', { 'powered-by': 'Cu sprijinul Remark42' });

    expect(container.textContent).toBe('Cu sprijinul Remark42');
    expect(container.querySelector('a')).toBeNull();
  });

  it('handles the same tag appearing more than once', () => {
    const { container } = render(
      <IntlProvider locale="en" messages={{ two: '<a>one</a> and <a>two</a>' }}>
        <FormattedMessage id="two" defaultMessage="x" values={{ a: (chunk: string) => <a href="/x">{chunk}</a> }} />
      </IntlProvider>
    );

    expect(container.textContent).toBe('one and two');
    expect(container.querySelectorAll('a')).toHaveLength(2);
  });

  // one well-formed pair must not vouch for a stray reference elsewhere
  it('falls back when a stray tag sits beside a well-formed pair', () => {
    const { container } = renderWith('xx', { 'powered-by': 'A <a>B</a> C </a>' });

    expect(container.textContent).toBe('Powered by Remark42');
  });

  // a translator inventing <b> gets English rather than visible markup
  it('falls back when the translation introduces a tag nothing handles', () => {
    const { container } = renderWith('xx', { 'powered-by': 'Made with <b>love</b>' });

    expect(container.textContent).toBe('Powered by Remark42');
  });

  it('falls back to the default message when a tag is unclosed', () => {
    const { container } = renderWith('th', { 'powered-by': 'ระบบแสดงความคิดเห็นโดย <a>Remark42</a' });

    expect(container.textContent).toBe('Powered by Remark42');
  });
});

describe('useIntl', () => {
  it('throws outside of a provider', () => {
    function Orphan() {
      useIntl();

      return null;
    }

    expect(() => render(<Orphan />)).toThrow('intl accessed outside of an IntlProvider');
  });
});

describe('formatting details that no shipped message exercises yet', () => {
  it('interpolates inside a rich-text chunk', () => {
    const intl = createIntl('en', { hi: 'Hello <b>{name}</b>' });

    expect(intl.formatMessage({ id: 'hi', defaultMessage: 'x' }, { name: 'Sam', b: (c: string) => c })).toEqual([
      'Hello ',
      'Sam',
    ]);
  });

  it('falls back to the id when there is no translation and no default', () => {
    expect(createIntl('en', {}).formatMessage({ id: 'orphan.key' })).toBe('orphan.key');
  });

  // a pair nothing handles is a format error
  it('falls back when the translation adds a pair nothing handles', () => {
    const intl = createIntl('en', { k: 'See <abbr>HTML</abbr> and <a>link</a>' });

    expect(intl.formatMessage({ id: 'k', defaultMessage: 'Go <a>home</a>' }, { a: (c: string) => c })).toEqual([
      'Go ',
      'home',
    ]);
  });

  // any tag that cannot resolve makes the whole string fall back, rather than
  // putting raw markup on the page
  it.each([
    ['an unclosed tag of another name', 'Go <a>home</a> <abbr'],
    ['an unclosed tag of our own name', 'Go <a>home'],
    ['a tag carrying an attribute', "Go <a href='x'>home</a>"],
    ['spaces inside the brackets', 'Go < a >home</ a >'],
    ['a tag nested inside a handled one', 'Go <a>ho<b>m</b>e</a>'],
  ])('falls back on %s', (_name, message) => {
    const intl = createIntl('en', { k: message });

    expect(intl.formatMessage({ id: 'k', defaultMessage: 'Go <a>home</a>' }, { a: (c: string) => c })).toEqual([
      'Go ',
      'home',
    ]);
  });

  it.each([
    ['an unclosed brace', 'Hello {name and <a>R</a>'],
    ['doubled braces', 'Hello {{name}} <a>R</a>'],
    ['a stray closing brace', 'Hello name} <a>R</a>'],
  ])('falls back on %s', (_name, message) => {
    const intl = createIntl('en', { k: message });

    expect(intl.formatMessage({ id: 'k', defaultMessage: 'Go <a>home</a>' }, { a: (c: string) => c })).toEqual([
      'Go ',
      'home',
    ]);
  });

  // formatjs leaves a self-closing tag as text, so it must not trigger the fallback
  it('renders a self-closing tag as text', () => {
    const intl = createIntl('en', { k: 'Go <a>home</a><br/>' });

    expect(intl.formatMessage({ id: 'k', defaultMessage: 'default' }, { a: (c: string) => c })).toEqual([
      'Go ',
      'home',
      '<br/>',
    ]);
  });

  it('leaves a bare less-than sign alone', () => {
    const intl = createIntl('en', { k: 'Under < 10 via <a>here</a>' });

    expect(intl.formatMessage({ id: 'k', defaultMessage: 'default' }, { a: (c: string) => c })).toEqual([
      'Under < 10 via ',
      'here',
    ]);
  });

  it('treats a non-ascii suffix as a reference to the tag, and falls back', () => {
    const intl = createIntl('en', { k: '<aб>X</aб>' });

    expect(intl.formatMessage({ id: 'k', defaultMessage: 'Go <a>home</a>' }, { a: (c: string) => c })).toEqual([
      'Go ',
      'home',
    ]);
  });
});

describe('every shipped catalogue', () => {
  const localesDir = join(__dirname, '..', 'locales');
  const catalogues = readdirSync(localesDir)
    .filter((file) => file.endsWith('.json'))
    .map(
      (file) =>
        [file.replace('.json', ''), JSON.parse(readFileSync(join(localesDir, file), 'utf8'))] as [
          string,
          Record<string, string>
        ]
    );

  const richText = [
    { id: 'root.powered-by', defaultMessage: 'Powered by <a>Remark42</a>' },
    { id: 'commentForm.notice-about-styling', defaultMessage: 'Styling with <a>Markdown</a> is supported' },
  ];

  // only the handled tag is stripped; a self-closing tag stays, as the binding keeps it
  const withoutTags = (message: string) => message.replace(/<\/?a>/g, '');

  // reads the real files, so a malformed entry added to any locale later fails here: the
  // binding falls back to English on one, and the assertion below expects the locale's own
  // words. structural validation of every other value is checkTranslation.js's job
  it.each(catalogues)('%s renders its own words in its rich-text messages', (locale, messages) => {
    const intl = createIntl(locale, messages);

    richText.forEach((descriptor) => {
      const rendered = ([] as unknown[])
        .concat(intl.formatMessage(descriptor, { a: (chunk: string) => chunk }) as never)
        .join('');
      // an empty entry counts as missing, matching the binding
      const translated = messages[descriptor.id] || undefined;

      expect(rendered).toBe(withoutTags(translated ?? descriptor.defaultMessage));
    });
  });

  it('formats dates and times in the widget locale rather than always in English', () => {
    const at = new Date(2026, 0, 2, 15, 4);

    expect(createIntl('ru', {}).formatDate(at)).not.toBe(createIntl('en', {}).formatDate(at));
    expect(createIntl('ru', {}).formatTime(at)).not.toBe(createIntl('en', {}).formatTime(at));
  });
});

describe('branches with no shipped input yet', () => {
  it('renders the broken message when there is nothing to fall back to', () => {
    const intl = createIntl('xx', { k: 'Broken <a>link</a' });

    expect(intl.formatMessage({ id: 'k' }, { a: (c: string) => c })).toEqual(['Broken <a>link</a']);
  });

  it('does not interpolate a placeholder whose value is a handler', () => {
    const intl = createIntl('en', { k: '{a} and <a>link</a>' });

    expect(intl.formatMessage({ id: 'k', defaultMessage: 'x' }, { a: (c: string) => c })).toEqual(['{a} and ', 'link']);
  });

  it('escapes a tag name that carries a regular-expression metacharacter', () => {
    const intl = createIntl('en', { k: 'a.b <a.b>x</a.b> c' });

    expect(() => intl.formatMessage({ id: 'k', defaultMessage: 'y' }, { 'a.b': (c: string) => c })).not.toThrow();
  });

  it.each([['minute'], ['second'], ['dateStyle'], ['timeStyle']])(
    'does not add hour and minute when %s is given',
    (option) => {
      const intl = createIntl('en', {});
      const at = new Date(2026, 0, 2, 15, 4);
      const options = { [option]: option === 'dateStyle' || option === 'timeStyle' ? 'short' : 'numeric' };

      expect(intl.formatTime(at, options as Intl.DateTimeFormatOptions)).toBe(
        new Intl.DateTimeFormat('en', options as Intl.DateTimeFormatOptions).format(at)
      );
    }
  );
});
