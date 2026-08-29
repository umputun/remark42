import { readdirSync, readFileSync } from 'fs';
import { join } from 'path';

import { render, screen } from '@testing-library/preact';

import { FormattedMessage, IntlProvider, createIntl, useIntl } from './intl';

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

describe('every shipped catalogue', () => {
  const localesDir = join(__dirname, '..', 'locales');
  const catalogues = readdirSync(localesDir)
    .filter((file) => file.endsWith('.json'))
    .map(
      (file) =>
        [file.replace('.json', ''), JSON.parse(readFileSync(join(localesDir, file), 'utf8'))] as [
          string,
          Record<string, string>,
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
