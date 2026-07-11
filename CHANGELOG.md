## [1.8.2](https://github.com/RuyXingubit/snmpEndLog/compare/v1.8.1...v1.8.2) (2026-07-11)


### Bug Fixes

* **logs:** remover allowlist de IP do syslog no docker ([5a2bff2](https://github.com/RuyXingubit/snmpEndLog/commit/5a2bff2f0d5c0c0622540688f106da43ab0afa21))

## [1.8.1](https://github.com/RuyXingubit/snmpEndLog/compare/v1.8.0...v1.8.1) (2026-07-11)


### Bug Fixes

* **alarms:** registra template alarms.html no InitTemplates ([e6c3bba](https://github.com/RuyXingubit/snmpEndLog/commit/e6c3bbaf88bf87a115251e089018e940ba87ec97))

# [1.8.0](https://github.com/RuyXingubit/snmpEndLog/compare/v1.7.2...v1.8.0) (2026-07-11)


### Features

* **alarms:** página de histórico e nome do equipamento ([3cad0cd](https://github.com/RuyXingubit/snmpEndLog/commit/3cad0cdea84360b19bbfcc2a87fe5d5c1435ac0f))

## [1.7.2](https://github.com/RuyXingubit/snmpEndLog/compare/v1.7.1...v1.7.2) (2026-07-10)


### Bug Fixes

* **css:** tabela Logs Recentes no dashboard não trava mais ([e376e98](https://github.com/RuyXingubit/snmpEndLog/commit/e376e98debaef7d5fbcf04e730ab7f45124e0979))

## [1.7.1](https://github.com/RuyXingubit/snmpEndLog/compare/v1.7.0...v1.7.1) (2026-07-10)


### Bug Fixes

* **css:** filtros não esticam mais com scroll horizontal da tabela ([97c4b87](https://github.com/RuyXingubit/snmpEndLog/commit/97c4b87e85b001c195cfa0c353b48c486a14444f))

# [1.7.0](https://github.com/RuyXingubit/snmpEndLog/compare/v1.6.0...v1.7.0) (2026-07-10)


### Features

* período customizado com date pickers em todas as telas ([ed4dafc](https://github.com/RuyXingubit/snmpEndLog/commit/ed4dafc7084be7d0b257e2377830eb61da20c80e))

# [1.6.0](https://github.com/RuyXingubit/snmpEndLog/compare/v1.5.1...v1.6.0) (2026-07-10)


### Features

* **ai:** textarea multiline com Shift+Enter e auto-resize ([d91cf46](https://github.com/RuyXingubit/snmpEndLog/commit/d91cf4610fa79185bd03b9e97c3c82e973f76953))

## [1.5.1](https://github.com/RuyXingubit/snmpEndLog/compare/v1.5.0...v1.5.1) (2026-07-10)


### Bug Fixes

* **logs:** busca expandida para message + app_name ([5fd2b04](https://github.com/RuyXingubit/snmpEndLog/commit/5fd2b04e69047b8cf87f651aba47277b7ae87e31))

# [1.5.0](https://github.com/RuyXingubit/snmpEndLog/compare/v1.4.0...v1.5.0) (2026-07-10)


### Features

* **logs:** pesquisa por palavra-chave com highlight, envio para IA e exportação TXT ([15bcbd3](https://github.com/RuyXingubit/snmpEndLog/commit/15bcbd3de54a5974925f604c457f536a289fb129))

# [1.4.0](https://github.com/RuyXingubit/snmpEndLog/compare/v1.3.1...v1.4.0) (2026-07-09)


### Features

* add system status page + update README and prod compose ([cb6092a](https://github.com/RuyXingubit/snmpEndLog/commit/cb6092ac98d3fb251e9651ce3f3df6acf0c26e17))

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
