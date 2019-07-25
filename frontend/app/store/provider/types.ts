export const PROVIDER_UPDATE = 'PROVIDER/UPDATE';
export interface PROVIDER_UPDATE_ACTION {
  type: typeof PROVIDER_UPDATE;
  payload: {
    name: string;
  };
}

export type PROVIDER_ACTIONS = PROVIDER_UPDATE_ACTION;
