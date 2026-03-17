# Changelog

## [0.3.0](https://github.com/kyungw00k/mnemo/compare/v0.2.0...v0.3.0) (2026-03-17)


### ⚠ BREAKING CHANGES

* Remove FTS5 full-text search, replaced by sqlite-vec vector search
    - Remove FTS5 virtual tables from migrations
    - Update TextSearch to use LIKE-based fallback
    - Remove memories_fts and notes_fts from test verification

### Features

* add CLI subcommand system with hook automation (Phase 17-19) ([d99c81c](https://github.com/kyungw00k/mnemo/commit/d99c81cbb20e4a9ea0fef20d6f38ad4db016daca))
* add markdown rendering and memory detail view to dashboard ([8e2f4d6](https://github.com/kyungw00k/mnemo/commit/8e2f4d65766b8835ecbb03106e8b1d91686404ad))
* add note detail modal to dashboard ([dec5ad9](https://github.com/kyungw00k/mnemo/commit/dec5ad9b3c3d921eab2cda6be4f37527f4b1aadc))
* add note detail view and dashboard improvements ([ed4a110](https://github.com/kyungw00k/mnemo/commit/ed4a1107050662511dc9a6fc3894695bcd978332))
* **config:** set ENABLE_GIT_CONTEXT default to true ([b43d04e](https://github.com/kyungw00k/mnemo/commit/b43d04e02a92e1323a13a7cdc5e6d74f9930908f))
* initial implementation of mnemo MCP memory server ([98ad5bf](https://github.com/kyungw00k/mnemo/commit/98ad5bfaff6d9d4b5d4cbd5d00b5e0ccbf09210e))


### Documentation

* update README to reflect latest CLI subcommand and hook architecture ([3a668d4](https://github.com/kyungw00k/mnemo/commit/3a668d4fe7b3d97b3bcb0faadc44c45b95615caa))

## [0.2.0](https://github.com/kyungw00k/mnemo/compare/v0.1.0...v0.2.0) (2026-03-16)


### Features

* initial implementation of mnemo MCP memory server ([d169fe0](https://github.com/kyungw00k/mnemo/commit/d169fe0d6ba8e61fe234c615a4de880351cf70ae))


### Documentation

* add AI tool integration guide and AGENT_INSTRUCTIONS template ([9872974](https://github.com/kyungw00k/mnemo/commit/9872974ceeb698ac2d6956c8e5084eeefd901344))
