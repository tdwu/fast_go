module github.com/tdwu/fast_db

go 1.19


replace github.com/ugorji/go v1.1.4 => github.com/ugorji/go/codec v0.0.0-20190204201341-e444a5086c43

require (
	github.com/golang-migrate/migrate/v4 v4.15.2
	go.uber.org/zap v1.24.0
	gorm.io/driver/mysql v1.5.0
	gorm.io/gorm v1.25.0
)

require (
	github.com/go-sql-driver/mysql v1.7.0 // indirect
	github.com/hashicorp/errwrap v1.1.0 // indirect
	github.com/hashicorp/go-multierror v1.1.1 // indirect
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/jinzhu/now v1.1.5 // indirect
	go.uber.org/atomic v1.7.0 // indirect
	go.uber.org/multierr v1.6.0 // indirect
)
