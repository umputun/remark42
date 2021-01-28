import { setItem, getItem } from 'common/local-storage';
import { StoreAction } from 'store';
import { PROVIDER_UPDATE_ACTION, PROVIDER_UPDATE } from './types';

const PROVIDER_LOCALSTORAGE_KEY = '__remarkProvider';

/** saves last login provider from localstorage and put to store */
export function updateProvider(payload: PROVIDER_UPDATE_ACTION['payload']): StoreAction<void, PROVIDER_UPDATE_ACTION> {
  return (dispatch) => {
    setItem(PROVIDER_LOCALSTORAGE_KEY, JSON.stringify(payload));
    dispatch({
      type: PROVIDER_UPDATE,
      payload,
    });
  };
}

/** restores last login provider from localstorage and put to store */
export function restoreProvider(): StoreAction<void, PROVIDER_UPDATE_ACTION> {
  return (dispatch) => {
    const payloadString = getItem(PROVIDER_LOCALSTORAGE_KEY);
    if (!payloadString) return;
    try {
      const payload = JSON.parse(payloadString);
      dispatch({
        type: PROVIDER_UPDATE,
        payload,
      });
    } catch {}
  };
}
