import { h, Component } from 'preact';

export default class Input extends Component {
  constructor(props) {
    super(props);

    this.autoResize = this.autoResize.bind(this);
  }

  autoResize() {
    this.rootNode.style.height = '';
    this.setState({ height: this.rootNode.scrollHeight });
  }

  render(props, { height }) {
    return (
      <textarea
        className={b('input', props)}
        onInput={this.autoResize}
        style={{ height }}
        ref={r => (this.rootNode = r)}
      >{props.children}</textarea>
    );
  }
}
