module.exports = {
  readVersion(contents) {
    const match = contents.match(/LogCourierVersion string = "([^"]+)"/)
    if (!match) {
      throw new Error('Could not parse version.go');
    }
    return match[1];
  },
  writeVersion(contents, version) {
    const [major, minor, patch] = version.split('.');
    return contents
      .replace(/(LogCourierMajorVersion uint32 =) \d+/, `$1 ${major}`)
      .replace(/(LogCourierMinorVersion uint32 =) \d+/, `$1 ${minor}`)
      .replace(/(LogCourierPatchVersion uint32 =) \d+/, `$1 ${patch}`)
      .replace(/(LogCourierVersion string =) "[^"]+"/, `$1 "${version}"`);
  }
};
