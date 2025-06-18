data "external_schema" "gorm" {
  program = [
    "go",
    "run",
    "-mod=mod",
    "ariga.io/atlas-provider-gorm",
    "load",
    "--path", "./internal/infrastructure/models",
    "--dialect", "mysql",
  ]
}

env "gorm" {
  src = data.external_schema.gorm.url
  url = "mysql://REDACTED_DB_USER_PROD:REDACTED_DB_PASSWORD_PROD@REDACTED_DB_HOST:3306/ms_fatidentitycat?tls=true"
  dev = "mysql://REDACTED_DB_USER_DEV:REDACTED_DB_PASSWORD_DEV@REDACTED_DB_HOST:3306/ms_fatidentitycat?tls=true"
  migration {
    // directory to store .sql and atlas.sum files
    dir = "file://migrations"
    // 添加 PlanetScale 特定配置
    lock_timeout = "5s"
  }
  format {
    migrate {
      diff = "{{ sql . \"  \" }}"
    }
  }
}
