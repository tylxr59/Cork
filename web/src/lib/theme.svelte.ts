export type ThemePref = 'system' | 'light' | 'dark';

const KEY = 'cork:theme';
const media = matchMedia('(prefers-color-scheme: dark)');

export const theme = $state<{ pref: ThemePref }>({
  pref: (localStorage.getItem(KEY) as ThemePref) || 'system',
});

function apply() {
  const dark = theme.pref === 'dark' || (theme.pref === 'system' && media.matches);
  document.documentElement.dataset.theme = dark ? 'dark' : 'light';
  document.querySelector('meta[name=theme-color]')?.setAttribute('content', dark ? '#1b1a19' : '#ffffff');
}

export function setTheme(pref: ThemePref) {
  theme.pref = pref;
  localStorage.setItem(KEY, pref);
  apply();
}

media.addEventListener('change', apply);
apply();
