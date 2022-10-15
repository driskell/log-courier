module.exports = {
  readVersion(contents) {
    const match = contents.match(/gem.version\s+= '([^']+)'/)
    if (!match) {
      throw new Error('Could not parse gemspec');
    }
    return match[1];
  },
  writeVersion(contents, version) {
    return contents
      .replace(/(gem.version\s+=) '[^']+'/, `$1 '${version}'`)
      .replace(/(gem.add_runtime_dependency 'log-courier',) '= [^']+'/, `$1 '= ${version}'`);
  }
};
