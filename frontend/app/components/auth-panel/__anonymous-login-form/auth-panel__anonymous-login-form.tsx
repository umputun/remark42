/** @jsx h */
import { h, Component, RenderableProps } from 'preact';
import b from 'bem-react-helper';
import { Theme } from '@app/common/types';

interface Props {
  onSubmit(username: string): Promise<void>;
  theme: Theme;
  className?: string;
}

interface State {
  inputValue: string;
  honeyPotValue: boolean;
}

export class AnonymousLoginForm extends Component<Props, State> {
  static usernameRegex = /^[a-zA-Z][\w ]+$/;

  inputRef?: HTMLInputElement;

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
    if (value.length < 3) return 'Username must be at least 3 characters long';
    if (!AnonymousLoginForm.usernameRegex.test(value))
      return 'Username must start from the letter and contain only latin letters, numbers, underscores, and spaces';
    return null;
  }

  onCheckedChange(e: Event) {
    this.setState({ honeyPotValue: (e.target as HTMLInputElement).checked });
  }

  componentDidUpdate() {
    setTimeout(() => {
      this.inputRef && this.inputRef.focus();
    }, 100);
  }

  render(props: RenderableProps<Props>) {
    // TODO: will be great to `b` to accept `string | undefined | (string|undefined)[]` as classname
    let className = b('auth-panel-anonymous-login-form', {}, { theme: props.theme });
    if (props.className) {
      className += ' ' + b('auth-panel-anonymous-login-form', {}, { theme: props.theme });
    }

    const usernameInvalidReason = this.getUsernameInvalidReason();

    return (
      <form className={className} onSubmit={this.onSubmit}>
        <input
          className="auth-panel-anonymous-login-form__input"
          ref={ref => (this.inputRef = ref)}
          type="text"
          placeholder="Username"
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
        <input
          className="auth-panel-anonymous-login-form__submit"
          type="submit"
          value="Log in"
          title={usernameInvalidReason || ''}
          disabled={usernameInvalidReason !== null}
        />
      </form>
    );
  }
}
