# BEM → CSS Modules migration, batch 1: leaf components

## Overview
- Migrate 4 leaf BEM components to CSS Modules: **button**, **dropdown**, **thread**, **auth-panel**
- Consolidates 19 BEM CSS files into 4 CSS module files
- Cleans up dead CSS classes and unused props discovered during analysis
- Follows the pattern established in PR #2013 (batch 0: dropdown-item, list-comments, subscribe-by-rss, settings)

## Context
- PR #2013 migrated 4 small components with zero visual regression (verified pixel-by-pixel on built artefacts)
- Key learning from batch 0: `mix` is pure string concatenation — parent and child can be migrated independently in any order
- These 4 "leaf" components are migrated first because they have no inward `mix` coupling with unmigrated parents (or the coupling is dead)
- Batch 2 (comment-form, subscribe-by-email, comment, root) follows after this lands

## Development Approach
- **Testing approach**: Regular (run existing tests after each change; no new tests needed as these are CSS-only changes with no logic)
- Complete each task fully before moving to the next
- Make small, focused changes
- Run `cd frontend && pnpm lint` and `cd frontend && pnpm test` after each task

## Progress Tracking
- Mark completed items with `[x]` immediately when done
- Add newly discovered tasks with ➕ prefix
- Document issues/blockers with ⚠️ prefix

## Implementation Steps

### Task 1: Migrate `button` to CSS Modules

7 BEM CSS files → 1 `button.module.css`. Uses lookup objects for dynamic kind/size/theme props.

**Files to modify:**
- `frontend/apps/remark42/app/components/button/button.tsx`
- `frontend/apps/remark42/app/components/button/index.ts`

**Files to create:**
- `frontend/apps/remark42/app/components/button/button.module.css`

**Files to delete (7):**
- `button/button.css`
- `button/_kind/_link/button_kind_link.css` + parent dirs
- `button/_kind/_primary/button_kind_primary.css` + parent dirs
- `button/_kind/_secondary/button_kind_secondary.css` + parent dirs
- `button/_size/_large/button_size_large.css` + parent dirs
- `button/_size/_middle/button_size_middle.css` + parent dirs
- `button/_theme/_dark/button_theme_dark.css` + parent dirs

**Steps:**
- [x] Create `button.module.css` consolidating all 7 CSS files:
  ```css
  .root {
    background: none;
    border: 0;
    padding: 0;
    margin: 0;
    border-radius: 4px;
    font-family: inherit;
    font-size: inherit;
    cursor: pointer;
    white-space: nowrap;

    &:focus {
      box-shadow: 0 0 0 2px var(--color47);
      outline: none;
    }

    &:disabled {
      opacity: 0.6;
      cursor: default;
    }
  }

  .kindLink {
    background: transparent;
    font-weight: 600;
    color: var(--color9);

    &:hover { color: var(--color33); }
    &:disabled, &:hover:disabled { color: var(--color9); }
  }

  .kindPrimary {
    background: var(--color15);
    color: var(--color6);

    &:hover { background: var(--color33); }
    &:hover:disabled { background: var(--color15); }
  }

  .kindSecondary {
    background: var(--color6);
    color: inherit;

    &:hover { box-shadow: inset 0 0 0 2px var(--color33); }
  }

  .sizeMiddle {
    height: 2rem;
    padding: 0 12px;
  }

  .sizeLarge {
    height: 36px;
    padding: 0 12px;
    font-size: 16px;
  }

  .themeDark {
    &.kindSecondary {
      background: var(--color8);
      color: var(--color20);
    }

    &.kindLink {
      &:disabled, &:hover:disabled { color: var(--color6); }
    }
  }
  ```
- [x] Update `button.tsx`:
  - Remove `import b, { Mods, Mix } from 'bem-react-helper'`
  - Add `import styles from './button.module.css'`
  - Add lookup objects:
    ```tsx
    const kindStyles: Record<string, string> = {
      primary: styles.kindPrimary,
      secondary: styles.kindSecondary,
      link: styles.kindLink,
    };

    const sizeStyles: Record<string, string> = {
      middle: styles.sizeMiddle,
      large: styles.sizeLarge,
    };
    ```
  - Remove `mods` from `ButtonProps` type and destructuring (never passed by any caller)
  - Change `Mix` type import to plain `string | string[]` for `mix` prop
  - Change className to: `clsx(styles.root, kind && kindStyles[kind], size && sizeStyles[size], theme === 'dark' && styles.themeDark, mix, className)`
- [x] Update `index.ts`: remove all 7 CSS imports, keep only `export { Button } from './button'`
- [x] Delete all 7 old CSS files and their BEM directories
- [x] Run `cd frontend && pnpm lint && pnpm test` — must pass before next task

### Task 2: Migrate `dropdown` to CSS Modules

7 BEM CSS files → 1 `dropdown.module.css`. Replaces 3 DOM class queries with refs and `data-dropdown` attribute. Removes dead `heading` prop (no callers) and dead `dropdown__heading` class (no CSS). Updates already-migrated `dropdown-item.module.css`.

**Files to modify:**
- `frontend/apps/remark42/app/components/dropdown/dropdown.tsx`
- `frontend/apps/remark42/app/components/dropdown/index.ts`
- `frontend/apps/remark42/app/components/dropdown/__item/dropdown-item.module.css` (change `:global(.dropdown)` → `[data-dropdown]`)

**Files to create:**
- `frontend/apps/remark42/app/components/dropdown/dropdown.module.css`

**Files to delete (7):**
- `dropdown/dropdown.css`
- `dropdown/__content/dropdown__content.css` + dir
- `dropdown/__items/dropdown__items.css` + dir
- `dropdown/__title/dropdown__title.css` + dir
- `dropdown/_active/dropdown_active.css` + dir
- `dropdown/_theme/_dark/dropdown_theme_dark.css` + dirs
- `dropdown/_theme/_light/dropdown_theme_light.css` + dirs

**Steps:**
- [x] Create `dropdown.module.css` consolidating all 7 CSS files:
  ```css
  .root {
    display: inline-block;
    position: relative;
  }

  .content {
    position: absolute;
    z-index: 20;
    outline-width: 0;
    display: none;
    top: 100%;
    left: 0;
    transform: translate(-0.5em, 5px);
    min-width: 120px;
    max-width: 260px;
    border: 2px solid var(--color15);
    border-radius: 3px;
    padding: 0 0 5px;
  }

  .items {
    padding: 5px 0;

    &:last-child { padding-bottom: 0; }
  }

  .title {
    &::after {
      content: '\25BE';
      margin-left: 2px;
    }
  }

  .active > .content {
    display: block;
  }

  .themeDark > .content {
    background-color: var(--color8);
  }

  .themeLight > .content {
    background-color: var(--color6);
  }
  ```
- [x] Update `dropdown.tsx`:
  - Replace `import b from 'bem-react-helper'` with `import clsx from 'clsx'` and `import styles from './dropdown.module.css'`
  - Add `contentRef = createRef<HTMLDivElement>()` alongside existing `rootNode` ref
  - Add `data-dropdown` attribute to root div (for nested dropdown detection by parent traversal)
  - Replace 3 DOM queries with ref:
    - Line 78: `parent.classList.contains('dropdown')` → `parent.hasAttribute('data-dropdown')`
    - Line 95: `Array.from(...).find(c => c.classList.contains('dropdown__content'))` → `this.contentRef.current`
    - Line 118: `this.rootNode.current.querySelector('.dropdown__content')` → `this.contentRef.current`
  - Root div: `className={clsx(styles.root, isActive && styles.active, theme === 'dark' ? styles.themeDark : styles.themeLight, mix)}`
  - Content div: `className={styles.content}` with `ref={this.contentRef}`
  - Items div: `className={styles.items}`
  - Button mix: `mix={[styles.title, titleClass]}` (clsx in Button handles arrays)
  - Remove dead `heading` prop from Props type and the heading div from JSX (prop is never passed by any caller, class has no CSS)
- [x] Update `dropdown-item.module.css`: change `& > :global(.dropdown)` to `& > [data-dropdown]`
- [x] Update `index.ts`: remove all 7 CSS imports, keep `export { Dropdown }` and `export { DropdownItem }`
- [x] Delete all 7 old CSS files and their BEM directories
- [x] Run `cd frontend && pnpm lint && pnpm test` — must pass before next task

### Task 3: Migrate `thread` to CSS Modules

3 BEM CSS files → 1 `thread.module.css`. Levels 0-5 had no CSS rules — only level 6 matters. The `mix` prop (receives `"root__thread"` from root component) is passed through as a plain class string.

**Files to modify:**
- `frontend/apps/remark42/app/components/thread/thread.tsx`
- `frontend/apps/remark42/app/components/thread/index.ts`

**Files to create:**
- `frontend/apps/remark42/app/components/thread/thread.module.css`

**Files to delete (3):**
- `thread/thread.css`
- `thread/__collapse/thread__collapse.css` + dir
- `thread/_theme_dark/thread_theme_dark.css` + dir

**Steps:**
- [x] Create `thread.module.css` consolidating all 3 CSS files:
  ```css
  .root {
    position: relative;
  }

  .indented {
    margin-left: 17px;
  }

  .level6 .level6 {
    margin-left: 0;
  }

  .collapse {
    height: calc(100% - 50px);
    width: 11px;
    position: absolute;
    top: 50px;
    left: -4px;
    cursor: pointer;

    &::after {
      display: block;
      content: '';
      position: absolute;
      left: 5px;
      top: 0;
      border-left: 1px dotted var(--color35);
      height: 100%;
    }

    &:hover::after {
      transform: translateX(-1px);
      border-left: 3px solid var(--color10);
      z-index: 10;
    }
  }

  .collapsed {
    composes: collapse;
    width: 18px;
    height: 18px;
    top: 12px;
    left: 0;
    display: flex;
    text-align: center;
    opacity: 0.8;
    border-radius: 2px;
    border: 1px solid;

    &::after { display: none; }
    &:hover { opacity: 1; }
    &:hover::after { transform: translateX(0); }

    & > div {
      position: relative;
      top: 6px;
      left: 3px;
      width: 12px;
      height: 2px;
      border-bottom: 2px solid;

      &::before, &::after {
        content: '';
        width: 100%;
        height: 2px;
        border-bottom: 2px solid;
        position: absolute;
        top: -4px;
        left: 0;
      }

      &::after { top: 4px !important; }
    }
  }

  .themeDark {
    & .collapse {
      &::after { border-color: var(--color36); }
      &:hover::after { border-color: var(--color6); }
    }
  }
  ```
- [x] Update `thread.tsx`:
  - Replace `import b from 'bem-react-helper'` with `import clsx from 'clsx'` and `import styles from './thread.module.css'`
  - Root div: `className={clsx(styles.root, indented && styles.indented, level === 6 && styles.level6, theme === 'dark' && styles.themeDark, mix)}`
  - Collapse div: `className={collapsed ? styles.collapsed : styles.collapse}`
- [x] Update `index.ts`: remove all 3 CSS imports, keep only `export { Thread } from './thread'`
- [x] Delete all 3 old CSS files and their BEM directories
- [x] Run `cd frontend && pnpm lint && pnpm test` — must pass before next task

### Task 4: Migrate `auth-panel` BEM remnants to CSS Modules

2 BEM CSS files → merge into existing `auth-panel.module.css`. Removes dead global class strings alongside module classes. Has test file to update.

**Dead code to clean up:**
- `auth-panel__pseudo-link` — no CSS rules, remove from JSX
- `auth-panel_loggedIn` / `auth-panel_theme_*` — dead mods from `b()`, no CSS rules
- `clsx('user', styles.user)` etc. — bare global strings (`'user'`, `'user-profile-button'`, `'user-avatar'`, `'user-logout-button'`) have no CSS; remove from `clsx()`

**Files to modify:**
- `frontend/apps/remark42/app/components/auth-panel/auth-panel.tsx`
- `frontend/apps/remark42/app/components/auth-panel/auth-panel.module.css`
- `frontend/apps/remark42/app/components/auth-panel/auth-panel.test.tsx`
- `frontend/apps/remark42/app/components/auth-panel/index.ts`

**Files to delete (2):**
- `auth-panel/auth-panel.css`
- `auth-panel/__column/auth-panel__column.css` + dir

**Steps:**
- [x] Merge BEM styles into existing `auth-panel.module.css` — add these classes after existing ones:
  ```css
  .root {
    display: flex;
    justify-content: space-between;
    font-size: 14px;
    line-height: 16px;
    align-items: center;
  }

  .column:last-child {
    margin-left: 8px;
    text-align: right;
  }

  .columnSeparated > * + * {
    position: relative;
    display: inline-block;
    margin-left: 20px;

    &::before {
      position: absolute;
      left: -15px;
      display: inline-block;
      width: 10px;
      text-align: center;
      content: '•';
    }
  }

  .adminAction { }
  ```
- [x] Update `auth-panel.tsx`:
  - Remove `import b from 'bem-react-helper'`
  - Root div: `className={styles.root}` (drop dead `theme`/`loggedIn` mods)
  - Column divs: `className={styles.column}`
  - Separated column: `className={clsx(styles.column, styles.columnSeparated)}`
  - Remove `className="auth-panel__pseudo-link"` from `<a>` (dead class, no CSS)
  - Clean up dual class patterns: `clsx('user', styles.user)` → `styles.user`, same for userButton/userAvatar/userLogoutButton
  - Change `mix="auth-panel__admin-action"` → `className={styles.adminAction}` on both Button calls
- [x] Update `auth-panel.test.tsx`:
  - Add `import styles from './auth-panel.module.css'`
  - `.find('.auth-panel__admin-action')` → `` .find(`.${styles.adminAction}`) ``
  - `.find('.auth-panel__column')` → `` .find(`.${styles.column}`) ``
- [x] Update `index.ts`: remove 2 CSS imports, keep `export * from './auth-panel'`
- [x] Delete `auth-panel.css` and `__column/` directory
- [x] Run `cd frontend && pnpm lint && pnpm test` — must pass before next task

### Task 5: Verify acceptance criteria
- [x] Verify all 4 components use CSS Modules (no remaining `b()` calls in migrated files)
- [x] Run full frontend test suite: `cd frontend && pnpm test`
- [x] Run frontend linter: `cd frontend && pnpm lint`
- [x] Grep for old BEM class names to verify no remaining references to deleted CSS files
- [x] Verify `bem-react-helper` is no longer imported in any of the 4 migrated components

## Technical Details

### Class naming convention
- BEM block → `root`
- BEM element → camelCase of element name (`dropdown__content` → `content`, `auth-panel__column` → `column`)
- BEM modifier → camelCase (`button_kind_primary` → `kindPrimary`, `thread_theme_dark` → `themeDark`)
- Combined modifier → compound `&.` nesting (`button_theme_dark.button_kind_secondary` → `.themeDark { &.kindSecondary { ... } }`)

### Button `mix` prop after migration
- Type changes from `Mix` (bem-react-helper) to `string | string[] | undefined`
- Passed directly to `clsx()` which handles all these types
- Callers in batch 2 (not yet migrated) continue passing BEM strings — works fine
- Already-migrated callers pass module hashes — works fine

### Dropdown DOM query replacements
| Before | After |
|---|---|
| `classList.contains('dropdown')` | `hasAttribute('data-dropdown')` |
| `querySelector('.dropdown__content')` | `this.contentRef.current` |
| `Array.from(...).find(c => c.classList.contains('dropdown__content'))` | `this.contentRef.current` |

### Files to delete (total: 19 CSS files + BEM directories)
- Button: 7 CSS files
- Dropdown: 7 CSS files
- Thread: 3 CSS files
- Auth-panel: 2 CSS files

## Post-Completion
- Visual smoke test: `cd frontend && pnpm dev:app`, verify button variants, dropdown open/close, thread collapse/expand, auth-panel admin actions in both light and dark themes
- Built artefact comparison: build docker image from branch, compare against master (same method as PR #2013)
- Batch 2 (comment-form, subscribe-by-email, comment, root) as follow-up
