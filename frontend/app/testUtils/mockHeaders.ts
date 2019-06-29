// eslint-disable-next-line @typescript-eslint/no-explicit-any
const originalHeaders = (window as any).Headers;

export const mockHeaders = {
  mock: () => {
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    (window as any).Headers = class {
      append() {}
      has() {
        return false;
      }
      get() {
        return null;
      }
    };
  },
  restore: () => {
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    (window as any).Headers = originalHeaders;
  },
};
