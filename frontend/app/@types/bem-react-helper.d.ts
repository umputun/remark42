declare module 'bem-react-helper' {
  export interface Mods {
    [key: string]: string | number | boolean | undefined | null;
  }
  export type Mix = Array<string | undefined> | string;
  export default function b(
    classname: string,
    props?: {
      mods?: Mods;
      mix?: Mix;
    },
    override_props?: Mods
  ): string;
}
