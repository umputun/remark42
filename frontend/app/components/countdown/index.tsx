import { h, JSX, Component, createRef } from 'preact';
import { exclude } from 'utils/exclude';

type Props = {
  time: Date;
  onTimePassed?: () => void;
} & JSX.HTMLAttributes;

interface State {
  /** props.time converted to timestamp */
  time: number;
}

/** Component which uses plain DOM mutation instead of rerendering react reactive reactivity */
export default class Countdown extends Component<Props, State> {
  elemRef = createRef<HTMLSpanElement>();
  intervalID?: number;
  constructor(props: Props) {
    super(props);
    this.state = {
      time: props.time.getTime(),
    };
  }
  componentDidMount() {
    this.start();
  }
  componentWillReceiveProps(nextProps: Props) {
    if (nextProps.time === this.props.time) return;
    this.setState({
      time: nextProps.time.getTime(),
    });
    this.start();
  }
  componentWillUnmount() {
    window.clearInterval(this.intervalID);
  }
  shouldComponentUpdate() {
    return false;
  }
  tick() {
    if (this.elemRef) {
      const value = Math.max(0, (this.state.time - new Date().getTime()) / 1000).toFixed(0);
      this.elemRef.current!.innerText = value;
      if (value === '0') {
        this.props.onTimePassed && this.props.onTimePassed();
        window.clearInterval(this.intervalID);
        this.intervalID = undefined;
      }
    }
  }
  start() {
    if (this.intervalID) clearInterval(this.intervalID);
    this.tick();
    this.intervalID = window.setInterval(() => {
      this.tick();
    }, 1000);
  }
  render(props: Props) {
    return <span {...exclude(props, 'time', 'onTimePassed')} ref={this.elemRef} />;
  }
}
