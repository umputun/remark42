// hack for https://github.com/developit/preact-compat/issues/475
// should be changed when preact@10 is available

import React from 'preact';
import { createContext } from 'preact-context';

React.createContext = createContext;
