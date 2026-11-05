// Applies the saved theme before first paint to avoid a flash. Loaded as an
// external script because the CSP blocks inline scripts.
(function () {
  var pref = localStorage.getItem('cork:theme') || 'system';
  var dark = pref === 'dark' || (pref === 'system' && matchMedia('(prefers-color-scheme: dark)').matches);
  document.documentElement.dataset.theme = dark ? 'dark' : 'light';
})();
