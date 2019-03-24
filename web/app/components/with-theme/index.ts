/** @jsx h */
import { AnyComponent } from 'preact';
import { connect } from 'preact-redux';
import { StoreState } from '@app/store';
import { Theme } from '@app/common/types';

/**
 * Connects redux theme property to component's
 */
// eslint-disable-next-line @typescript-eslint/explicit-function-return-type
function withTheme<P extends { theme: Theme }, S extends object>(PlainComponent: AnyComponent<P, S>) {
  return connect((state: StoreState) => ({ theme: state.theme }))(PlainComponent);
}

export default withTheme;
