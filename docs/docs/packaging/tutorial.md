---
title: "Tutorial"
summary:  "A tutorial working through how to create a package for jq."
---

For this tutorial we're going to package up [jq](https://stedolan.github.io/jq), 
a supremely useful tool for filtering and transforming JSON.

Writing package manifests for Hermit should be fairly familiar to anyone who
has had experience with package managers like Homebrew, though it should be
significantly more straightforward assuming the package provides
cross-platform binaries for download.

This tutorial covers a fairly simple package definition, but more complex
examples exist such as [graalvm](https://github.com/cashapp/hermit-packages/blob/master/graalvm.hcl). Please refer to the
[hermit-packages](https://github.com/cashapp/hermit-packages) repository for
many more examples.

## Clone and Activate the Manifest Repository

```shell
git clone https://github.com/cashapp/hermit-packages
cd hermit-packages
. ./bin/activate-hermit
```

!!! hint
    The Hermit manifest repository is itself a Hermit environment configured to
    use itself as the source of packages. This makes testing very convenient.


## Find the Releases

The releases for jq are conveniently in a [single page](https://stedolan.github.io/jq/download/)
and by downloading one of the [links](https://github.com/stedolan/jq/releases/download/jq-1.6/jq-linux64)
we can see that they're directly downloadable binaries. Convenient.

## Create a Basic Manifest

Create an empty `jq.hcl` file in the `hermit-packages` directory. The first
thing you'll want is a description, for which typically just copy the project
description from their site or GitHub repository:

```terraform
description = "jq is like sed for JSON data - you can use it to slice and filter and map and transform structured data with the same ease that sed, awk, grep and friends let you play with text."
```

!!! hint
    The `hermit` CLI includes a best-effort command to create a stub manifest.

    ```shell
    hermit manifest create https://github.com/stedolan/jq/releases/download/jq-1.6/jq-linux64`
    ```

    Currently there a few limitations:

    1. It only works for binaries in GitHub releases.
    2. The download link must include `${os}` and either `${arch}` or `${xarch}`.

    Hopefully these limitations will be removed over time.

## Add a Version

[`version`](../schema/version) blocks tell Hermit what versions of a package
are available for download and are specified as blocks. We'll start with an
empty one for `jq-1.6`:

```terraform
version "1.6" {}
```

## Add Download URLs for Each OS

Looking at the links we can see that there are downloads for Linux and OSX:

1. https://github.com/stedolan/jq/releases/download/jq-1.6/jq-linux64
2. https://github.com/stedolan/jq/releases/download/jq-1.6/jq-osx-amd64

So we'll add blocks for the respective operating systems ([`linux`](../schema/linux) and [`darwin`](../schema/darwin)) and populate the
`source` attribute, which tells Hermit where to download packages from:

```terraform
version "1.6" {
  linux {
    source = "https://github.com/stedolan/jq/releases/download/jq-1.6/jq-linux64"
  }

  darwin {
    source = "https://github.com/stedolan/jq/releases/download/jq-1.6/jq-osx-amd64"
  }
}
```

## DRY our URLs

The raw URLs will work fine, but if we add more versions later it would be
nice not to have to duplicate this configuration. To do that we can pull the
OS blocks out to the top level and use Hermit's
[variable interpolation](../reference/#variable-interpolation)
support to substitute the `${version}` variable:

```terraform
description = "jq is like sed for JSON data - you can use it to slice and filter and map and transform structured data with the same ease that sed, awk, grep and friends let you play with text."
linux {
  source = "https://github.com/stedolan/jq/releases/download/jq-${version}/jq-linux64"
}

darwin {
  source = "https://github.com/stedolan/jq/releases/download/jq-${version}/jq-osx-amd64"
}

version "1.6" {}
```

When selecting a version/channel, Hermit will look for sources in the matching
block and fallback to the top-level.

## Specifying the Binaries

At this point Hermit knows where to download our binaries from, but not what
to do with them. The binaries will also have different names(`jq-linux64` and
`jq-osx-amd64`) depending on which OS we're on. We need to rename this
binaries to the canonical `jq`. To solve this we're going to need to use a
[trigger](../schema/on) to apply an action when unpacking, specifically the 
[rename](../schema/rename) action.

```terraform
linux {
  source = "https://github.com/stedolan/jq/releases/download/jq-${version}/jq-linux64"
  on unpack {
    rename { from = "${root}/jq-linux64" to = "${root}/jq" }
  }
}

darwin {
  source = "https://github.com/stedolan/jq/releases/download/jq-${version}/jq-osx-amd64"
  on unpack {
    rename { from = "${root}/jq-osx-amd64" to = "${root}/jq" }
  }
}
```

And tell Hermit which binaries to link when installed:

```terraform
binaries = ["jq"]
```

{{< hint ok >}}
The `binaries` attribute supports globs, which will be expanded at unpack time.
{{< /hint >}}

## Adding SHA256 Sums

Populating sha256 checksums for each of your package downloads allows Hermit to validate the integrity after downloading. (provided in the `sha256sums` field)

You can use Hermit to automatically generate them for you:

```shell
hermit add-digests jq.hcl
```

## Testing the Package

Hermit packages can include a `testing` attribute which is a command to run to
test whether the package is functioning. This will typically just be
something like:

```terraform
test = "jq --version"
```

The Hermit packages CI will run these tests periodically.

To test your package run:

```shell
$ hermit test jq --trace
debug:jq-1.6:exec: /Users/user/Library/Caches/hermit/pkg/jq-1.6/jq --version
debug: jq-1.6
```

## The End Result

And we're done.

```terraform
description = "jq is like sed for JSON data - you can use it to slice and filter and map and transform structured data with the same ease that sed, awk, grep and friends let you play with text."
binaries = ["jq"]
test = "jq --version"

linux {
  source = "https://github.com/stedolan/jq/releases/download/jq-${version}/jq-linux64"
  on unpack {
    rename { from = "${root}/jq-linux64" to = "${root}/jq" }
  }
}

darwin {
  source = "https://github.com/stedolan/jq/releases/download/jq-${version}/jq-osx-amd64"
  on unpack {
    rename { from = "${root}/jq-osx-amd64" to = "${root}/jq" }
  }
}

version "1.6" {}
```

## Local Manual Testing

As mentioned above, `hermit-packages` is also a Hermit environment. Now we
have our manifest we can attempt to install it with:

```shell
$ hermit install jq
$ jq --version
jq-1.6
```

## Distribute the Package

At this point you can (and should!) contribute the package back to the
community via a [PR](https://github.com/cashapp/hermit-packages/pulls).
