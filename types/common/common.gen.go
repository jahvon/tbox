// Code generated by github.com/atombender/go-jsonschema, DO NOT EDIT.

package common

// Alternate names that can be used to reference the executable in the CLI.
type Aliases []string

// A list of tags.
// Tags can be used with list commands to filter returned data.
type Tags []string

type Visibility string

const VisibilityHidden Visibility = "hidden"
const VisibilityInternal Visibility = "internal"
const VisibilityPrivate Visibility = "private"
const VisibilityPublic Visibility = "public"
