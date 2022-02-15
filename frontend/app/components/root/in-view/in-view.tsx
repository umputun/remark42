import { Component, VNode } from 'preact';
import { useState, useEffect, useRef, Ref } from 'preact/hooks';

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

type Props = {
  children: <T>(props: { inView: boolean; ref: Ref<T> }) => VNode;
};

export function InView({ children }: Props) {
  const [inView, setInView] = useState(false);
  const ref = useRef<Component<unknown, unknown>>(null);

  useEffect(() => {
    const componentBase = ref.current?.base;
    const { observer, instanceMap } = getObserver();

    if (!(componentBase instanceof Element)) {
      return;
    }

    observer.observe(componentBase);
    instanceMap.set(componentBase, setInView);

    return () => {
      observer.unobserve(componentBase);
      instanceMap.delete(componentBase);
    };
  }, []);

  return children({ inView, ref });
}
