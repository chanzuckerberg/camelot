# Camelot

**Please note**: If you believe you have found a security issue, _please responsibly disclose_ by contacting us at [security@chanzuckerberg.com](mailto:security@chanzuckerberg.com).

Compute Asset Management End-of-Life Object Tracking (CAMELOT) is an end-of-life tracker and versioned infrastructure scraper. It keeps track of Lambda runtimes, EKS cluster, RDS engine versions (PostgreSQL and MySQL only), terraform module pins in Github repos, and AWS resources referenced in TFC/TFE workspace states. 


## Installation

```sh
brew tap chanzuckerberg/tap
brew install camelot
```

## Usage

To scrape a specific AWS profile for versioned resources, run 

```sh
AWS_PROFILE=<PROFILE> camelot scrape aws
```

To scrape all AWS profiles specified in `~/.aws/config` (we will de-dupe the accounts automatically), run

```sh
camelot scrape aws --all
```

To scrape all github terraform repos in an org for outdated module references, use
```sh
GITHUB_TOKEN=<TOKEN> ./camelot scrape github --github-org <ORG-NAME>
```

To scrape all TFC/TFE workspaces for AWS resources, use
```sh
TFE_ADDRESS=<ADDRESS> TFE_TOKEN=<TOKEN> ./camelot scrape tfc
```

All scraping commands accept the following flags:
* `-v`: verbose mode
* `-o`: output format, could be `json`, `yaml` or `text` (`text` is default)

## Contributors
This project was initially developed by [Alex Lokshin](https://github.com/alexlokshin-czi), [Alex Biju](https://github.com/abiju-czi), [Hayden Spitzley](https://github.com/hspitzley-czi), and [Travis Fields](https://github.com/cyberious).

## Contributing
Contributions and ideas are welcome! Please don't hesitate to open an issue, join our [gitter chat room](https://gitter.im/chanzuckerberg/camelot), or send a pull request.

Go version >= 1.20 required.

This project is governed under the [Contributor Covenant](https://www.contributor-covenant.org/version/1/4/code-of-conduct) code of conduct.

## Copyright

Copyright 2017-2023, Chan Zuckerberg Initiative, LLC

For license, see [LICENSE](LICENSE).
