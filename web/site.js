const compiler = new URLSearchParams(location.search).get('compiler') === 'tinygo' ? 'tinygo' : 'go';
const name = compiler === 'tinygo' ? 'TinyGo' : 'Stock Go';
for (const link of document.querySelectorAll('[data-compiler]')) {
  if (link.dataset.compiler === compiler) link.setAttribute('aria-current', 'page');
}
const demo = document.querySelector('#demo');
demo.title = `ntcharts3d gallery — ${name}`;
demo.src = `${compiler}/`;
document.querySelector('#open-demo').href = `${compiler}/`;
document.title = `ntcharts3d — ${name} demo`;
try {
  const response = await fetch('builds.json');
  if (!response.ok) throw new Error(`Build details: HTTP ${response.status}`);
  const build = (await response.json())[compiler];
  const version = build.version.match(/version (?:go)?([\d.]+)/)?.[1] || build.version;
  document.querySelector('#build-info').textContent = `${build.compiler} ${version} · ${(build.bytes / 1048576).toFixed(1)} MiB WebAssembly`;
} catch (error) {
  console.warn(error);
  document.querySelector('#build-info').textContent = name;
}
