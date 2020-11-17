/** @jsx createElement */
import { createElement, Component, createRef } from 'preact';
import { forwardRef } from 'preact/compat';
import b from 'bem-react-helper';
import { Theme, User } from '@app/common/types';
import { sendEmailVerificationRequest } from '@app/common/api';
import { extractErrorMessageFromResponse } from '@app/utils/errorUtils';
import { getHandleClickProps } from '@app/common/accessibility';
import { sleep } from '@app/utils/sleep';
import TextareaAutosize from '@app/components/comment-form/textarea-autosize';
import { Input } from '@app/components/input';
import { Button } from '@app/components/button';
import { isJwtExpired } from '@app/utils/jwt';
import { defineMessages, IntlShape, useIntl, FormattedMessage } from 'react-intl';

import { validateUserName } from '../validateUserName';

import { messages as loginForm } from '../__anonymous-login-form/auth__anonymous-login-form';
import { LS_EMAIL_KEY } from '@app/common/constants';

interface OwnProps {
  onSignIn(token: string): Promise<User | null>;
  onSuccess?(user: User): Promise<void>;
  theme: Theme;
  className?: string;
}

export type Props = OwnProps & { intl: IntlShape; sendEmailVerification: typeof sendEmailVerificationRequest };

export interface State {
  usernameValue: string;
  addressValue: string;
  tokenValue: string;
  verificationSent: boolean;
  loading: boolean;
  error: string | null;
}

const messages = defineMessages({
  expiredToken: {
    id: 'emailLoginForm.expired-token',
    defaultMessage: 'Token is expired',
  },
  userNotFound: {
    id: 'emailLoginForm.user-not-found',
    defaultMessage: 'No user was found',
  },
  loading: {
    id: 'emailLoginForm.loading',
    defaultMessage: 'Loading...',
  },
  invalidEmail: {
    id: 'emailLoginForm.invalid-email',
    defaultMessage: 'Address should be valid email address',
  },
  emptyToken: {
    id: 'emailLoginForm.empty-token',
    defaultMessage: 'Token field must not be empty',
  },
  emailAddress: {
    id: 'emailLoginForm.email-address',
    defaultMessage: 'Email Address',
  },
  token: {
    id: 'emailLoginForm.token',
    defaultMessage: 'Token',
  },
});

export class EmailLoginForm extends Component<Props, State> {
  static emailRegex = /[^@]+@[^.]+\..+/;

  usernameInputRef = createRef<HTMLInputElement>();
  tokenRef = createRef<TextareaAutosize>();

  state = {
    usernameValue: '',
    addressValue: '',
    tokenValue: '',
    verificationSent: false,
    loading: false,
    error: null,
  };

  focus = async () => {
    await sleep(100);
    if (this.usernameInputRef.current) {
      this.usernameInputRef.current.focus();
      return;
    }
    if (this.tokenRef.current?.textareaRef?.current) {
      this.tokenRef.current.textareaRef.current.select();
    }
  };

  onVerificationSubmit = async (e: Event) => {
    e.preventDefault();
    this.setState({ loading: true, error: null });
    try {
      await this.props.sendEmailVerification(this.state.usernameValue, this.state.addressValue);
      this.setState({ verificationSent: true });
      setTimeout(() => {
        this.tokenRef.current && this.tokenRef.current.focus();
      }, 100);
    } catch (e) {
      this.setState({ error: extractErrorMessageFromResponse(e, this.props.intl) });
    } finally {
      this.setState({ loading: false });
    }
  };

  async sendForm(token: string = this.state.tokenValue) {
    const intl = this.props.intl;
    try {
      this.setState({ loading: true });
      const user = await this.props.onSignIn(token);
      if (!user) {
        this.setState({ error: intl.formatMessage(messages.userNotFound) });
        return;
      }
      this.setState({ verificationSent: false, tokenValue: '' });
      localStorage.setItem(LS_EMAIL_KEY, this.state.addressValue);
      if (this.props.onSuccess) {
        await this.props.onSuccess(user);
      }
    } catch (e) {
      this.setState({ error: extractErrorMessageFromResponse(e, this.props.intl) });
    } finally {
      this.setState({ loading: false });
    }
  }

  onSubmit = async (e: Event) => {
    e.preventDefault();
    this.sendForm();
  };

  onUsernameChange = (e: Event) => {
    this.setState({ error: null, usernameValue: (e.target as HTMLInputElement).value });
  };

  onAddressChange = (e: Event) => {
    this.setState({ error: null, addressValue: (e.target as HTMLInputElement).value });
  };

  onTokenChange = (e: Event) => {
    const intl = this.props.intl;
    const { value } = e.target as HTMLInputElement;

    this.setState({ error: null, tokenValue: value });

    try {
      if (value.length > 0 && isJwtExpired(value)) {
        this.setState({ error: intl.formatMessage(messages.expiredToken) });
        return;
      }
      this.sendForm(value);
    } catch (e) {}
  };

  goBack = async () => {
    // Wait for finding back button in DOM by dropbox
    // It prevents dropdown from closing, because if dropdown doesn't find clicked element it closes
    await sleep(0);

    this.setState({
      tokenValue: '',
      error: null,
      verificationSent: false,
    });

    // Wait for rendering username+email step to find user input
    await sleep(0);

    if (this.usernameInputRef.current) {
      this.usernameInputRef.current.focus();
    }
  };

  getForm1InvalidReason(): string | null {
    const intl = this.props.intl;
    if (this.state.loading) return intl.formatMessage(messages.loading);
    const username = this.state.usernameValue;
    if (username.length < 3) return intl.formatMessage(loginForm.lengthLimit);
    if (!validateUserName(username)) return intl.formatMessage(loginForm.symbolLimit);
    if (!EmailLoginForm.emailRegex.test(this.state.addressValue)) return intl.formatMessage(messages.invalidEmail);
    return null;
  }

  getForm2InvalidReason(): string | null {
    const intl = this.props.intl;
    if (this.state.loading) return intl.formatMessage(messages.loading);
    if (this.state.tokenValue.length === 0) return intl.formatMessage(messages.emptyToken);
    return null;
  }

  render(props: Props) {
    const intl = props.intl;
    // TODO: will be great to `b` to accept `string | undefined | (string|undefined)[]` as classname
    let className = b('auth-email-login-form', {}, { theme: props.theme });
    if (props.className) {
      className += ' ' + b('auth-email-login-form', {}, { theme: props.theme });
    }

    const form1InvalidReason = this.getForm1InvalidReason();

    if (!this.state.verificationSent)
      return (
        <form className={className} onSubmit={this.onVerificationSubmit}>
          <Input
            autoFocus
            name="username"
            mix="auth-email-login-form__input"
            ref={this.usernameInputRef}
            placeholder={intl.formatMessage(loginForm.userName)}
            value={this.state.usernameValue}
            onInput={this.onUsernameChange}
          />
          <Input
            mix="auth-email-login-form__input"
            type="email"
            name="email"
            placeholder={intl.formatMessage(messages.emailAddress)}
            value={this.state.addressValue}
            onInput={this.onAddressChange}
          />
          {this.state.error && <div className="auth-email-login-form__error">{this.state.error}</div>}
          <Button
            mix="auth-email-login-form__submit"
            kind="primary"
            size="middle"
            type="submit"
            title={form1InvalidReason || ''}
            disabled={form1InvalidReason !== null}
          >
            <FormattedMessage id="emailLoginForm.send-verification" defaultMessage="Send Verification" />
          </Button>
        </form>
      );

    const form2InvalidReason = this.getForm2InvalidReason();

    return (
      <form className={className} onSubmit={this.onSubmit}>
        <Button kind="link" mix="auth-email-login-form__back-button" {...getHandleClickProps(this.goBack)}>
          <FormattedMessage id="emailLoginForm.back" defaultMessage="Back" />
        </Button>
        <TextareaAutosize
          autofocus={true}
          name="token"
          className="auth-email-login-form__token-input"
          ref={this.tokenRef}
          placeholder={intl.formatMessage(messages.token)}
          value={this.state.tokenValue}
          onInput={this.onTokenChange}
          spellcheck={false}
          autocomplete="off"
        />
        {this.state.error && <div className="auth-email-login-form__error">{this.state.error}</div>}
        <Button
          mix="auth-email-login-form__submit"
          type="submit"
          kind="primary"
          size="middle"
          title={form2InvalidReason || ''}
          disabled={form2InvalidReason !== null}
        >
          <FormattedMessage id="emailLoginForm.confirm" defaultMessage="Confirm" />
        </Button>
      </form>
    );
  }
}

export type EmailLoginFormRef = EmailLoginForm;

export const EmailLoginFormConnected = forwardRef<EmailLoginForm, OwnProps>((props, ref) => {
  const intl = useIntl();
  return <EmailLoginForm {...props} sendEmailVerification={sendEmailVerificationRequest} intl={intl} ref={ref} />;
});
