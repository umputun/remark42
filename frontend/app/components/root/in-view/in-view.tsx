import { Component } from 'preact';
import { sleep } from '@app/utils/sleep';

interface Props {
  children: (props: { inView: boolean; ref: (ref: Component) => Component }) => JSX.Element;
}

interface State {
  inView: boolean;
  ref: Element | undefined;
}

const instance_map: WeakMap<Element, Component<Props, State>> = new WeakMap();

const observer = new IntersectionObserver(
  entries => {
    entries.forEach(e => {
      const instance = instance_map.get(e.target);
      if (!instance) return;
      instance.setState({
        inView: e.isIntersecting,
      });
    });
  },
  {
    rootMargin: '50px',
  }
);

export class InView extends Component<Props, State> {
  state: State = {
    inView: false,
    ref: undefined,
  };

  componentWillUpdate(_nextProps: Props, nextState: State) {
    if (this.state.ref === nextState.ref) return;

    if (this.state.ref instanceof Element) {
      observer.unobserve(this.state.ref);
      instance_map.delete(this.state.ref);
    }

    if (nextState.ref instanceof Element) {
      observer.observe(nextState.ref);
      instance_map.set(nextState.ref, this);
    }
  }

  refSetter = async (ref: Component | null) => {
    await sleep(1);
    const el = ref ? ref.base : undefined;
    if (el === this.state.ref) return;
    this.setState({
      ref: ref ? ref.base : undefined,
    });
  };

  componentWillUnmount() {
    if (!(this.state.ref instanceof Element)) return;

    observer.unobserve(this.state.ref);
    instance_map.delete(this.state.ref);
  }

  render() {
    const props = { inView: this.state.inView, ref: this.refSetter };
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    const r = (this.props.children as any)[0](props);
    return r;
  }
}
