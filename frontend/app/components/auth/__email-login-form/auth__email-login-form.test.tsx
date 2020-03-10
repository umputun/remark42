/** @jsx createElement */
import { createElement } from 'preact';
import { mount, ReactWrapper } from 'enzyme';
import { EmailLoginFormConnected as EmailLoginForm, Props, State } from './auth__email-login-form';
import { User } from '@app/common/types';
import { sleep } from '@app/utils/sleep';
import { validToken } from '@app/testUtils/mocks/jwt';
import { sendEmailVerificationRequest } from '@app/common/api';
import { IntlProvider } from 'react-intl';
import enMessages from '../../../locales/en.json';

jest.mock('@app/utils/jwt', () => ({
  isJwtExpired: jest
    .fn()
    .mockImplementationOnce(() => true)
    .mockImplementationOnce(() => false)
    .mockImplementationOnce(() => true),
}));

jest.mock('@app/common/api');

function simulateInput(input: ReactWrapper, value: string) {
  input.getDOMNode<HTMLTextAreaElement>().value = value;
  input.simulate('input');
}

describe('EmailLoginForm', () => {
  const testUser = ({} as any) as User;
  const onSuccess = jest.fn(async () => {});
  const onSignIn = jest.fn(async () => testUser);

  beforeEach(() => {
    (sendEmailVerificationRequest as any).mockReset();
  });

  it('works', async () => {
    (sendEmailVerificationRequest as any).mockResolvedValueOnce({});
    const el = mount<Props, State>(
      <IntlProvider locale="en" messages={enMessages}>
        <EmailLoginForm onSignIn={onSignIn} onSuccess={onSuccess} theme="light" />
      </IntlProvider>
    );
    simulateInput(el.find(`input[name="email"]`), 'someone@example.com');
    simulateInput(el.find(`input[name="username"]`), 'someone');
    el.find('form').simulate('submit');
    await sleep(100);
    expect(sendEmailVerificationRequest).toBeCalledWith('someone', 'someone@example.com');
    el.update();
    simulateInput(el.find(`textarea[name="token"]`), 'abcd');

    el.find('form').simulate('submit');
    await sleep(100);
    expect(onSignIn).toBeCalledWith('abcd');
    expect(onSuccess).toBeCalledWith(testUser);
  });

  it('should send form by pasting token', async () => {
    (sendEmailVerificationRequest as any).mockResolvedValueOnce({});
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
    (sendEmailVerificationRequest as any).mockResolvedValueOnce({});
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
