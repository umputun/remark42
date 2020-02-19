/** @jsx createElement */
import { createElement } from 'preact';
import { mount } from 'enzyme';
import { EmailLoginForm, Props, State } from './auth-panel__email-login-form';
import { User } from '@app/common/types';
import { sleep } from '@app/utils/sleep';
import { validToken } from '@app/testUtils/mocks/jwt';
import { createIntl } from 'react-intl';
import enMessages from '../../../locales/en.json';

jest.mock('@app/utils/jwt', () => ({
  isJwtExpired: jest
    .fn()
    .mockImplementationOnce(() => false)
    .mockImplementationOnce(() => true),
}));

const intl = createIntl({
  locale: `en`,
  messages: enMessages,
});

describe('EmailLoginForm', () => {
  const testUser = ({} as any) as User;
  const onSuccess = jest.fn(async () => {});
  const onSignIn = jest.fn(async () => testUser);

  it('works', async () => {
    const sendEmailVerification = jest.fn(async () => {});

    const el = mount<Props, State>(
      <EmailLoginForm
        sendEmailVerification={sendEmailVerification}
        onSignIn={onSignIn}
        onSuccess={onSuccess}
        theme="light"
        intl={intl}
      />
    );

    await new Promise(resolve =>
      el.setState({ usernameValue: 'someone', addressValue: 'someone@example.com' } as State, resolve)
    );

    el.find('form').simulate('submit');
    await sleep(100);
    expect(sendEmailVerification).toBeCalledWith('someone', 'someone@example.com');
    expect(el.state().verificationSent).toBe(true);

    await new Promise(resolve => el.setState({ tokenValue: 'abcd' } as State, resolve));

    el.find('form').simulate('submit');
    await sleep(100);
    expect(onSignIn).toBeCalledWith('abcd');
    expect(onSuccess).toBeCalledWith(testUser);
  });

  it('should send form by pasting token', async () => {
    const sendEmailVerification = jest.fn(async () => {});
    const onSignIn = jest.fn(async () => testUser);

    const wrapper = mount<Props, State>(
      <EmailLoginForm
        sendEmailVerification={sendEmailVerification}
        onSignIn={onSignIn}
        onSuccess={onSuccess}
        theme="light"
        intl={intl}
      />
    );
    await new Promise(resolve =>
      wrapper.setState({ usernameValue: 'someone', addressValue: 'someone@example.com' } as State, resolve)
    );
    wrapper.find('form').simulate('submit');
    await sleep(100);
    wrapper.update();

    wrapper.find('textarea').getDOMNode<HTMLTextAreaElement>().value = validToken;
    wrapper.find('textarea').simulate('input');

    expect(onSignIn).toBeCalledWith(validToken);
  });

  it('should show error "Token is expired" on paste', async () => {
    const sendEmailVerification = jest.fn(async () => {});
    const onSignIn = jest.fn(async () => testUser);

    const wrapper = mount<Props, State>(
      <EmailLoginForm
        sendEmailVerification={sendEmailVerification}
        onSignIn={onSignIn}
        onSuccess={onSuccess}
        theme="light"
        intl={intl}
      />
    );
    await new Promise(resolve =>
      wrapper.setState({ usernameValue: 'someone', addressValue: 'someone@example.com' } as State, resolve)
    );
    wrapper.find('form').simulate('submit');
    await sleep(100);
    wrapper.update();
    wrapper.find('textarea').getDOMNode<HTMLTextAreaElement>().value = validToken;
    wrapper.find('textarea').simulate('input');

    expect(wrapper.find('.auth-panel-email-login-form__error').text()).toBe('Token is expired');
  });
});
