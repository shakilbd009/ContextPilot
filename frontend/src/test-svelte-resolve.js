const mod = await import('svelte');
console.log(Object.keys(mod).slice(0, 10));
console.log('mount:', typeof mod.mount);