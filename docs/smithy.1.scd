smithy(1)

# NAME

smithy - a small git forge

# SYNOPSIS

*smithy* _command_ [<_options_>...]

# DESCRIPTION

*smithy* is a web frontend for git repositories. It's implemented entirely in
Golang, compiles to a single binary, and it's fast and easy to deploy. Smithy
is an alternative to cgit or gitweb, and doesn't seek to compete with Gitea and
the like.

# COMMANDS

*generate*
	Generate a sample configuration file, outputs to *STDOUT*.
	Check *smithy.yml(5)* for more information.

*serve --config path/to/config.toml*
	Serve the application, you'll need to supply a configuration file.
	Outputs its log to *STDOUT*.

# GLOBAL FLAGS

*--debug*
	Display debug messages to *STDOUT*.

*--config <path>*
	Use the given configuration file. See *smithy.yml(5)* for a reference.

# AUTHORS

Maintained by Honza Pokorny <honza@pokorny.ca>, who is assisted by other free
software contributors. For more information about smithy development, see
https://github.com/honza/smithy.
