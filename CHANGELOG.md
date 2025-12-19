# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [1.0.0] - 2024-12-19

### Added

- Initial release
- Push container images to private ECR registries
- Push container images to public ECR (public.ecr.aws)
- Automatic ECR authentication via AWS credentials
- Support for AWS profiles and role assumption
- Multiple image tag support with template variables
- Tag templates: `{{.Version}}`, `{{.GitSHA}}`, `{{.GitBranch}}`, etc.
- Automatic repository creation option
- Dry-run mode for testing
- PostPublish hook support
