/** @jsx createElement */
import { JSX } from 'preact';
import { useState, useEffect, useRef, PropRef } from 'preact/hooks';

interface Props {
  children: <T>(props: { inView: boolean; ref: PropRef<T> }) => JSX.Element;
}

let instanceMap: WeakMap<Element, (inView: boolean) => void>;
let observer: IntersectionObserver;

function getObserver(): { observer: IntersectionObserver; instanceMap: WeakMap<Element, (inView: boolean) => void> } {
  if (observer && instanceMap) {
    return { observer, instanceMap };
  }
  instanceMap = new WeakMap<Element, (inView: boolean) => void>();
  observer = new window.IntersectionObserver(
    entries => {
      entries.forEach(e => {
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

export function InView({ children }: Props) {
  const [inView, setInView] = useState(false);
  const ref = useRef<any>(); // eslint-disable-line

  useEffect(() => {
    const { observer, instanceMap } = getObserver();
    if (ref.current) {
      observer.observe(ref.current.base);
      instanceMap.set(ref.current.base, setInView);
    }

    return () => {
      if (ref.current) {
        observer.unobserve(ref.current.base);
        instanceMap.delete(ref.current.base);
      }
    };
  });

  return children({ inView, ref });
}
