import { PROVIDER_ACTIONS, PROVIDER_UPDATE } from './types';

export interface ProviderState {
  name: string | null;
}

function provider(state: ProviderState = { name: null }, action: PROVIDER_ACTIONS): ProviderState {
  switch (action.type) {
    case PROVIDER_UPDATE: {
      return { ...state, ...action.payload };
    }
    default:
      return state;
  }
}

export default { provider };
