package stages

// MaxNamedMissingIDsForTest exposes the production truncation cap to the
// external black-box tests without adding it to the package's runtime API.
const MaxNamedMissingIDsForTest = maxNamedMissingIDs
