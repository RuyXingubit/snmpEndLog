## [1.3.1](https://github.com/RuyXingubit/snmpEndLog/compare/v1.3.0...v1.3.1) (2026-07-09)


### Bug Fixes

* reset pagination on severity filter change ([a1538bf](https://github.com/RuyXingubit/snmpEndLog/commit/a1538bf4f844d84fc3ac6490323a016194919773))

# [1.3.0](https://github.com/RuyXingubit/snmpEndLog/compare/v1.2.3...v1.3.0) (2026-07-09)


### Features

* add clear context button to AI analysis ([904e0a7](https://github.com/RuyXingubit/snmpEndLog/commit/904e0a79f826475e3aed87e189b7520f9ad96fea))

## [1.2.3](https://github.com/RuyXingubit/snmpEndLog/compare/v1.2.2...v1.2.3) (2026-07-09)


### Bug Fixes

* merge consecutive same-role messages for Gemini API ([23258bc](https://github.com/RuyXingubit/snmpEndLog/commit/23258bcebcde6d2b818ce5f0bbc3add04120ebb6))

## [1.2.2](https://github.com/RuyXingubit/snmpEndLog/compare/v1.2.1...v1.2.2) (2026-07-09)


### Bug Fixes

* use X-goog-api-key header for Gemini auth ([16f14d7](https://github.com/RuyXingubit/snmpEndLog/commit/16f14d7b2b1ed3c980608ab1806c858b5dcc380f))

## [1.2.1](https://github.com/RuyXingubit/snmpEndLog/compare/v1.2.0...v1.2.1) (2026-07-09)


### Bug Fixes

* parse Huawei syslog timestamps with year in RFC 3164 ([a2ecffc](https://github.com/RuyXingubit/snmpEndLog/commit/a2ecffcfc66017053d513cb4c52fa3c185f68c73))

# [1.2.0](https://github.com/RuyXingubit/snmpEndLog/compare/v1.1.3...v1.2.0) (2026-07-09)


### Features

* add AI log analysis with Gemini + CSV export + scroll fix ([6756e31](https://github.com/RuyXingubit/snmpEndLog/commit/6756e31a05b47b555bef24a99ce1c67c358306f9))

## [1.1.3](https://github.com/RuyXingubit/snmpEndLog/compare/v1.1.2...v1.1.3) (2026-07-09)


### Bug Fixes

* use server receive time for RFC 3164 syslog timestamps ([012beeb](https://github.com/RuyXingubit/snmpEndLog/commit/012beebfa7e2ae8ebf85c50d1272359c9d3d3b1c))

## [1.1.2](https://github.com/RuyXingubit/snmpEndLog/compare/v1.1.1...v1.1.2) (2026-07-09)


### Bug Fixes

* add 120s timeout per device poll to prevent cycle blocking ([bbe28a3](https://github.com/RuyXingubit/snmpEndLog/commit/bbe28a3822b0dff2fe046c3aeeb1b3b8860dba72))

## [1.1.1](https://github.com/RuyXingubit/snmpEndLog/compare/v1.1.0...v1.1.1) (2026-07-09)


### Bug Fixes

* update device status to 'up' when ping is reachable ([961afea](https://github.com/RuyXingubit/snmpEndLog/commit/961afea05c1733f6f0bdb50705bc2b64f6a69fe9))

# [1.1.0](https://github.com/RuyXingubit/snmpEndLog/compare/v1.0.2...v1.1.0) (2026-07-09)


### Features

* add RBAC user management (admin/viewer roles) ([6aac4fc](https://github.com/RuyXingubit/snmpEndLog/commit/6aac4fc0f99849104185605e15fdac2bf12404dc))

## [1.0.2](https://github.com/RuyXingubit/snmpEndLog/compare/v1.0.1...v1.0.2) (2026-07-09)


### Bug Fixes

* include interface alias in dashboard and improve table sorting behavior ([6188959](https://github.com/RuyXingubit/snmpEndLog/commit/618895999af6d4625e0ed183f97ec174fe2b442e))

## [1.0.1](https://github.com/RuyXingubit/snmpEndLog/compare/v1.0.0...v1.0.1) (2026-07-09)


### Bug Fixes

* hardcode network name in production compose to match Caddy ingress ([cbd939f](https://github.com/RuyXingubit/snmpEndLog/commit/cbd939fba95c1f43c9dc63f0fb165c5269ba4719))

# 1.0.0 (2026-07-09)


### Bug Fixes

* correct repository URL in package.json for semantic release ([2839610](https://github.com/RuyXingubit/snmpEndLog/commit/28396102672bf5a753033a8a7998316bd9fd0aca))


### Features

* configure production deployment with Caddy, Semantic Release and Docker Hub ([22771ec](https://github.com/RuyXingubit/snmpEndLog/commit/22771ecc713b3f14eb5aea9514ecc73641346f40))
* initial commit of snmpEndLog with BGP and PPPoE monitoring ([b54c257](https://github.com/RuyXingubit/snmpEndLog/commit/b54c25745e7abb52c5d58d6b63d15cae1dd4a847))
