export const parseJwt = (token: string) => {
  const [, base64Url] = token.split('.');
  const base64 = base64Url.replace(/-/g, '+').replace(/_/g, '/');
  const jsonPayload = decodeURIComponent(
    atob(base64)
      .split('')
      .map(c => '%' + ('00' + c.charCodeAt(0).toString(16)).slice(-2))
      .join('')
  );

  return JSON.parse(jsonPayload);
};

export const isJwtExpired = (token: string) => {
  const { exp } = parseJwt(token);

  return exp * 1000 < Date.now();
};
