smithy.yml(5)

# NAME

*smithy.yml* - configuration file for *smithy*(1)

# DESCRIPTION

This file describes where smithy should scan for repositories, their
respectives titles, description and slug. Also if it should include the default
styles and assets or if it should load from a respective directory.

# GLOBAL DIRECTIVES

*host: <address>*
	Address will be displayed on a repository, indicating the URL he can use
	to clone.

*port: <...>*
	Port to serve smithy from. You can use a reverse-proxy (nginx, apache) to
	expose smithy.

# GIT DIRECTIVES

*root: <path>*
	The main directory where smithy should scan for repositories.

*repos*
	A list of repositories and their respective configurations.

# STATIC DIRECTIVES

If you'd like to customize the templates or the css, you can grab the source
code, copy the `include` directory somewhere, and then set `root`, and
`templates.dir` to that directory.

*root: <path>*
	When set to an empty string, it will load the static assets bundled within
	the project.

*prefix: <path>*
	A given prefix that all assets will receive.

# TEMPLATES DIRECTIVES

*dir: <path>*
	The directory to load templates from.

# EXAMPLE CONFIGURATION

When manually building smithy from source, a sample config file will be
included on `/usr/local/share/smithy/smithy.yml`.

```
title: Smithy, a lightweight git force
description: Publish your git repositories with ease
host: git.example.com
port: 3456
git:
  root: "/srv/git"
  repos:
    - path: "git"
      slug: "git"
      title: "git"
      description: "git is a fast, scalable distributed revision control system"
static:
  root: ""
  prefix: /static/
templates:
  dir: ""
```

# AUTHORS

Maintained by Honza Pokorny <honza@pokorny.ca>, who is assisted by other free
software contributors. For more information about smithy development, see
https://github.com/honza/smithy.
