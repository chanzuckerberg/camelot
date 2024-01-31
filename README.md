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
* `-f`: report filter (this flag can be repeated multiple times), supported expressions are: `id=<ID>`, `kind=<RESOURCE_KIND>`, `parent.kind=<PARENT_KIND>`, `parent.id=<ID>`, `status=<STATUS>[,<STATUS1>]`, `version=<VERSION>`. For example: `camelot scrape tfc -f kind=tfc-workspace -f parent.kind=tfc-org -f parent.id=my-infra -f status=warning,critical -f version=0.13.5` or `camelot scrape aws --all -f kind=eks`.

Following resource types (`kind`) are supported:
* `aws` (AWS Account resources)
* `ec2` (EC2 instnace resources)
* `ami` (AWS AMI resources)
* `rds` (RDS resources)
* `vol` (Disk volume resources)
* `lambda` (AWS Lambda resources)
* `cert` (ACM Certificate resources)
* `eks` (AWS EKS resources)
* `helm` (Helm release resources)
* `github-org` (Github Organization resources)
* `github-repo` (Github Repository resources)
* `git-path` (Git Repo Relative Path resources)
* `tf-module` (Terraform Module resources)
* `tfc-org` (Terraform Cloud/Enterprise Organization resources)
* `tfc-workspace` (Terraform Cloud/Enterprise Workspace resources)
* `tfc-resource` (Terraform Cloud/Enterprise managed resources)
* `tfc-provider` (Terraform Provider resources)

## Contributors
This project was initially developed by [Alex Lokshin](https://github.com/alexlokshin-czi), [Alex Biju](https://github.com/abiju-czi), [Hayden Spitzley](https://github.com/hspitzley-czi), and [Travis Fields](https://github.com/cyberious).

## Contributing
Contributions and ideas are welcome! Please don't hesitate to open an issue, join our [gitter chat room](https://gitter.im/chanzuckerberg/camelot), or send a pull request.

Go version >= 1.21 required.

This project is governed under the [Contributor Covenant](https://www.contributor-covenant.org/version/1/4/code-of-conduct) code of conduct.

## Copyright

Copyright 2017-2023, Chan Zuckerberg Initiative, LLC

For license, see [LICENSE](LICENSE).
