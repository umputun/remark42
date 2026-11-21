import type { Component, VNode } from 'preact';
import { createContext } from 'preact';
import type { PropRef } from 'preact/hooks';
import { useState, useEffect, useRef } from 'preact/hooks';

/**
 * Ids of comments to render in full whether or not they are in view. A comment out of view is an
 * empty placeholder of a guessed size, so whatever is measured against placeholders moves once they
 * turn into comments.
 */
export const RevealedComments = createContext<ReadonlySet<string>>(new Set());

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
  children: <T>(props: { inView: boolean; ref: PropRef<T> }) => VNode;
};

export function InView({ children }: Props) {
  const [inView, setInView] = useState(false);
  const ref = useRef<Component<unknown, unknown>>(null);

  useEffect(() => {
    const element = ref.current;
    const { observer, instanceMap } = getObserver();

    if (!(element?.base instanceof Element)) {
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
}
