/** @jsx h */
import { h } from 'preact';
import { mount } from 'enzyme';
import { EmailLoginForm, Props, State } from './auth-panel__email-login-form';
import { User } from '@app/common/types';
import { sleep } from '@app/utils/sleep';

describe('EmailLoginForm', () => {
  it('works', async () => {
    const testUser = ({} as any) as User;
    const sendEmailVerification = jest.fn(async () => {});
    const onSignIn = jest.fn(async () => testUser);
    const onSuccess = jest.fn(async () => {});
    const el = mount<Props, State>(
      <EmailLoginForm
        sendEmailVerification={sendEmailVerification}
        onSignIn={onSignIn}
        onSuccess={onSuccess}
        theme="light"
      />
    );
    await new Promise(resolve =>
      el.setState(
        {
          usernameValue: 'someone',
          addressValue: 'someone@example.com',
        } as State,
        resolve
      )
    );
    el.find('form').simulate('submit');
    await sleep(100);
    expect(sendEmailVerification).toBeCalledWith('someone', 'someone@example.com');
    expect(el.state().verificationSent).toBe(true);

    await new Promise(resolve =>
      el.setState(
        {
          tokenValue: 'abcd',
        } as State,
        resolve
      )
    );

    el.find('form').simulate('submit');
    await sleep(100);
    expect(onSignIn).toBeCalledWith('abcd');
    expect(onSuccess).toBeCalledWith(testUser);
  });
});
