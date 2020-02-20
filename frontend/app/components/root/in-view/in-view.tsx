import { Component, JSX } from 'preact';
import { sleep } from '@app/utils/sleep';

interface Props {
  children: (props: { inView: boolean; ref: (ref: Component) => unknown }) => JSX.Element;
}

interface State {
  inView: boolean;
  ref: Element | undefined;
}

let instanceMap: WeakMap<Element, Component<Props, State>>;
let observer: IntersectionObserver;

function getObserver(): { observer: IntersectionObserver; instanceMap: WeakMap<Element, Component<Props, State>> } {
  if (observer && instanceMap) {
    return { observer, instanceMap };
  }
  instanceMap = new WeakMap<Element, Component<Props, State>>();
  observer = new window.IntersectionObserver(
    entries => {
      entries.forEach(e => {
        const instance = instanceMap.get(e.target);
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
  return { observer, instanceMap };
}

export class InView extends Component<Props, State> {
  state: State = {
    inView: false,
    ref: undefined,
  };

  componentWillUpdate(_nextProps: Props, nextState: State) {
    if (this.state.ref === nextState.ref) return;

    if (this.state.ref instanceof Element) {
      const { observer, instanceMap } = getObserver();
      observer.unobserve(this.state.ref);
      instanceMap.delete(this.state.ref);
    }

    if (nextState.ref instanceof Element) {
      const { observer, instanceMap } = getObserver();
      observer.observe(nextState.ref);
      instanceMap.set(nextState.ref, this);
    }
  }

  refSetter = async (ref: Component | null) => {
    await sleep(1);
    const el = ref ? ref.base : undefined;
    if (el === this.state.ref) return;
    this.setState({
      ref: ref ? (ref.base as Element) : undefined,
    });
  };

  componentWillUnmount() {
    if (!(this.state.ref instanceof Element)) return;
    const { observer, instanceMap } = getObserver();
    observer.unobserve(this.state.ref);
    instanceMap.delete(this.state.ref);
  }

  render() {
    const props = { inView: this.state.inView, ref: this.refSetter };
    const r = this.props.children(props);
    return r;
  }
}
