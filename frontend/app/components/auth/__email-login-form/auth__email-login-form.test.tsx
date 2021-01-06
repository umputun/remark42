import { h } from 'preact';
import { mount, ReactWrapper } from 'enzyme';
import { EmailLoginFormConnected as EmailLoginForm, Props, State } from './auth__email-login-form';
import { IntlProvider } from 'react-intl';

import { validToken } from '__stubs__/jwt';
import { LS_EMAIL_KEY } from 'common/constants';
import { User } from 'common/types';
import { sleep } from 'utils/sleep';
import { sendEmailVerificationRequest } from 'common/api';
import enMessages from 'locales/en.json';

jest.mock('utils/jwt', () => ({
  isJwtExpired: jest
    .fn()
    .mockImplementationOnce(() => true)
    .mockImplementationOnce(() => false)
    .mockImplementationOnce(() => true),
}));

jest.mock('common/api');

const sendEmailVerificationRequestMock = sendEmailVerificationRequest as jest.Mock<
  ReturnType<typeof sendEmailVerificationRequest>
>;

function simulateInput(input: ReactWrapper, value: string) {
  input.getDOMNode<HTMLTextAreaElement>().value = value;
  input.simulate('input');
}

describe('EmailLoginForm', () => {
  const testUser = {} as User;
  const onSuccess = jest.fn(async () => undefined);
  const onSignIn = jest.fn(async () => testUser);

  beforeEach(() => {
    sendEmailVerificationRequestMock.mockReset();
  });

  it('works', async () => {
    sendEmailVerificationRequestMock.mockResolvedValueOnce();
    const el = mount<Props, State>(
      <IntlProvider locale="en" messages={enMessages}>
        <EmailLoginForm onSignIn={onSignIn} onSuccess={onSuccess} theme="light" />
      </IntlProvider>
    );
    simulateInput(el.find(`input[name="email"]`), 'someone@example.com');
    simulateInput(el.find(`input[name="username"]`), 'someone');
    el.find('form').simulate('submit');
    await sleep(100);
    expect(sendEmailVerificationRequestMock).toBeCalledWith('someone', 'someone@example.com');
    el.update();
    simulateInput(el.find(`textarea[name="token"]`), 'abcd');

    el.find('form').simulate('submit');
    await sleep(100);
    expect(onSignIn).toBeCalledWith('abcd');
    expect(onSuccess).toBeCalledWith(testUser);
    //test that email is saved in local storage after email login
    expect(localStorage.getItem(LS_EMAIL_KEY)).toEqual('someone@example.com');
  });

  it('should send form by pasting token', async () => {
    sendEmailVerificationRequestMock.mockResolvedValueOnce();
    const onSignIn = jest.fn(async () => testUser);

    const wrapper = mount<Props, State>(
      <IntlProvider locale="en" messages={enMessages}>
        <EmailLoginForm onSignIn={onSignIn} onSuccess={onSuccess} theme="light" />
      </IntlProvider>
    );
    simulateInput(wrapper.find(`input[name="email"]`), 'someone@example.com');
    simulateInput(wrapper.find(`input[name="username"]`), 'someone');
    wrapper.find('form').simulate('submit');
    await sleep(100);
    wrapper.update();
    simulateInput(wrapper.find(`textarea[name="token"]`), validToken);
    await sleep(100);
    wrapper.update();
    expect(onSignIn).toBeCalledWith(validToken);
  });

  it('should show error "Token is expired" on paste', async () => {
    sendEmailVerificationRequestMock.mockResolvedValueOnce();
    const onSignIn = jest.fn(async () => testUser);

    const wrapper = mount<Props, State>(
      <IntlProvider locale="en" messages={enMessages}>
        <EmailLoginForm onSignIn={onSignIn} onSuccess={onSuccess} theme="light" />
      </IntlProvider>
    );
    simulateInput(wrapper.find(`input[name="email"]`), 'someone@example.com');
    simulateInput(wrapper.find(`input[name="username"]`), 'someone');
    wrapper.find('form').simulate('submit');
    await sleep(100);
    wrapper.update();
    wrapper.find('textarea').getDOMNode<HTMLTextAreaElement>().value = validToken;
    wrapper.find('textarea').simulate('input');

    expect(wrapper.find('.auth-email-login-form__error').text()).toBe('Token is expired');
  });
});
