/** @jsx createElement */
import { createElement, Component, createRef } from 'preact';
import b from 'bem-react-helper';
import { IntlShape, defineMessages, FormattedMessage } from 'react-intl';
import { Theme } from '@app/common/types';

import { Input } from '@app/components/input';
import { Button } from '@app/components/button';

interface Props {
  onSubmit(username: string): Promise<void>;
  theme: Theme;
  className?: string;
  intl: IntlShape;
}

interface State {
  inputValue: string;
  honeyPotValue: boolean;
}

const messages = defineMessages({
  lengthLimit: {
    id: 'anonymousLoginForm.length-limit',
    defaultMessage: 'Username must be at least 3 characters long',
  },
  symbolLimit: {
    id: 'anonymousLoginForm.symbol-limit',
    defaultMessage:
      'Username must start from the letter and contain only latin letters, numbers, underscores, and spaces',
  },
  userName: {
    id: 'anonymousLoginForm.user-name',
    defaultMessage: 'Username',
  },
});

export class AnonymousLoginForm extends Component<Props, State> {
  static usernameRegex = /^[a-zA-Z][\w ]+$/;

  inputRef = createRef<HTMLInputElement>();

  constructor(props: Props) {
    super(props);

    this.state = {
      inputValue: '',
      honeyPotValue: false,
    };

    this.onSubmit = this.onSubmit.bind(this);
    this.onChange = this.onChange.bind(this);
    this.onCheckedChange = this.onCheckedChange.bind(this);
  }

  onSubmit(e: Event) {
    e.preventDefault();
    if (this.state.honeyPotValue) {
      // what should i do if bot uncovered?
      window.location.reload();
      return;
    }
    this.props.onSubmit(this.state.inputValue);
  }

  onChange(e: Event) {
    this.setState({ inputValue: (e.target as HTMLInputElement).value });
  }

  getUsernameInvalidReason(): string | null {
    const value = this.state.inputValue;
    const intl = this.props.intl;
    if (value.length < 3) return intl.formatMessage(messages.lengthLimit);
    if (!AnonymousLoginForm.usernameRegex.test(value)) return intl.formatMessage(messages.symbolLimit);
    return null;
  }

  onCheckedChange(e: Event) {
    this.setState({ honeyPotValue: (e.target as HTMLInputElement).checked });
  }

  componentDidUpdate() {
    setTimeout(() => {
      this.inputRef.current && this.inputRef.current.focus();
    }, 100);
  }

  render() {
    const props = this.props;
    const intl = props.intl;
    // TODO: will be great to `b` to accept `string | undefined | (string|undefined)[]` as classname
    let className = b('auth-panel-anonymous-login-form', {}, { theme: props.theme });
    if (props.className) {
      className += ' ' + b('auth-panel-anonymous-login-form', {}, { theme: props.theme });
    }

    const usernameInvalidReason = this.getUsernameInvalidReason();

    return (
      <form className={className} onSubmit={this.onSubmit}>
        <Input
          ref={this.inputRef}
          mix="auth-panel-anonymous-login-form__input"
          placeholder={intl.formatMessage(messages.userName)}
          value={this.state.inputValue}
          onInput={this.onChange}
        />
        {/* honeypot input */}
        <input
          className="auth-panel-anonymous-login-form__remember-me"
          type="checkbox"
          tabIndex={-1}
          autocomplete="off"
          onChange={this.onCheckedChange}
          checked={this.state.honeyPotValue}
        />
        <Button
          mix="auth-panel-anonymous-login-form__submit"
          type="submit"
          kind="primary"
          size="middle"
          title={usernameInvalidReason || ''}
          disabled={usernameInvalidReason !== null}
        >
          <FormattedMessage id="anonymousLoginForm.log-in" defaultMessage="Log in" />
        </Button>
      </form>
    );
  }
}
