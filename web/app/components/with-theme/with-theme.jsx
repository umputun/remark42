/** @jsx h */
import { h } from 'preact';
import store from 'common/store';

const withTheme = Component => {
  const ThemedComponent = props => {
    const { mods = {} } = props;
    const theme = store.get('theme');

    return <Component {...props} mods={{ ...mods, theme }} />;
  };

  return ThemedComponent;
};

export default withTheme;
