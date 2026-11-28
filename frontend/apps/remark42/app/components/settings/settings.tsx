import { h } from 'preact';
import { useLayoutEffect, useRef } from 'preact/hooks';

import { useTheme } from 'hooks/useTheme';

import type { SettingsProps } from './user-lists.js';
import { mountSettings } from './user-lists.js';

type Props = Omit<SettingsProps, 'theme'>;

/**
 * Mounts the plain settings into a div preact renders without props or children, so a re-render
 * of the page leaves the div's attributes and content to the plain component.
 */
export function Settings(props: Props) {
  const theme = useTheme();
  const root = useRef<HTMLDivElement>(null);
  const settings = useRef<ReturnType<typeof mountSettings> | null>(null);

  useLayoutEffect(() => {
    settings.current = mountSettings(root.current!, { ...props, theme });
    return () => settings.current?.destroy();
    // mounted once; the props and theme of every later render reach it through update below
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  // every render, so each button calls the handler the page passed last
  useLayoutEffect(() => {
    settings.current?.update({ ...props, theme });
  });

  return <div ref={root} />;
}
