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

const mapStateToProps = () => ({
  sendEmailVerification: sendEmailVerificationRequest,
});

interface OwnProps {
  onSignIn(token: string): Promise<User | null>;
  onSuccess?(user: User): Promise<void>;
  theme: Theme;
  className?: string;
}

export type Props = OwnProps & ReturnType<typeof mapStateToProps>;

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

  inputRef = createRef<HTMLInputElement>();
  tokenRef = createRef<TextareaAutosize>();

  constructor(props: Props) {
    super(props);

    this.state = {
      usernameValue: '',
      addressValue: '',
      tokenValue: '',
      verificationSent: false,
      loading: false,
      error: null,
    };

    this.focus = this.focus.bind(this);
    this.onVerificationSubmit = this.onVerificationSubmit.bind(this);
    this.onSubmit = this.onSubmit.bind(this);
    this.onUsernameChange = this.onUsernameChange.bind(this);
    this.onAddressChange = this.onAddressChange.bind(this);
    this.onTokenChange = this.onTokenChange.bind(this);
    this.goBack = this.goBack.bind(this);
  }

  async focus() {
    await sleep(100);
    if (this.inputRef.current) {
      this.inputRef.current.focus();
      return;
    }
    this.tokenRef.current && this.tokenRef.current.textareaRef && this.tokenRef.current.textareaRef.select();
  }

  async onVerificationSubmit(e: Event) {
    e.preventDefault();
    this.setState({ loading: true, error: null });
    try {
      await this.props.sendEmailVerification(this.state.usernameValue, this.state.addressValue);
      this.setState({ verificationSent: true });
      setTimeout(() => {
        this.tokenRef.current && this.tokenRef.current.focus();
      }, 100);
    } catch (e) {
      this.setState({ error: extractErrorMessageFromResponse(e) });
    } finally {
      this.setState({ loading: false });
    }
  }

  async onSubmit(e: Event) {
    e.preventDefault();
    try {
      this.setState({ loading: true });
      const user = await this.props.onSignIn(this.state.tokenValue);
      if (!user) {
        this.setState({ error: 'No user was found' });
        return;
      }
      this.setState({ verificationSent: false, tokenValue: '' });
      this.props.onSuccess && this.props.onSuccess(user);
    } catch (e) {
      this.setState({ error: extractErrorMessageFromResponse(e) });
    } finally {
      this.setState({ loading: false });
    }
  }

  onUsernameChange(e: Event) {
    this.setState({ error: null, usernameValue: (e.target as HTMLInputElement).value });
  }

  onAddressChange(e: Event) {
    this.setState({ error: null, addressValue: (e.target as HTMLInputElement).value });
  }

  onTokenChange(e: Event) {
    this.setState({ error: null, tokenValue: (e.target as HTMLInputElement).value });
  }

  goBack() {
    this.setState({
      tokenValue: '',
      error: null,
      verificationSent: false,
    });
    setTimeout(() => {
      this.inputRef.current && this.inputRef.current.focus();
    }, 100);
  }

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

  componentDidMount() {
    setTimeout(() => {
      this.inputRef.current && this.inputRef.current.focus();
    }, 100);
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
          {/*
           * We adding hidden span element to bear with DropDown's onOutSideClick handler.
           * This function checks if element that was clicked is a children of it's root component.
           * And the problem is that by the time handler gets executed our target element is not a
           * part of a dom, so handler suggests that we clicked somewhere outside and hides dropdown
           */}
          <span
            className="auth-panel-email-login-form__back-button"
            {...getHandleClickProps(this.goBack)}
            style={{ display: 'none' }}
          >
            {'< Back'}
          </span>
          <Input
            mix="auth-panel-email-login-form__input"
            ref={this.inputRef}
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
          {this.state.error && <div className="auth-panel-email-login-form__error">{this.state.error}</div>}
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
        {this.state.error && <div className="auth-panel-email-login-form__error">{this.state.error}</div>}
      </form>
    );
  }
}

export type EmailLoginFormRef = EmailLoginForm;

export const EmailLoginFormConnected = forwardRef<EmailLoginForm, OwnProps>((props, ref) => {
  const connectedProps = useSelector(mapStateToProps);
  return <EmailLoginForm {...props} {...connectedProps} ref={ref} />;
});
