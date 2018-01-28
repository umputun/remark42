import { h, Component } from 'preact';

export default class Input extends Component {
  constructor(props) {
    super(props);

    this.autoResize = this.autoResize.bind(this);
  }

  autoResize() {
    this.fieldNode.style.height = '';
    this.setState({ height: this.fieldNode.scrollHeight });
  }

  render(props, { height }) {
    return (
      <div className={b('input', props)}>
        <textarea
          className="input__field"
          onInput={this.autoResize}
          style={{ height }}
          ref={r => (this.fieldNode = r)}
          required
        >
        {props.children}
        </textarea>

        <button className="input__button" type="button">Send</button>
      </div>
    );
  }
}
