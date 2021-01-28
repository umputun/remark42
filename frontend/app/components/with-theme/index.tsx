import { h, FunctionComponent, ComponentType } from 'preact';
import { useSelector } from 'react-redux';
import { StoreState } from 'store';
import { Theme } from 'common/types';

const themeSelector = (state: StoreState) => state.theme;

/**
 * Connects redux theme property to component's
 */
function withTheme<P extends { theme: Theme }>(PlainComponent: ComponentType<P>) {
  const C: FunctionComponent<Omit<P, 'theme'>> = (props) => {
    const theme = useSelector(themeSelector);
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    return <PlainComponent theme={theme} {...(props as any)} />;
  };
  C.displayName = `withTheme(${PlainComponent.displayName || PlainComponent.name})`;

  return C;
}

export default withTheme;
