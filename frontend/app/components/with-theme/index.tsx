/** @jsx createElement */
import { createElement, FunctionComponent, ComponentType } from 'preact';
import { useSelector } from 'react-redux';
import { StoreState } from '@app/store';
import { Theme } from '@app/common/types';

const themeSelector = (state: StoreState) => state.theme;

/**
 * Connects redux theme property to component's
 */
function withTheme<P extends { theme: Theme }>(PlainComponent: ComponentType<P>) {
  const o: FunctionComponent<Omit<P, 'theme'>> = props => {
    const theme = useSelector(themeSelector);
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    return <PlainComponent theme={theme} {...(props as any)} />;
  };
  o.displayName = `withTheme(${PlainComponent.displayName || PlainComponent.name})`;

  return o;
}

export default withTheme;
