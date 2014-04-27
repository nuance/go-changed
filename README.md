# go-changed

go-changed takes a list of changed files on stdin and outputs a list of packages that import (directly or indirectly) packages
containing these changed files.

This is intended to be used to filter deploy targets - given a list of targets corresponding to individual import paths,
determine with paths need to be updated.
