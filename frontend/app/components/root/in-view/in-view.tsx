import { Component, VNode } from 'preact';
import { useState, useEffect, useRef, PropRef } from 'preact/hooks';

let instanceMap: WeakMap<Element, (inView: boolean) => void>;
let observer: IntersectionObserver;

function getObserver(): { observer: IntersectionObserver; instanceMap: WeakMap<Element, (inView: boolean) => void> } {
  if (observer && instanceMap) {
    return { observer, instanceMap };
  }
  instanceMap = new WeakMap<Element, (inView: boolean) => void>();
  observer = new window.IntersectionObserver(
    (entries) => {
      entries.forEach((e) => {
        const setInView = instanceMap.get(e.target);
        if (!setInView) return;
        setInView(e.isIntersecting);
      });
    },
    {
      rootMargin: '50px',
    }
  );
  return { observer, instanceMap };
}

type InViewProps = {
  children: <T>(props: { inView: boolean; ref: PropRef<T> }) => VNode;
};

const InView = ({ children }: InViewProps) => {
  const [inView, setInView] = useState(false);
  const ref = useRef<Component<unknown, unknown>>(null);

  useEffect(() => {
    const element = ref.current;
    const { observer, instanceMap } = getObserver();

    if (!(element.base instanceof Element)) {
      return;
    }

    observer.observe(element.base);
    instanceMap.set(element.base, setInView);

    return () => {
      if (!(element.base instanceof Element)) {
        return;
      }
      observer.unobserve(element.base);
      instanceMap.delete(element.base);
    };
  }, []);

  return children({ inView, ref });
};

export default InView;
