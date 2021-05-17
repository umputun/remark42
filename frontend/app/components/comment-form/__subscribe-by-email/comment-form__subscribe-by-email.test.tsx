import { h } from 'preact';
import { act } from 'preact/test-utils';
import { Provider } from 'react-redux';
import { Middleware } from 'redux';
import createMockStore from 'redux-mock-store';
import { IntlProvider } from 'react-intl';
import { fireEvent, waitFor, render, getByText } from '@testing-library/preact';

jest.mock('common/api');

import { user, anonymousUser } from '__stubs__/user';
import { validToken } from '__stubs__/jwt';
import { mockStore } from '__stubs__/store';
import * as api from 'common/api';
import { sleep } from 'utils/sleep';
import { Input } from 'components/input';
import { Button } from 'components/button';
import { Dropdown } from 'components/dropdown';
import enMessages from 'locales/en.json';
import { LS_EMAIL_KEY } from 'common/constants';

import { SubscribeByEmail, SubscribeByEmailForm } from '.';
import { debug } from 'webpack';

const initialStore = {
  user,
  theme: 'light',
} as const;

jest.mock('utils/jwt', () => ({
  isJwtExpired: jest.fn(() => false),
}));

describe('<SubscribeByEmail/>', () => {
  const createContainer = (store: ReturnType<typeof mockStore> = mockStore(initialStore)) =>
    render(
      <IntlProvider locale="en" messages={enMessages}>
        <Provider store={store}>
          <SubscribeByEmail />
        </Provider>
      </IntlProvider>
    );

  it('should be rendered with disabled email button when user is anonymous', () => {
    const store = mockStore({ ...initialStore, user: anonymousUser });
    const { container } = createContainer(store);
    const dropdown = container.querySelector('.dropdown__title');

    expect(dropdown).toHaveAttribute('disabled');
    expect(dropdown).toHaveAttribute('title', 'Available only for registered users');
  });

  it('should be rendered with enabled email button when user is registrated', () => {
    const store = mockStore(initialStore);
    const { container } = createContainer(store);
    const dropdown = container.querySelector('.dropdown__title');

    expect(dropdown).not.toHaveAttribute('disabled');
    expect(dropdown).toHaveAttribute('title', 'Subscribe by Email');
  });
});

describe('<SubscribeByEmailForm/>', () => {
  const createContainer = (store: ReturnType<typeof mockStore> = mockStore(initialStore)) =>
    render(
      <IntlProvider locale="en" messages={enMessages}>
        <Provider store={store}>
          <SubscribeByEmailForm />
        </Provider>
      </IntlProvider>
    );
  it('should render email form by default', () => {
    const store = mockStore(initialStore);
    const { container } = createContainer(store);
    const title = container.querySelector('.comment-form__subscribe-by-email__title');
    const button = container.querySelector('.comment-form__subscribe-by-email__button');

    expect(title).toHaveTextContent('Subscribe to replies');
    expect(button).toHaveTextContent('Submit');
    expect(button).toHaveAttribute('disabled');
  });

  it('should render subscribed state if user subscribed', () => {
    const store = mockStore({ ...initialStore, user: { email_subscription: true } });
    const { container } = createContainer(store);

    expect(container.querySelector('.comment-form__subscribe-by-email_subscribed')).toHaveTextContent(
      /^You are subscribed on updates by email/
    );
  });

  it('should pass throw subscribe process', async () => {
    const { container, getByRole, getByPlaceholderText, getByText, debug } = createContainer();
    const input = getByPlaceholderText('Email');
    const button = getByText('Submit');
    const emailVerificationForSubscribe = jest.spyOn(api, 'emailVerificationForSubscribe');
    const emailConfirmationForSubscribe = jest.spyOn(api, 'emailConfirmationForSubscribe');

    fireEvent.input(input, 'some@email.com');
    console.log((input as HTMLInputElement).value);
    await waitFor(() => expect(button).not.toHaveAttribute('disabled'));
    fireEvent.click(button);
    await waitFor(() => expect(emailVerificationForSubscribe).toHaveBeenCalledWith('some@email.com'));

    const textarea = getByRole('textarea');

    expect(getByText('Back')).toBeInTheDocument();
    expect(getByText('Subscribe')).toBeInTheDocument();

    fireEvent.input(textarea, 'tokentokentoken');
    fireEvent.click(getByText('Subscribe'));

    expect(emailConfirmationForSubscribe).toHaveBeenCalledWith('tokentokentoken');
    expect(container.querySelector('.comment-form__subscribe-by-email')).toHaveTextContent(
      'You have been subscribed on updates by email'
    );
    expect(getByRole('button')).toEqual('Unsubscribe');
  });
  //   it('should fill in email from local storage', async () => {
  //     localStorage.setItem(LS_EMAIL_KEY, 'someone@email.com');
  //     const { container } = createContainer();
  //     const form = container.querySelector('form');
  //     expect(form.find('input').props().value).toEqual('someone@email.com');
  //   });
  //   it('should send form by paste valid token', async () => {
  //     const { container } = createContainer();
  //     const onInputEmail = container.querySelector(Input).prop('onInput');
  //     const form = container.querySelector('form');
  //     expect(typeof onInputEmail === 'function').toBe(true);
  //     act(() => onInputEmail(makeInputEvent('some@email.com')));
  //     form.simulate('submit');
  //     await sleep(0);
  //     wrapper.update();
  //     const textarea = container.querySelector('textarea');
  //     textarea.getDOMNode<HTMLTextAreaElement>().value = validToken;
  //     textarea.simulate('input');
  //     await sleep(0);
  //     wrapper.update();
  //     expect(wrapper.text().startsWith('You have been subscribed on updates by email')).toBe(true);
  //     expect(container.querySelector(Button).text()).toEqual('Unsubscribe');
  //   });
  //   it('should pass throw unsubscribe process', async () => {
  //     const store = mockStore({ ...initialStore, user: { email_subscription: true } });
  //     const { container } = createContainer(store);
  //     const onClick = container.querySelector(Button).prop('onClick');
  //     expect(typeof onClick === 'function').toBe(true);
  //     act(() => onClick());
  //     expect(unsubscribeFromEmailUpdatesMock).toHaveBeenCalled();
  //     await sleep(0);
  //     wrapper.update();
  //     expect(wrapper.text().startsWith('You have been unsubscribed by email to updates')).toBe(true);
  //     expect(container.querySelector(Button).text()).toEqual('Close');
  //   });
});
