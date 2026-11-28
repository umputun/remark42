import { h } from 'preact';
import { useLayoutEffect, useRef } from 'preact/hooks';

import { useIntl } from 'common/intl';
import { useTheme } from 'hooks/useTheme';

import { mountSubscribeByRss } from './subscribe-by-rss-element.js';

export { createSubscribeUrl } from './subscribe-by-rss-element.js';

/**
 * Mounts the plain RSS dropdown into a div preact renders without props or children, so a
 * re-render of the form leaves the div's attributes and content to the plain component.
 */
export function SubscribeByRSS({ userId }: { userId: string | null }) {
  const theme = useTheme();
  const intl = useIntl();
  const root = useRef<HTMLDivElement>(null);
  const rss = useRef<ReturnType<typeof mountSubscribeByRss> | null>(null);

  useLayoutEffect(() => {
    rss.current = mountSubscribeByRss(root.current!, { userId, theme }, intl);
    return () => rss.current?.destroy();
    // mounted once; the props of every later render reach it through update below
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  useLayoutEffect(() => {
    rss.current?.update({ userId, theme }, intl);
  });

  return <div ref={root} />;
}
