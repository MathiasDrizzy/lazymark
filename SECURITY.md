# Security policy

## Supported versions

Only the latest minor release receives security fixes (currently 1.2.x).

| Version | Supported |
| ------- | --------- |
| 1.2.x   | yes       |
| < 1.2   | no        |

## Reporting a vulnerability

Please do not open a public issue for a security problem.

Use GitHub's private vulnerability reporting: open the
[Security tab](https://github.com/MathiasDrizzy/lazymark/security) of this repository and choose
"Report a vulnerability". If that is not available to you, contact the maintainer through the
contact details on the [GitHub profile](https://github.com/MathiasDrizzy).

Include the lazymark version (`lazymark --version`), your operating system, and the steps to reproduce.
You can expect an acknowledgement within a few days. Once a fix is released, the report is credited in
the [changelog](CHANGELOG.md) unless you ask to stay anonymous.

## Verifying a release

Every release archive and its SBOM (an SPDX JSON file next to each archive, made by Syft) is listed in `checksums.txt`,
and `checksums.txt` has a signed build provenance attestation that ties the files to the commit and the workflow
that built them. With the GitHub CLI and nothing else installed:

```sh
gh attestation verify lazymark_<version>_<os>_<arch>.tar.gz -R MathiasDrizzy/lazymark
```

It prints the workflow and the commit that built the file, or fails if the file was changed. The same works for the
`.sbom.json` files and for `checksums.txt`.

## Scope

lazymark reads and writes Markdown files inside the notes folder you choose, and exposes the same
operations through its CLI and an MCP server. Reports about escaping the notes folder, terminal escape
sequence injection, or unsafe handling of untrusted note content are in scope.
