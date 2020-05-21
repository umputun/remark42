const userNameRegex = /^[\p{L}\d_ ]+$/u;
export function validateUserName(userName: string) {
  return userNameRegex.test(userName.trim());
}
