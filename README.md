Smithy
===============================================================================

*smithy* (n) A blacksmith's shop; a forge.

Smithy is a web frontend for git repositories.  It's implemented entirely in
Golang, compiles to a single binary, and it's fast and easy to deploy.  Smithy
is an alternative to cgit or gitweb, and doesn't seek to compete with Gitea and
the like.

* Golang
* Single binary
* Easy to deploy
* Fast
* Customizable
* Free software
* Javascript-free

Building
-------------------------------------------------------------------------------

The only dependency is [Golang](https://golang.org/).

```
$ git clone https://github.com/honza/smithy
$ make
$ ./smithy --help
```

Installing
-------------------------------------------------------------------------------

```
$ smithy generate > config.yaml
$ smithy serve --config smithy.yaml
```

Configuration
-------------------------------------------------------------------------------

``` yaml
title: Smithy, a lightweight git forge
description: Publish your git repositories with ease
port: 3456
git:
  root: "/var/www/git"
  repos:
    - path: "some-cool-project"
      slug: "some-cool-project"
      title: "Some Cool Project"
      description: "Something really cool to change the world"
    - path: "ugly-hacks"
      exclude: true

static:
  root:
  prefix: /static/

templates:
  dir:
```

Customizing templates and css
-------------------------------------------------------------------------------

Out of the box, smithy bundles templates and css in the binary.  Setting
`static.root`, and `templates.dir` to empty string will cause smithy to use the
bundled assets.

If you'd like to customize the templates or the css, copy the `include`
directory somewhere, and then set `static.root`, and `templates.dir` to that
directory.

Demo
-------------------------------------------------------------------------------

Smithy is currently hosting [itself on my
domain](https://smithy.honza.ca/smithy).

Contributing
-------------------------------------------------------------------------------

Contributions are most welcome.  You can open a pull request on
[GitHub](https://github.com/honza/smithy), or [email a patch][1] to `me@honza.ca`.

[1]: https://git-send-email.io

TODO
-------------------------------------------------------------------------------

* Add support for go modules
* Add routes for commits ending in .patch

License
-------------------------------------------------------------------------------

This program is free software: you can redistribute it and/or modify it under
the terms of the GNU General Public License as published by the Free Software
Foundation, either version 3 of the License, or (at your option) any later
version.

This program is distributed in the hope that it will be useful, but WITHOUT ANY
WARRANTY; without even the implied warranty of MERCHANTABILITY or FITNESS FOR A
PARTICULAR PURPOSE.  See the GNU General Public License for more details.

You should have received a copy of the GNU General Public License along with
this program.  If not, see <https://www.gnu.org/licenses/>.
