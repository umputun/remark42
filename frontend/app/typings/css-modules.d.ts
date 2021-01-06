declare module '*.module.css' {
  type ClassNames = {
    [className: string]: string;
  };

  const classNames: ClassNames;
  export = classNames;
}
