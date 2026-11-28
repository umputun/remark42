import { h } from 'preact';
import { render } from '@testing-library/preact';

import { createIntl, IntlProvider } from 'common/intl';
import en from 'locales/en.json';
import { avatarElement } from 'components/avatar/avatar-element.js';
import { Button, type ButtonProps } from 'components/button/button';
import { buttonElement } from 'components/button/button-element.js';
import { Input } from 'components/input';
import { inputElement } from 'components/input/input-element.js';
import { TelegramLink } from 'components/telegram/telegram-link';
import { telegramLink } from 'components/telegram/telegram-link-element.js';
import { TextareaAutosize } from 'components/textarea-autosize';
import { textareaElement } from 'components/textarea-element.js';

import { Button as AuthButton } from './components/button';
import { buttonElement as authButtonElement, suffixElement } from './components/button-element.js';
import { ArrowIcon } from 'components/icons/arrow';
import { arrowIcon } from 'components/icons/arrow-icon.js';
import { BASE_URL } from 'common/constants.config';

import { oauthList } from './components/oauth-list.js';

const screenWidth = window.screen.width;

afterEach(() => {
  Object.defineProperty(window.screen, 'width', { value: screenWidth, configurable: true });
});

function drawn(node: preact.ComponentChild, locale = 'en') {
  return render(
    <IntlProvider locale={locale} messages={en}>
      {node}
    </IntlProvider>
  ).container.innerHTML;
}

function html(...nodes: Node[]) {
  const box = document.createElement('div');
  box.append(...nodes);
  return box.innerHTML;
}

describe('the plain leaves draw the markup of the preact ones', () => {
  it.each([
    ['a text field', {}, {}],
    [
      'an invalid email field with attributes',
      { className: 'f', type: 'email', invalid: true },
      { name: 'email', required: true, placeholder: 'Email', disabled: true, dir: 'auto' },
    ],
  ])('%s', (_, look, attributes) => {
    const was = drawn(<Input {...look} {...(attributes as Record<string, unknown>)} />);

    expect(inputElement(look, attributes).outerHTML).toBe(was);
  });

  it('a field holds its value as the property, as the preact Input does', () => {
    const input = inputElement({}, { value: 'typed' });

    expect(input.value).toBe('typed');
    expect(input.hasAttribute('value')).toBe(false);
  });

  it('the avatar changes the same image', () => {
    const avatar = avatarElement({ size: 20 });
    const { img } = avatar;

    avatar.update({ url: 'https://example.com/a.png', size: 20, identicon: true });

    expect(avatar.img).toBe(img);
    expect(img.outerHTML).toBe(
      avatarElement({ url: 'https://example.com/a.png', size: 20, identicon: true }).img.outerHTML
    );
  });

  it.each<[string, ButtonProps]>([
    ['a plain button', {}],
    [
      'a primary large submit button with a class',
      { kind: 'primary', size: 'large', className: 'auth-submit button', type: 'submit' },
    ],
    ['a dark link button with mixes', { kind: 'link', theme: 'dark', mix: ['a', 'b'] }],
  ])('the button: %s', (_, look) => {
    const was = drawn(<Button {...look}>Go</Button>);
    const button = buttonElement(look as Parameters<typeof buttonElement>[0]);
    button.textContent = 'Go';

    expect(button.outerHTML).toBe(was);
  });

  it('the sign-in button, selected, with its suffix', () => {
    const was = drawn(
      <AuthButton className="auth-button" selected suffix={<ArrowIcon />}>
        Sign in
      </AuthButton>
    );
    const button = authButtonElement({ className: 'auth-button', selected: true });
    button.append('Sign in', suffixElement(arrowIcon()));

    expect(button.outerHTML).toBe(was);
  });

  it('the textarea, measured once on the page', () => {
    const was = render(<TextareaAutosize name="token" className="t" placeholder="Paste" />).container.innerHTML;
    const field = textareaElement({ name: 'token', class: 't', placeholder: 'Paste' });
    document.body.append(field.textarea);
    field.resize();

    expect(field.textarea.outerHTML).toBe(was);
    field.textarea.remove();
  });

  it('the textarea carries its id as the test id, as the preact one does', () => {
    const was = render(<TextareaAutosize id="text" />).container.innerHTML;

    expect(textareaElement({ id: 'text' }).textarea.outerHTML).toBe(was.replace(' style="height: 0px;"', ''));
  });

  it('a controlled textarea leaves the height to setValue, and an uncontrolled one measures on input', () => {
    const onInput = jest.fn();
    const controlled = textareaElement({}, { onInput, controlled: true });
    const free = textareaElement({}, { onInput });

    controlled.textarea.dispatchEvent(new Event('input'));
    free.textarea.dispatchEvent(new Event('input'));

    expect(controlled.textarea.style.height).toBe('');
    expect(free.textarea.style.height).toBe('0px');
    expect(onInput).toHaveBeenCalledTimes(2);

    controlled.setValue('text');
    expect(controlled.textarea.value).toBe('text');
    expect(controlled.textarea.style.height).toBe('0px');
  });

  it.each([
    ['on a narrow screen', 360, 'en', null],
    ['on a wide screen, with the QR code', 1024, 'en', null],
    ['with an error', 360, 'en', 'not confirmed'],
    ['in Japanese, with no spaces between its parts', 1024, 'ja', null],
  ])('the Telegram step %s', (_, width, locale, errorMessage) => {
    Object.defineProperty(window.screen, 'width', { value: width, configurable: true });
    const props = { bot: 'remark_bot', token: 'tok', onSubmit: jest.fn(), errorMessage };
    const was = drawn(<TelegramLink {...props} />, locale);

    expect(html(...telegramLink(props, createIntl(locale, en)).nodes())).toBe(was);
  });

  it('the Telegram step loads no QR code on a narrow screen', () => {
    Object.defineProperty(window.screen, 'width', { value: 360, configurable: true });
    const sources = jest.spyOn(HTMLImageElement.prototype, 'src', 'set');

    telegramLink({ bot: 'b', token: 't', onSubmit: jest.fn() }, createIntl('en', en)).update({
      bot: 'b',
      token: 'u',
      onSubmit: jest.fn(),
    });

    expect(sources.mock.calls.filter(([src]) => String(src).includes('/qr/telegram'))).toEqual([]);
    sources.mockRestore();
  });

  it('the OAuth list keeps its class names, loads its icons from the server and links to the login', () => {
    const { root } = oauthList(['google'], { intl: createIntl('en', en), theme: 'light', onOauthClick: jest.fn() });

    expect(root.getAttribute('class')).toContain('oauth');
    expect(root.querySelector('li')?.getAttribute('class')).toContain('oauth-item');
    expect(root.querySelector('a')?.getAttribute('class')).toContain('oauth-button');
    expect(root.querySelector('img')?.getAttribute('class')).toContain('oauth-icon');
    expect(root.querySelector('img')?.getAttribute('src')).toBe('http://test.com/web/google.svg');
    expect(root.querySelector('a')?.getAttribute('href')).toBe(
      `${BASE_URL}/auth/google/login?from=http%3A%2F%2Flocalhost%2F%3FselfClose&site=remark`
    );
  });

  // a source set again, even to the same value, makes the browser load the image afresh, so an icon
  // or avatar redrawn while it loads is not shown
  it.each([
    [
      'the OAuth list',
      () => {
        const deps = { intl: createIntl('en', en), theme: 'light' as const, onOauthClick: jest.fn() };
        const list = oauthList(['google', 'dev'], deps);
        return { root: list.root, again: () => list.update({ ...deps }) };
      },
    ],
    [
      'the avatar',
      () => {
        const avatar = avatarElement({ url: 'https://example.com/a.png', size: 20 });
        return { root: avatar.img, again: () => avatar.update({ url: 'https://example.com/a.png', size: 20 }) };
      },
    ],
  ])('%s keeps its image sources across an update that does not change them', async (_, make) => {
    const { root, again } = make();
    const writes: MutationRecord[] = [];
    const watch = new MutationObserver((records) => writes.push(...records));
    watch.observe(root, { attributes: true, attributeFilter: ['src'], subtree: true });

    again();
    writes.push(...watch.takeRecords());
    watch.disconnect();

    expect(writes).toEqual([]);
  });

  // the step sits inside the comment form, so a submit button there would also post the draft
  it('the Telegram step does not submit the form it is placed in when Check is pressed', () => {
    const onSubmit = jest.fn();
    const onFormSubmit = jest.fn((evt: Event) => evt.preventDefault());
    const form = document.createElement('form');
    form.addEventListener('submit', onFormSubmit);
    form.append(...telegramLink({ bot: 'b', token: 't', onSubmit }, createIntl('en', en)).nodes());
    document.body.append(form);

    (form.querySelector('.auth-submit') as HTMLButtonElement).click();

    expect(onSubmit).toHaveBeenCalledTimes(1);
    expect(onFormSubmit).not.toHaveBeenCalled();
    form.remove();
  });

  it('the Telegram step sends a press of Check to the latest handler, keeping its nodes', () => {
    const first = jest.fn();
    const latest = jest.fn();
    const step = telegramLink({ bot: 'b', token: 't', onSubmit: first }, createIntl('en', en));
    const before = step.nodes();

    step.update({ bot: 'b', token: 't', onSubmit: latest });
    (step.nodes()[step.nodes().length - 1] as HTMLButtonElement).click();

    expect(step.nodes()).toHaveLength(before.length);
    step.nodes().forEach((node, i) => expect(node).toBe(before[i]));
    expect(latest).toHaveBeenCalledTimes(1);
    expect(first).not.toHaveBeenCalled();
  });
});
