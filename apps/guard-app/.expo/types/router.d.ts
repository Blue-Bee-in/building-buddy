/* eslint-disable */
import * as Router from 'expo-router';

export * from 'expo-router';

declare module 'expo-router' {
  export namespace ExpoRouter {
    export interface __routes<T extends string | object = string> {
      hrefInputParams: { pathname: Router.RelativePathString, params?: Router.UnknownInputParams } | { pathname: Router.ExternalPathString, params?: Router.UnknownInputParams } | { pathname: `/_sitemap`; params?: Router.UnknownInputParams; } | { pathname: `${'/(gate)'}/expected` | `/expected`; params?: Router.UnknownInputParams; } | { pathname: `${'/(gate)'}` | `/`; params?: Router.UnknownInputParams; } | { pathname: `${'/(gate)'}/scan` | `/scan`; params?: Router.UnknownInputParams; };
      hrefOutputParams: { pathname: Router.RelativePathString, params?: Router.UnknownOutputParams } | { pathname: Router.ExternalPathString, params?: Router.UnknownOutputParams } | { pathname: `/_sitemap`; params?: Router.UnknownOutputParams; } | { pathname: `${'/(gate)'}/expected` | `/expected`; params?: Router.UnknownOutputParams; } | { pathname: `${'/(gate)'}` | `/`; params?: Router.UnknownOutputParams; } | { pathname: `${'/(gate)'}/scan` | `/scan`; params?: Router.UnknownOutputParams; };
      href: Router.RelativePathString | Router.ExternalPathString | `/_sitemap${`?${string}` | `#${string}` | ''}` | `${'/(gate)'}/expected${`?${string}` | `#${string}` | ''}` | `/expected${`?${string}` | `#${string}` | ''}` | `${'/(gate)'}${`?${string}` | `#${string}` | ''}` | `/${`?${string}` | `#${string}` | ''}` | `${'/(gate)'}/scan${`?${string}` | `#${string}` | ''}` | `/scan${`?${string}` | `#${string}` | ''}` | { pathname: Router.RelativePathString, params?: Router.UnknownInputParams } | { pathname: Router.ExternalPathString, params?: Router.UnknownInputParams } | { pathname: `/_sitemap`; params?: Router.UnknownInputParams; } | { pathname: `${'/(gate)'}/expected` | `/expected`; params?: Router.UnknownInputParams; } | { pathname: `${'/(gate)'}` | `/`; params?: Router.UnknownInputParams; } | { pathname: `${'/(gate)'}/scan` | `/scan`; params?: Router.UnknownInputParams; };
    }
  }
}
