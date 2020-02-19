/** @jsx createElement */
import { createElement, Component, createRef } from 'preact';
import { forwardRef } from 'preact/compat';
import b from 'bem-react-helper';
import { useSelector } from 'react-redux';
import { Theme, User } from '@app/common/types';
import { sendEmailVerificationRequest } from '@app/common/api';
import { extractErrorMessageFromResponse } from '@app/utils/errorUtils';
import { getHandleClickProps } from '@app/common/accessibility';
import { sleep } from '@app/utils/sleep';
import TextareaAutosize from '@app/components/comment-form/textarea-autosize';
import { Input } from '@app/components/input';
import { Button } from '@app/components/button';
import { isJwtExpired } from '@app/utils/jwt';
import { IntlShape, useIntl } from 'react-intl';

const mapStateToProps = () => ({
  sendEmailVerification: sendEmailVerificationRequest,
});

interface OwnProps {
  onSignIn(token: string): Promise<User | null>;
  onSuccess?(user: User): Promise<void>;
  theme: Theme;
  className?: string;
}

export type Props = OwnProps & ReturnType<typeof mapStateToProps> & { intl: IntlShape };

export interface State {
  usernameValue: string;
  addressValue: string;
  tokenValue: string;
  verificationSent: boolean;
  loading: boolean;
  error: string | null;
}

export class EmailLoginForm extends Component<Props, State> {
  static usernameRegex = /^[a-zA-Z][\w ]+$/;
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
    this.tokenRef.current && this.tokenRef.current.textareaRef && this.tokenRef.current.textareaRef.select();
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
    try {
      this.setState({ loading: true });
      const user = await this.props.onSignIn(token);
      if (!user) {
        this.setState({ error: 'No user was found' });
        return;
      }
      this.setState({ verificationSent: false, tokenValue: '' });
      this.props.onSuccess && this.props.onSuccess(user);
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
    const { value } = e.target as HTMLInputElement;

    this.setState({ error: null, tokenValue: value });

    try {
      if (value.length > 0 && isJwtExpired(value)) {
        this.setState({ error: 'Token is expired' });
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
    if (this.state.loading) return 'Loading...';
    const username = this.state.usernameValue;
    if (username.length < 3) return 'Username must be at least 3 characters long';
    if (!EmailLoginForm.usernameRegex.test(username))
      return 'Username must start from the letter and contain only latin letters, numbers, underscores, and spaces';
    if (!EmailLoginForm.emailRegex.test(this.state.addressValue)) return 'Address should be valid email address';
    return null;
  }

  getForm2InvalidReason(): string | null {
    if (this.state.loading) return 'Loading...';
    if (this.state.tokenValue.length === 0) return 'Token field must not be empty';
    return null;
  }

  render(props: Props) {
    // TODO: will be great to `b` to accept `string | undefined | (string|undefined)[]` as classname
    let className = b('auth-panel-email-login-form', {}, { theme: props.theme });
    if (props.className) {
      className += ' ' + b('auth-panel-email-login-form', {}, { theme: props.theme });
    }

    const form1InvalidReason = this.getForm1InvalidReason();

    if (!this.state.verificationSent)
      return (
        <form className={className} onSubmit={this.onVerificationSubmit}>
          <Input
            autoFocus
            mix="auth-panel-email-login-form__input"
            ref={this.usernameInputRef}
            placeholder="Username"
            value={this.state.usernameValue}
            onInput={this.onUsernameChange}
          />
          <Input
            mix="auth-panel-email-login-form__input"
            type="email"
            placeholder="Email Address"
            value={this.state.addressValue}
            onInput={this.onAddressChange}
          />
          {this.state.error && <div className="auth-panel-email-login-form__error">{this.state.error}</div>}
          <Button
            mix="auth-panel-email-login-form__submit"
            kind="primary"
            size="middle"
            type="submit"
            title={form1InvalidReason || ''}
            disabled={form1InvalidReason !== null}
          >
            Send Verification
          </Button>
        </form>
      );

    const form2InvalidReason = this.getForm2InvalidReason();

    return (
      <form className={className} onSubmit={this.onSubmit}>
        <Button kind="link" mix="auth-panel-email-login-form__back-button" {...getHandleClickProps(this.goBack)}>
          Back
        </Button>
        <TextareaAutosize
          autofocus={true}
          className="auth-panel-email-login-form__token-input"
          ref={this.tokenRef}
          placeholder="Token"
          value={this.state.tokenValue}
          onInput={this.onTokenChange}
          spellcheck={false}
          autocomplete="off"
        />
        {this.state.error && <div className="auth-panel-email-login-form__error">{this.state.error}</div>}
        <Button
          mix="auth-panel-email-login-form__submit"
          type="submit"
          kind="primary"
          size="middle"
          title={form2InvalidReason || ''}
          disabled={form2InvalidReason !== null}
        >
          Confirm
        </Button>
      </form>
    );
  }
}

export type EmailLoginFormRef = EmailLoginForm;

export const EmailLoginFormConnected = forwardRef<EmailLoginForm, OwnProps>((props, ref) => {
  const connectedProps = useSelector(mapStateToProps);
  const intl = useIntl();
  return <EmailLoginForm {...props} {...connectedProps} intl={intl} ref={ref} />;
});
