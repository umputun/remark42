const handleBtnKeyPress = (event, handler) => {
  if (event.key === ' ' || event.key === 'Enter') {
    event.preventDefault();
    handler && handler();
  }
};

export const getHandleClickProps = handler => ({
  role: 'button',
  tabIndex: 0,
  onClick: handler,
  onKeyPress: event => handleBtnKeyPress(event, handler),
});
