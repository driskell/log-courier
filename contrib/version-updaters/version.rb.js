module.exports = {
  readVersion(contents) {
    const match = contents.match(/VERSION = '([^']+)'/)
    if (!match) {
      throw new Error('Could not parse version.rb');
    }
    return match[1];
  },
  writeVersion(contents, version) {
    const [major, minor, patch] = version.split('.');
    return contents
      .replace(/(MAJOR_VERSION =) \d+/, `$1 ${major}`)
      .replace(/(MINOR_VERSION =) \d+/, `$1 ${minor}`)
      .replace(/(PATCH_VERSION =) \d+/, `$1 ${patch}`)
      .replace(/(VERSION =) '[^']+'/, `$1 '${version}'`);
  }
};
