# Changelog

## [0.14.0](https://github.com/chanzuckerberg/camelot/compare/v0.13.1...v0.14.0) (2023-08-30)


### Features

* Added engine version to lambda resources ([e71574c](https://github.com/chanzuckerberg/camelot/commit/e71574cc56c18c22006c6eb9fc79576f675fffb9))
* Allow precise parent reference ([ece87bd](https://github.com/chanzuckerberg/camelot/commit/ece87bd6c4f1c4407a586b188a218e2c69b975a6))
* parse tf state to pull arn more consistently ([7e807ee](https://github.com/chanzuckerberg/camelot/commit/7e807eeb0b7421293d9bc2edb50452e07f56929f))


### Bug Fixes

* Camelot cannot process padded encoded CA cert ([9355de0](https://github.com/chanzuckerberg/camelot/commit/9355de0525e75669a679bc7532284c3b7033e33f))

## [0.13.1](https://github.com/chanzuckerberg/camelot/compare/v0.13.0...v0.13.1) (2023-08-28)


### Bug Fixes

* Remove CZI TFE url references ([#119](https://github.com/chanzuckerberg/camelot/issues/119)) ([e4932e2](https://github.com/chanzuckerberg/camelot/commit/e4932e2648d34d36fb81e991a715aa22795a11ab))

## [0.13.0](https://github.com/chanzuckerberg/camelot/compare/v0.12.0...v0.13.0) (2023-08-25)


### Features

* Add parallelism to TFC/TFE state scraping ([#111](https://github.com/chanzuckerberg/camelot/issues/111)) ([11464c5](https://github.com/chanzuckerberg/camelot/commit/11464c5dd3f0128d853367c0417a6cdd04a0a68e))
* Consider terraform version on a workspace as a marker for state consideration ([#113](https://github.com/chanzuckerberg/camelot/issues/113)) ([05eda5c](https://github.com/chanzuckerberg/camelot/commit/05eda5c2df963c75266d6cd1a2582681be31ad8d))

## [0.12.0](https://github.com/chanzuckerberg/camelot/compare/v0.11.0...v0.12.0) (2023-08-22)


### Features

* Add tfe/tfc scraper for AWS resource mapping ([#97](https://github.com/chanzuckerberg/camelot/issues/97)) ([267c8be](https://github.com/chanzuckerberg/camelot/commit/267c8be5a399308036c2711fb162beaf175ccd79))
* Added tfe/tfc scraping reference ([#108](https://github.com/chanzuckerberg/camelot/issues/108)) ([ca6c71c](https://github.com/chanzuckerberg/camelot/commit/ca6c71c0ef1cef874e68fe7835cd2c83d16a61d0))
* Scraping inactive TFE/TFC plans ([#109](https://github.com/chanzuckerberg/camelot/issues/109)) ([826ffdf](https://github.com/chanzuckerberg/camelot/commit/826ffdfd5a8efae77879ae9c2b4091c9f18417c4))
* Unify logging and fix display of TFE versioned resources ([#99](https://github.com/chanzuckerberg/camelot/issues/99)) ([50bab69](https://github.com/chanzuckerberg/camelot/commit/50bab697b03b8996e7db9030d1de379c02c35c55))

## [0.11.0](https://github.com/chanzuckerberg/camelot/compare/v0.10.0...v0.11.0) (2023-08-15)


### Features

* AMI tracking ([#76](https://github.com/chanzuckerberg/camelot/issues/76)) ([e78ee2a](https://github.com/chanzuckerberg/camelot/commit/e78ee2a9ea2129f0bae287c18dada674c9205282))

## [0.10.0](https://github.com/chanzuckerberg/camelot/compare/v0.9.0...v0.10.0) (2023-08-08)


### Features

* Speed up helm release discovery ([#70](https://github.com/chanzuckerberg/camelot/issues/70)) ([0d946f9](https://github.com/chanzuckerberg/camelot/commit/0d946f9e3438decfefc051713f16b0d0bc43e2e5))

## [0.9.0](https://github.com/chanzuckerberg/camelot/compare/v0.8.1...v0.9.0) (2023-08-08)


### Features

* Retrieve helm release inventory ([#59](https://github.com/chanzuckerberg/camelot/issues/59)) ([ba44f64](https://github.com/chanzuckerberg/camelot/commit/ba44f643e949f913be12066802791323805a0e73))

## [0.8.1](https://github.com/chanzuckerberg/camelot/compare/v0.8.0...v0.8.1) (2023-08-02)


### Bug Fixes

* Remove duplicate resources from the report caused by mishandled region, refactor, mocked unit tests ([#56](https://github.com/chanzuckerberg/camelot/issues/56)) ([a015bde](https://github.com/chanzuckerberg/camelot/commit/a015bde46a56d1eb2f172e09b20777080f24492c))

## [0.8.0](https://github.com/chanzuckerberg/camelot/compare/v0.7.0...v0.8.0) (2023-08-01)


### Features

* Detect abandoned repos ([#47](https://github.com/chanzuckerberg/camelot/issues/47)) ([2233b5b](https://github.com/chanzuckerberg/camelot/commit/2233b5bb957b06247a1b72e89a2ea92ce59b8b3f))


### Bug Fixes

* Improve flag management and remove chanzuckerberg references from tests ([#46](https://github.com/chanzuckerberg/camelot/issues/46)) ([a6d4ffe](https://github.com/chanzuckerberg/camelot/commit/a6d4ffe7d4116e5f5a5c978158b35499b16e43f8))

## [0.7.0](https://github.com/chanzuckerberg/camelot/compare/v0.6.0...v0.7.0) (2023-08-01)


### Features

* Keep track of the inventory of pinned modules ([#25](https://github.com/chanzuckerberg/camelot/issues/25)) ([167b380](https://github.com/chanzuckerberg/camelot/commit/167b38023be7c700ccd19773283dd5cee80bd36e))

## [0.6.0](https://github.com/chanzuckerberg/camelot/compare/v0.5.0...v0.6.0) (2023-07-14)


### Features

* Restructure scraping commands ([#22](https://github.com/chanzuckerberg/camelot/issues/22)) ([3b19ac7](https://github.com/chanzuckerberg/camelot/commit/3b19ac75b4e014114c19d1cec4122a4f1d2233f6))

## [0.5.0](https://github.com/chanzuckerberg/camelot/compare/v0.4.0...v0.5.0) (2023-07-14)


### Features

* Added installation instructions ([#12](https://github.com/chanzuckerberg/camelot/issues/12)) ([638feff](https://github.com/chanzuckerberg/camelot/commit/638feff0efc608d693e1af1919f610e70c2c2d8d))
* Combine report printing ([#21](https://github.com/chanzuckerberg/camelot/issues/21)) ([65026ff](https://github.com/chanzuckerberg/camelot/commit/65026ff66477fcb63efecb53759fa15fbae2582d))
* List out infra repos ([#13](https://github.com/chanzuckerberg/camelot/issues/13)) ([f2dfd5b](https://github.com/chanzuckerberg/camelot/commit/f2dfd5bd75c88491be4fd17d29181e199526d614))

## [0.4.0](https://github.com/chanzuckerberg/camelot/compare/v0.3.0...v0.4.0) (2023-07-13)


### Features

* Publish the release and add a version command ([#8](https://github.com/chanzuckerberg/camelot/issues/8)) ([6ea18df](https://github.com/chanzuckerberg/camelot/commit/6ea18df64ac48ad111c269c9c4a9c21966676f4f))

## [0.3.0](https://github.com/chanzuckerberg/camelot/compare/v0.2.0...v0.3.0) (2023-07-12)


### Features

* Support for scrape-all (scraping of versioned infra across all accounts) ([#9](https://github.com/chanzuckerberg/camelot/issues/9)) ([0a9610a](https://github.com/chanzuckerberg/camelot/commit/0a9610acb77a471282e3b26c977e503951c737a9))

## [0.2.0](https://github.com/chanzuckerberg/camelot/compare/v0.1.0...v0.2.0) (2023-07-12)


### Features

* Added a dependabot workflow ([#5](https://github.com/chanzuckerberg/camelot/issues/5)) ([67b4a77](https://github.com/chanzuckerberg/camelot/commit/67b4a77803e3ca3c3fad761b3a482114bb0dfc46))

## [0.1.0](https://github.com/chanzuckerberg/camelot/compare/v0.0.1...v0.1.0) (2023-07-12)


### Features

* Added Account Identity to the header, release please, and conventional commits workflows, converted to snake case ([387287d](https://github.com/chanzuckerberg/camelot/commit/387287d203620196630c489ec1ee1d7705f88634))
* Concurrent scraping ([bbb3204](https://github.com/chanzuckerberg/camelot/commit/bbb320437f6a92abc4b9e7fc8fcba8c96951c45a))
* Extract end-of-life data ([7fa43e7](https://github.com/chanzuckerberg/camelot/commit/7fa43e76ee86a4b3aaace02c9ca22c41bd5808b7))
